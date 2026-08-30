// Package connectors defines the minimal shape every connector in this
// codebase must satisfy (docs/05-integrations/connector-catalog.md: owner,
// data classification, health check, contract tests, target allowlist) and a
// reusable contract-test harness that proves it. Concrete connectors
// (enrichment, ticketing, and any real provider added at M5) implement
// Connector and are exercised by RunContractTests — this package holds no
// connector logic of its own.
package connectors

import "context"

// Manifest mirrors the required fields of schemas/connector.schema.json.
// It is hand-written to match the checked-in example under examples/, not
// generated — see adr/0010-mock-enrichment-and-ticketing-connectors.md for
// why (the same disclosed hand-sync as genapi vs. openapi.yaml).
type Manifest struct {
	ID                 string
	Owner              string
	Provider           string
	Intents            []string
	ErrorClasses       []string
	DryRun             bool
	Idempotent         bool
	DataClassification string
	TargetAllowlist    []string
}

// ErrorClass is one of a manifest's declared error_classes — connectors must
// classify every failure as one of these rather than leaking raw errors.
type ErrorClass string

const (
	ErrClassValidation    ErrorClass = "validation"
	ErrClassNotFound      ErrorClass = "not_found"
	ErrClassPolicyDenied  ErrorClass = "policy_denied"
	ErrClassProviderError ErrorClass = "provider_error"
)

// ConnectorError is the typed error every Connector.Invoke must return on
// failure, so callers (and the contract-test harness) can assert on the
// class instead of matching error strings.
type ConnectorError struct {
	Class   ErrorClass
	Message string
}

func (e *ConnectorError) Error() string { return e.Message }

// Invocation is one call into a connector: the intent being requested, the
// target the call would affect (checked against the manifest's
// target_allowlist before any work happens), the parameters, and an
// idempotency key the connector must use to make repeats safe.
type Invocation struct {
	Intent         string
	Target         string
	Params         map[string]any
	IdempotencyKey string
}

// Result is a successful Invoke outcome. Effect is true only when the call
// actually changed state (used by the harness to check dry-run and
// idempotent-repeat behavior); a cache hit or a repeat of an idempotent call
// should report Effect: false.
type Result struct {
	Effect bool
	Data   map[string]any
}

// Connector is the minimal interface a contract-tested connector
// implements. HealthCheck must never mutate state (contract requirement:
// schemas/connector.schema.json's health_check.read_only is always true).
type Connector interface {
	Manifest() Manifest
	HealthCheck(ctx context.Context) error
	Invoke(ctx context.Context, inv Invocation) (Result, error)
}
