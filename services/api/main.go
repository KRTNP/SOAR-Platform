// Command api is the SOAR M1 HTTP service: event ingestion and case
// read/write, per docs/10-roadmap/milestones.md's "synthetic alert reaches a
// case" gate. Deliberately out of scope at this milestone: real auth
// (TenantStubMiddleware fixes a single tenant), NATS/Temporal (ingestion runs
// synchronously in one DB transaction — see events.Handler.Ingest), and any
// playbook/approval/connector logic. See docs/10-roadmap/milestones.md for
// the milestone sequence this fits into.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"soar-platform/services/api/actions"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/config"
	"soar-platform/services/api/connectors"
	"soar-platform/services/api/events"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/network"
	"soar-platform/services/api/opnsense"
	"soar-platform/services/api/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := store.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	netConn, err := buildNetworkConnector()
	if err != nil {
		return err
	}
	logger.Info("network connector selected", "provider", netConn.Version)

	router := newRouter(logger, cfg, pool, netConn)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", cfg.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("shutting down")
		return srv.Shutdown(shutdownCtx)
	}
}

// buildNetworkConnector picks which connector backs block_network_indicator:
// the mock services/api/network unless OPNSENSE_ENABLE_REAL=true, in which
// case the real services/api/opnsense connector against the lab VM is
// required to configure cleanly or startup fails fast (a bad OPNsense
// config should never silently fall back to the mock — see
// adr/0013-real-opnsense-lab-connector.md).
func buildNetworkConnector() (actions.NetworkConnector, error) {
	if !opnsense.Enabled() {
		return actions.NetworkConnector{
			New:     func(tx pgx.Tx) connectors.Connector { return network.New(tx) },
			Release: func(ctx context.Context, tx pgx.Tx, actionID, _ string) error { return network.Release(ctx, tx, actionID) },
			Target:  "network_blocks",
			Version: "connector-mock-network-block@1.0.0",
		}, nil
	}

	ocfg, err := opnsense.LoadConfig()
	if err != nil {
		return actions.NetworkConnector{}, fmt.Errorf("OPNSENSE_ENABLE_REAL=true but config is invalid: %w", err)
	}
	conn := opnsense.NewWithConfig(ocfg)
	return actions.NetworkConnector{
		New:     func(tx pgx.Tx) connectors.Connector { return conn },
		Release: func(ctx context.Context, tx pgx.Tx, actionID, targetValue string) error { return conn.ReleaseTemp(ctx, targetValue) },
		Target:  "SOAR_TEMP_BLOCK",
		Version: "connector-opnsense-lab@1.0.0",
	}, nil
}

func newRouter(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, netConn actions.NetworkConnector) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(httpx.RecoveryMiddleware(logger), httpx.RequestIDMiddleware(), httpx.TenantStubMiddleware(cfg.DefaultTenantID), httpx.AccessLogMiddleware(logger))

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/readyz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})

	eventsH := events.NewHandler(pool)
	casesH := cases.NewHandler(pool)
	actionsH := actions.NewHandler(pool, netConn)
	evidenceH := evidence.NewHandler(pool)
	auditH := audit.NewHandler(pool)

	v1 := r.Group("/v1")
	{
		v1.POST("/events", eventsH.Ingest)
		v1.GET("/events", eventsH.List)

		v1.POST("/cases", casesH.Create)
		v1.GET("/cases", casesH.List)
		v1.GET("/cases/:caseId", casesH.Get)
		v1.PATCH("/cases/:caseId", casesH.Update)
		v1.GET("/cases/:caseId/timeline", casesH.Timeline)
		v1.GET("/cases/:caseId/evidence", casesH.Evidence)
		v1.POST("/cases/:caseId/export", casesH.Export)

		v1.POST("/actions/preview", actionsH.Preview)
		v1.GET("/actions/:actionId", actionsH.Get)
		v1.POST("/actions/:actionId/execute", actionsH.Execute)

		v1.GET("/approvals", actionsH.ListApprovals)
		v1.POST("/approvals/:approvalId/approve", actionsH.Approve)
		v1.POST("/approvals/:approvalId/reject", actionsH.Reject)

		v1.POST("/evidence/manifests", evidenceH.CreateManifest)
		v1.GET("/evidence/:evidenceId", evidenceH.Get)

		v1.GET("/audit", auditH.List)
	}

	return r
}
