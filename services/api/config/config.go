// Package config loads process configuration from the environment.
package config

import (
	"cmp"
	"fmt"
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

	return Config{
		Addr:                     cmp.Or(os.Getenv("ADDR"), ":8080"),
		DatabaseURL:              dbURL,
		DefaultTenantID:          cmp.Or(os.Getenv("DEFAULT_TENANT_ID"), "lab"),
		SchedulerEnabled:         schedulerEnabled,
		SchedulerIntervalSeconds: schedulerIntervalSeconds,
	}, nil
}
