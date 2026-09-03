// Command api is the SOAR M1 HTTP service: event ingestion and case
// read/write, per docs/10-roadmap/milestones.md's "synthetic alert reaches a
// case" gate. Deliberately out of scope at this milestone: real auth
// (httpx.TenantKeyMiddleware selects a tenant by API key, adr/0022, but is
// not authentication — no identity, no RBAC), NATS/Temporal (ingestion runs
// synchronously in one DB transaction — see events.Handler.Ingest), and any
// playbook/approval/connector logic. See docs/10-roadmap/milestones.md for
// the milestone sequence this fits into.
package main

import (
	"context"
	"fmt"
	"io"
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
	"soar-platform/services/api/allowlist"
	"soar-platform/services/api/audit"
	"soar-platform/services/api/cases"
	"soar-platform/services/api/config"
	"soar-platform/services/api/connectors"
	"soar-platform/services/api/criticaltargets"
	"soar-platform/services/api/events"
	"soar-platform/services/api/evidence"
	"soar-platform/services/api/httpx"
	"soar-platform/services/api/metrics"
	"soar-platform/services/api/network"
	"soar-platform/services/api/opnsense"
	"soar-platform/services/api/scheduler"
	"soar-platform/services/api/searchindex"
	"soar-platform/services/api/store"
	"soar-platform/services/api/telemetry"
	"soar-platform/services/api/tenants"
	"soar-platform/services/api/wazuhpoll"
)

func main() {
	logger := newLogger()
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

// newLogger writes structured JSON to stdout, same as always, plus — when
// LOG_FILE_PATH is set — to that file too, so Promtail (adr/0020) can tail a
// plain file on a shared volume instead of needing access to the container
// runtime's own API socket. Unset (the default) is byte-for-byte today's
// stdout-only behavior; a file that fails to open falls back to stdout-only
// rather than failing startup over a log-shipping convenience.
func newLogger() *slog.Logger {
	w := io.Writer(os.Stdout)
	if path := os.Getenv("LOG_FILE_PATH"); path != "" {
		if f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
			w = io.MultiWriter(os.Stdout, f)
		}
	}
	return slog.New(slog.NewJSONHandler(w, nil))
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

	rec := telemetry.NewRecorder()

	schedulerDone := make(chan struct{})
	if cfg.SchedulerEnabled {
		logger.Info("scheduler enabled", "interval_seconds", cfg.SchedulerIntervalSeconds)
		go func() {
			defer close(schedulerDone)
			scheduler.Run(ctx, pool, netConn, time.Duration(cfg.SchedulerIntervalSeconds)*time.Second, logger, rec)
		}()
	} else {
		close(schedulerDone)
	}

	wazuhPollDone := make(chan struct{})
	if wazuhpoll.Enabled() {
		wcfg, err := wazuhpoll.LoadConfig()
		if err != nil {
			return fmt.Errorf("WAZUH_POLL_ENABLE_REAL=true but config is invalid: %w", err)
		}
		logger.Info("wazuh poller enabled", "indexer", wcfg.IndexerBaseURL, "interval_seconds", wcfg.PollIntervalSeconds)
		go func() {
			defer close(wazuhPollDone)
			wazuhpoll.Run(ctx, pool, wcfg, cfg.WazuhIngestBaseURL, logger, rec)
		}()
	} else {
		close(wazuhPollDone)
	}

	var searchH *searchindex.Handler
	searchIndexDone := make(chan struct{})
	if searchindex.Enabled() {
		scfg, err := searchindex.LoadConfig()
		if err != nil {
			return fmt.Errorf("OPENSEARCH_ENABLE_REAL=true but config is invalid: %w", err)
		}
		logger.Info("search indexer enabled", "opensearch", scfg.BaseURL, "interval_seconds", scfg.PollIntervalSeconds)
		searchH = searchindex.NewHandler(scfg)
		go func() {
			defer close(searchIndexDone)
			searchindex.Run(ctx, pool, scfg, logger)
		}()
	} else {
		close(searchIndexDone)
	}

	router := newRouter(logger, cfg, pool, netConn, rec, searchH)

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

	var result error
	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			result = err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		logger.Info("shutting down")
		result = srv.Shutdown(shutdownCtx)
	}

	// Cancel explicitly (not just via the deferred stop()) so the scheduler
	// and wazuh-poller goroutines observe ctx.Done() and exit before
	// pool.Close() (deferred above, runs before the also-deferred stop() on
	// the way out) fires.
	stop()
	<-schedulerDone
	<-wazuhPollDone
	<-searchIndexDone
	return result
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
			New: func(tx pgx.Tx) connectors.Connector { return network.New(tx) },
			Release: func(ctx context.Context, tx pgx.Tx, actionID, _ string) error {
				return network.Release(ctx, tx, actionID)
			},
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
		New: func(tx pgx.Tx) connectors.Connector { return conn },
		Release: func(ctx context.Context, tx pgx.Tx, actionID, targetValue string) error {
			return conn.ReleaseTemp(ctx, targetValue)
		},
		Target:  "SOAR_TEMP_BLOCK",
		Version: "connector-opnsense-lab@1.0.0",
		Preview: func(ctx context.Context, target string) (bool, error) {
			r, err := conn.Preview(ctx, target)
			return r.AlreadyBlocked, err
		},
	}, nil
}

func newRouter(logger *slog.Logger, cfg config.Config, pool *pgxpool.Pool, netConn actions.NetworkConnector, rec *telemetry.Recorder, searchH *searchindex.Handler) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	resolveTenant := func(ctx context.Context, key string) (string, error) { return tenants.Resolve(ctx, pool, key) }
	r.Use(httpx.RecoveryMiddleware(logger), httpx.RequestIDMiddleware(), httpx.TenantKeyMiddleware(cfg.DefaultTenantID, resolveTenant), httpx.AccessLogMiddleware(logger), rec.Middleware())

	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/readyz", func(c *gin.Context) {
		if err := pool.Ping(c.Request.Context()); err != nil {
			c.Status(http.StatusServiceUnavailable)
			return
		}
		c.Status(http.StatusOK)
	})
	r.GET("/metrics", gin.WrapH(rec.Handler()))

	eventsH := events.NewHandler(pool, rec)
	casesH := cases.NewHandler(pool, rec)
	actionsH := actions.NewHandler(pool, netConn, rec)
	evidenceH := evidence.NewHandler(pool)
	auditH := audit.NewHandler(pool)
	metricsH := metrics.NewHandler(pool, netConn.Target, netConn.Version)
	allowlistH := allowlist.NewHandler(pool)
	criticalTargetsH := criticaltargets.NewHandler(pool)
	tenantsH := tenants.NewHandler(pool)

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
		v1.POST("/actions/:actionId/release", actionsH.Release)

		v1.GET("/approvals", actionsH.ListApprovals)
		v1.POST("/approvals/:approvalId/approve", actionsH.Approve)
		v1.POST("/approvals/:approvalId/reject", actionsH.Reject)

		v1.POST("/evidence/manifests", evidenceH.CreateManifest)
		v1.GET("/evidence/:evidenceId", evidenceH.Get)

		v1.GET("/audit", auditH.List)

		v1.GET("/metrics", metricsH.Get)

		v1.GET("/allowlist", allowlistH.List)
		v1.POST("/allowlist", allowlistH.Add)

		v1.GET("/critical-targets", criticalTargetsH.List)
		v1.POST("/critical-targets", criticalTargetsH.Add)

		v1.GET("/tenants", tenantsH.List)
		v1.POST("/tenants", tenantsH.Create)

		if searchH != nil {
			v1.GET("/search", searchH.Search)
		}
	}

	return r
}
