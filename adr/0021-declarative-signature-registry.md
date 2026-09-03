# ADR-0021: Declarative signature registry (generalizes the SSH brute-force auto-trigger)

## Status

Accepted — implemented; the SSH brute-force signature is registry-driven,
proven behavior-identical by the full existing integration suite. A second
real signature was not added — see Rationale.

## Decision

Generalize `adr/0018`'s single hardcoded correlation/auto-trigger signature
(repeated `authentication`-class events sharing an `ipv4` observable) into a
small declarative registry, narrowing — not reopening — that ADR's finding
that a generic playbook interpreter is out of scope (`trigger.condition` and
`steps[].purpose` are free-text prose across all 16 `playbooks/*.yaml`
files).

1. **`schemas/playbook.schema.json`** gains an optional top-level
   `automation` object: `{signature_id, match: {event_class,
   observable_type?}, threshold, window_seconds, action_intent}`. Optional
   and additive — a playbook without it stays pure documentation, exactly as
   before. `validation/validate-playbooks.sh` gained a matching structural
   check (required sub-keys, `match.event_class` present) when the block is
   present, mirroring the schema by hand the same way that script already
   mirrors the rest of `playbook.schema.json`'s required fields.
2. **`playbooks/ssh-bruteforce-response.yaml`** is the first (and, per the
   scan below, only) playbook to add this block:
   `signature_id: ssh-bruteforce, match: {event_class: authentication,
   observable_type: ipv4}, threshold: 5, window_seconds: 60, action_intent:
   block_network_indicator` — the exact values `events/handler.go` already
   hardcoded.
3. **New `services/api/signatures` package**: a `Signature` struct mirroring
   that schema block, and `Registry []Signature` — a hand-synced Go literal,
   not a runtime YAML parser (adr/0018 already ruled that out; `playbooks/`
   also lives outside `services/api/go.mod`'s module tree). This is the same
   "hand-copied mirror of a schema, documented as such" posture
   `services/api/actions/policy.go` already uses for
   `schemas/policy.schema.json`.
4. **`events/handler.go`**'s correlation/auto-trigger block now loops
   `signatures.MatchEventClass` instead of the two hardcoded constants
   (`correlationWindowSeconds`, `autoBlockThreshold`) and the hardcoded
   `"authentication"`+`ipv4` check — SSH brute-force is now the registry's
   one entry, not special-cased Go code. `cases/handler.go`'s close-gate
   (adr/0019) similarly now derives its threshold from
   `signatures.MatchEventClass` (parsed off the case's own
   `correlation_key` prefix) instead of a second hardcoded `= 5` local
   constant, falling back to `5` only if no signature matches (defensive,
   should not happen since `events.correlationKeyFor` always writes
   `eventClass:value`).

## Rationale: why no second signature was added

The approved plan for this phase named `credential-stuffing.yaml` and
`suspicious-ioc.yaml` as candidates, to be read for real numeric conditions
rather than assumed. Both turned out to be prose (`"repeated failures
followed by success or distributed failures"`, `"alert contains an
observable of type ipv4, ipv6, domain, url or hash"` — no threshold, no
window). A full scan of all 15 non-SSHBF `playbooks/*.yaml` files'
`trigger.condition` values confirmed **none** have a literal numeric
threshold and window the way SSHBF's `"5 failed SSH logins within 1
minute"` does — every other trigger reads like `"anomalous sign-in..."`,
`"multiple corroborated signals exceed approved risk threshold"`, or similar
qualitative language.

Inventing a threshold/window for one of these to populate a second registry
entry would fabricate specificity the source playbook document does not
contain — exactly the kind of unfounded precision this repo's
documentation-first practice avoids elsewhere (e.g. `adr/0017`'s disclosed
heuristic mappings are labeled as heuristics, not treated as authoritative).
The registry stays a single-entry structure until a playbook is authored
(or revised) with an actual concrete number to mirror.

## Consequences

- `services/api/signatures` is real, tested infrastructure with exactly one
  consumer today — a deliberate, disclosed trade-off: it is still a net
  improvement over the previous state (two hardcoded constants plus an
  inline `"authentication"`/`"ipv4"` check duplicated across
  `events/handler.go` and `cases/handler.go`), but it is not yet proof that
  the abstraction generalizes to a second real signature.
- Adding a genuinely new signature later means: read the new playbook's
  actual prose for a real number (do not invent one), add its `automation`
  block, add one `signatures.Registry` entry, and add integration test
  coverage mirroring `TestIngestion_FiveAuthFailuresFromOneIPAutoRequestsBlock`
  — no `events/handler.go` or `cases/handler.go` changes should be needed
  for a signature whose `action_intent` is already `block_network_indicator`.
- An `action_intent` other than `block_network_indicator` has no request
  path yet (`actions.RequestBlock` is the only response-action constructor
  wired to this trigger) — the field is recorded for documentation/future
  dispatch, not yet read to select between multiple action constructors.
- `docs/04-playbooks/playbook-catalog.md` now notes PB-SSHBF-001 as the only
  registry-backed playbook and records that the other 15 were checked and
  found not to qualify, so a future contributor does not re-run the same
  scan.
