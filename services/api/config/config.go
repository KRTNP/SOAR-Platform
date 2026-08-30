// Package config loads process configuration from the environment.
package config

import (
	"cmp"
	"fmt"
	"os"
)

type Config struct {
	Addr        string
	DatabaseURL string
	// DefaultTenantID is the lab-only auth stub: every request is treated as
	// belonging to this tenant until a real OIDC/Keycloak integration lands.
	DefaultTenantID string
}

func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return Config{
		Addr:            cmp.Or(os.Getenv("ADDR"), ":8080"),
		DatabaseURL:     dbURL,
		DefaultTenantID: cmp.Or(os.Getenv("DEFAULT_TENANT_ID"), "lab"),
	}, nil
}
