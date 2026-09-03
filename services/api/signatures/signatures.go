// Package signatures generalizes the one correlation/auto-trigger signature
// adr/0018 hand-coded (repeated authentication-class events sharing a
// source IP) into a small declarative registry, so a playbook that opts in
// with a structured `automation:` block (schemas/playbook.schema.json) can
// drive events.Handler.Ingest's correlation/auto-trigger logic as data
// instead of a second copy-pasted set of Go constants. See
// adr/0021-declarative-signature-registry.md for why this stops short of a
// generic playbook interpreter: adr/0018 already found that
// `trigger.condition`/`steps[].purpose` are free-text prose across all 16
// `playbooks/*.yaml` files, and a scan for this pass confirmed the same is
// true of every candidate `automation` block except this one — SSH
// brute-force is still the only playbook with a literal numeric threshold
// and window in its trigger condition.
package signatures

// Signature mirrors one playbook's optional `automation` object
// (schemas/playbook.schema.json). It intentionally does not parse
// playbooks/*.yaml at runtime — adr/0018 already ruled out a generic
// playbook interpreter, and these YAML files live outside this Go module's
// file tree (services/api/go.mod is the module root) — so Registry below is
// a hand-synced mirror, the same posture services/api/actions/policy.go
// already uses for schemas/policy.schema.json. Update both together when a
// playbook's automation block changes.
type Signature struct {
	// ID identifies the signature for logging/audit purposes.
	ID string
	// EventClass and ObservableType select which normalized events this
	// signature correlates — ObservableType empty means "any observable
	// carries the correlation value" (not used by any signature yet).
	EventClass     string
	ObservableType string
	// Threshold is the repeat_count at which the auto-trigger fires.
	Threshold int
	// WindowSeconds is the correlation window stored on the case row.
	WindowSeconds int
	// ActionIntent is passed to the response-authority policy
	// (actions.EvaluatePolicy) for the auto-triggered action's approval
	// reason; only "block_network_indicator" has a request path today
	// (actions.RequestBlock).
	ActionIntent string
}

// Registry is this package's mirror of every playbook that has opted into
// automation via a structured `automation` block — currently just
// playbooks/ssh-bruteforce-response.yaml's, matching adr/0018's
// correlationWindowSeconds=60/autoBlockThreshold=5 constants exactly (this
// registry replaces those constants, not changes their values).
var Registry = []Signature{
	{
		ID:             "ssh-bruteforce",
		EventClass:     "authentication",
		ObservableType: "ipv4",
		Threshold:      5,
		WindowSeconds:  60,
		ActionIntent:   "block_network_indicator",
	},
}

// MatchEventClass returns the first registry entry whose EventClass matches
// eventClass, or nil if none do. Every signature so far keys on event class
// alone (ObservableType only decides which observable to correlate on, not
// which signature applies) — a second signature needing to disambiguate by
// more than event class would extend this lookup, not this shape.
func MatchEventClass(eventClass string) *Signature {
	for i := range Registry {
		if Registry[i].EventClass == eventClass {
			return &Registry[i]
		}
	}
	return nil
}
