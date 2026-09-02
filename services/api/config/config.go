// Package config loads process configuration from the environment.
package config

import (
	"cmp"
	"fmt"
	"net"
	"os"
	"strconv"
)

type Config struct {
	Addr        string
	DatabaseURL string
	// DefaultTenantID is the lab-only auth stub: every request is treated as
	// belonging to this tenant until a real OIDC/Keycloak integration lands.
	DefaultTenantID string
	// SchedulerEnabled/SchedulerIntervalSeconds drive the M6 TTL-rollback
	// scheduler (services/api/scheduler) — see adr/0014.
	SchedulerEnabled         bool
	SchedulerIntervalSeconds int
	// WazuhIngestBaseURL is where services/api/wazuhpoll forwards polled
	// alerts (POST {url}/v1/events) — this process's own loopback address,
	// derived from Addr, never a separate env var (see adr/0017).
	WazuhIngestBaseURL string
}

func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	schedulerEnabled := true
	if v := os.Getenv("SCHEDULER_ENABLED"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("SCHEDULER_ENABLED must be a bool: %w", err)
		}
		schedulerEnabled = b
	}

	schedulerIntervalSeconds := 30
	if v := os.Getenv("SCHEDULER_INTERVAL_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("SCHEDULER_INTERVAL_SECONDS must be a positive integer: %q", v)
		}
		schedulerIntervalSeconds = n
	}

	addr := cmp.Or(os.Getenv("ADDR"), ":8080")
	return Config{
		Addr:                     addr,
		DatabaseURL:              dbURL,
		DefaultTenantID:          cmp.Or(os.Getenv("DEFAULT_TENANT_ID"), "lab"),
		SchedulerEnabled:         schedulerEnabled,
		SchedulerIntervalSeconds: schedulerIntervalSeconds,
		WazuhIngestBaseURL:       "http://" + loopbackHostPort(addr),
	}, nil
}

// loopbackHostPort turns a listen address like ":8080" or "0.0.0.0:8080"
// into "127.0.0.1:8080" — the host this same process should dial to reach
// its own POST /v1/events, regardless of which interface it binds to listen.
func loopbackHostPort(addr string) string {
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "8080"
	}
	return "127.0.0.1:" + port
}
