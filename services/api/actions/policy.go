package actions

// DefaultPolicyVersion identifies the rule set EvaluatePolicy applies below.
// RequestBlock stamps every Action it creates with this value (the existing
// PolicyVersion column), so a later multi-policy or per-tenant scheme has
// somewhere to diverge from without a migration.
const DefaultPolicyVersion = "policy-1.0"

// policyRule is this package's in-code mirror of one entry in
// schemas/policy.schema.json's "rules" array. It intentionally does not load
// examples/policy.default.json at runtime: that file lives outside this Go
// module's embeddable file tree (services/api/go.mod is the module root,
// examples/ is a repo-root sibling), so the two are kept in sync by hand —
// see adr/0016-response-authority-policy.md for that decision and its
// consequences, and update both together when a rule changes.
type policyRule struct {
	decision      string
	approvalRoles []string
}

// defaultPolicy is the "response authority" table: which role(s) a given
// intent's approval should be assigned to. It is evaluated for its
// approvalRoles only — every intent still reaches its existing hardcoded
// gate (only block_network_indicator has one, in RequestBlock); no intent
// here is newly allowed or newly denied by this table. See the ADR for why:
// there is no identity/authz system yet (adr/0005 selects Keycloak but it
// is not built), so approvalRoles is descriptive metadata recorded on the
// audit trail, not an enforced authorization check.
var defaultPolicy = map[string]policyRule{
	"notify":                  {decision: "allow"},
	"query_siem":              {decision: "allow"},
	"create_ticket":           {decision: "allow"},
	"enrich_observable":       {decision: "allow"},
	"collect_artifact":        {decision: "require_approval", approvalRoles: []string{"analyst"}},
	"quarantine_email":        {decision: "require_approval", approvalRoles: []string{"incident-commander"}},
	"revoke_session":          {decision: "require_approval", approvalRoles: []string{"incident-commander"}},
	"block_network_indicator": {decision: "require_approval", approvalRoles: []string{"incident-commander"}},
	"isolate_host":            {decision: "require_approval", approvalRoles: []string{"incident-commander", "service-owner"}},
	"disable_account":         {decision: "require_approval", approvalRoles: []string{"incident-commander", "service-owner"}},
}

// EvaluatePolicy resolves the default policy's decision and approval_roles
// for intent. risk is accepted because schemas/policy.schema.json's rules
// are (intent, risk) conditions in general, but the current fixed table
// only branches on intent — no rule here yet keys on risk.
//
// An intent absent from the table (not one of action.schema.json's known
// intents, or a future addition made to that schema before this table)
// falls back to the same posture ADR-0011 already established for every
// intent: require approval from whoever the hardcoded actor is today.
func EvaluatePolicy(intent, risk string) (decision string, approvalRoles []string) {
	rule, ok := defaultPolicy[intent]
	if !ok {
		return "require_approval", []string{"incident-commander"}
	}
	return rule.decision, rule.approvalRoles
}
