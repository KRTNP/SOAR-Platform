# Tabletop script: M5 lab connector approval walkthrough

This is a runnable script for the tabletop half of `docs/10-roadmap/milestones.md`'s
M5 gate ("sandbox + tabletop approval"). It is scoped to what M1–M5 actually
implement today, not the full target-state flow in
[`lab-scenario-ssh-bruteforce.md`](lab-scenario-ssh-bruteforce.md) (that scenario
describes auto-correlation, scheduled TTL rollback and escalation retries that do
not exist yet — see the gap list below). It complements the general principles in
[`tabletop-exercises.md`](tabletop-exercises.md) with an exact, copy-pasteable
sequence against this repository's real HTTP API and the real lab OPNsense VM
built for [ADR-0013](../../adr/0013-real-opnsense-lab-connector.md).

Running this exercise with real people in the roles below, and recording its
outputs (see "Outputs to capture"), is what actually closes M5's gate — nobody
should flip `milestones.md`'s M5 row to "Done" until this has happened at least
once.

## What this exercise proves, and what it does not

Proves:

- An analyst-initiated `block_network_indicator` request cannot reach the real
  OPNsense VM without a recorded approval bound to that specific action.
- The approved action actually changes the real firewall's `SOAR_TEMP_BLOCK`
  alias, and the connector verifies the post-condition rather than trusting
  OPNsense's own status field (`adr/0013`'s postcondition-mismatch discovery).
- Every step (request, approve, execute) lands in the append-only audit log with
  a target hash, not a raw IP.

Does **not** prove, and should not be reported as proven — these are known,
disclosed gaps to discuss, not implementation bugs to fix during the exercise:

- **No RBAC or authentication exists yet** (`adr/0011`). Every request is
  processed under a single stub tenant, and every audited actor is a fixed
  string (`"analyst"`, `"incident-commander"`, `"system"`) regardless of who
  actually clicked. Separation of duties in this exercise is enforced by the
  participants' discipline, not by the system — say out loud whose "turn" it
  is rather than relying on the API to stop the wrong person.
- **No scheduler drives TTL expiry or rollback.** `actions.Release` (the
  manual early-rollback function) exists and is unit-tested, but has **no
  HTTP endpoint** — `api/openapi.yaml` declares no `release` route today. A
  live rollback in this exercise means removing the IP from `SOAR_TEMP_BLOCK`
  directly via the OPNsense UI/API, not through this system. Record this as a
  gap (see below), don't paper over it.
- **No automatic Wazuh-to-SOAR ingestion.** M1 only ever proved a *synthetic*
  alert reaching a case. This script POSTs a hand-built event to
  `/v1/events` that mimics a Wazuh alert shape
  (`examples/normalized-event.json`), standing in for a real-time forwarder
  that does not exist yet.
- **Most of `docs/06-security/safe-response-controls.md`'s ten pre-execution
  gates are not mechanically enforced.** Today's code enforces: target is in
  the connector's allowlist, an approval exists and matches the action id, the
  approval hasn't expired, and the post-condition is verified after execute.
  It does **not** enforce: asset criticality lookup, blast-radius thresholds,
  or confidence/evidence-sufficiency checks — those are still human judgment
  calls the approver makes by reading the case. Have the group walk through
  all ten gates and mark each as "enforced by code" or "trusted to the
  approver" — this mapping is itself a useful output of the exercise.

## Prerequisites

- The lab sandbox is up: `labnet`, the OPNsense VM, Kali on `labnet`, the Wazuh
  stack, and the real victim host enrolled as a Wazuh agent (see `adr/0013`'s
  "Sandbox this was built and proven against").
- The API server is running against that sandbox with
  `OPNSENSE_ENABLE_REAL=true` and valid `OPNSENSE_BASE_URL` /
  `OPNSENSE_API_KEY` / `OPNSENSE_API_SECRET` — confirm with the server's
  startup log line `"network connector selected" provider=connector-opnsense-lab@1.0.0`.
  Running with the flag unset (mock connector) first is a reasonable dry run
  of the *process* before touching the real firewall, but does not satisfy
  M5's gate on its own.
- `DATABASE_URL` points at a real Postgres instance with this repo's schema
  applied (not the `//go:build integration` Testcontainers path).
- Use a TEST-NET-3 address (`203.0.113.0/24`, per RFC 5737) as the "attacker
  IP" observable for the live write step, the same convention
  `services/api/opnsense/opnsense_lab_test.go` uses — never the real victim's,
  Kali's, or OPNsense's actual address. This keeps the live block/verify/
  release cycle real (it genuinely round-trips through the OPNsense API) without
  ever touching real lab traffic. Note for discussion: this also means the
  exercise cannot demonstrate blocking Kali's literal address end-to-end,
  because of the double-NAT source-masking limitation `adr/0013` already
  discloses.

## Roles

| Role | Stands in for | Does in this exercise |
|---|---|---|
| Analyst | `services/api/cases` "analyst" actor | Ingests the event, reviews the case, calls preview, requests the block |
| Incident commander (approver) | `services/api/actions` "incident-commander" actor | Reviews the pending approval, decides approve/reject, asks the discussion questions below before approving |
| IT / victim service owner | Real-world stakeholder for the enrolled victim host | Observer; answers "would you have signed off on blocking this in production?" |
| DFIR / evidence reviewer | `services/api/evidence`, `services/api/audit` | After execute, pulls the audit log and evidence export, checks it reads sensibly to someone who wasn't in the room |
| Scribe | — | Fills in the decision log (see "Outputs to capture") in real time |

## Script

Run these against the live API (`http://<api-host>:8080` — adjust `ADDR` and
tenant). Every mutating call needs an `Idempotency-Key` header of at least 8
characters (`services/api/httpx.MinIdempotencyKeyLength`); use a fresh one per
call.

**1. Analyst ingests the alert.**

```bash
curl -s -X POST http://localhost:8080/v1/events \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: tabletop-ingest-001' \
  -d '{
    "source": "wazuh",
    "occurred_at": "2026-09-02T04:00:00Z",
    "payload": {
      "source_event_id": "wazuh-alert-tabletop-0001",
      "event_class": "authentication",
      "severity": "medium",
      "confidence": 0.82,
      "observables": [
        {"type": "ipv4", "value": "<the TEST-NET-3 IP>", "normalization": "rfc5737-test-net",
         "first_seen": "2026-09-02T04:00:00Z", "last_seen": "2026-09-02T04:00:00Z",
         "confidence": 0.82, "sensitivity": "internal", "source_refs": ["wazuh-alert-tabletop-0001"]}
      ]
    }
  }'
```
`POST /v1/events` takes this envelope shape (`source`/`occurred_at`/`payload`),
**not** the internal normalized-event shape `examples/normalized-event.json`
documents (that fixture is what an event looks like *after* normalization —
see `services/api/events/event.go`'s `Normalize`, and
`services/api/integration/ingestion_test.go`'s `fixture` helper for the
authoritative request shape). Note the returned `event_id`/`case_id` (or read
them back via `GET /v1/cases`).

**2. Analyst reviews the case and calls preview.**

```bash
curl -s -X POST http://localhost:8080/v1/actions/preview \
  -H 'Content-Type: application/json' \
  -d '{"intent":"block_network_indicator","target":"<the TEST-NET-3 IP>"}'
```
*Discussion:* with the real connector wired in, this response now reports
whether the alias already contains the IP
([wiring added after ADR-0013](../../adr/0013-real-opnsense-lab-connector.md)).
What would the analyst do differently if it came back "already blocked"?

**3. Analyst requests the block on the case.**

```bash
curl -s -X PATCH http://localhost:8080/v1/cases/<case_id> \
  -H 'Content-Type: application/json' \
  -d '{"operation":"request_block","target_value":"<the TEST-NET-3 IP>"}'
```
This only succeeds if the IP is already an observable on the case (it will be,
from step 1) — read `services/api/cases/handler.go`'s `RequestBlock` case if
the group wants to see why. Note the created `action_id`.

**4. Incident commander reviews the pending approval.**

```bash
curl -s http://localhost:8080/v1/approvals
```
*Discussion, before approving:* What information would you want here that
this response does not carry? Is a target IP and a case ID enough to approve a
live firewall change in a production incident? What would "asset criticality"
or "blast radius" need to look like for this system to answer that question
instead of the human?

**5. Incident commander approves.**

```bash
curl -s -X POST http://localhost:8080/v1/approvals/<approval_id>/approve \
  -H 'Idempotency-Key: tabletop-approve-001'
```

**6. Analyst executes.**

```bash
curl -s -X POST http://localhost:8080/v1/actions/<action_id>/execute \
  -H 'Idempotency-Key: tabletop-execute-001'
```
Expect `202 Accepted`. This is the call that reaches the real OPNsense VM.

**7. Everyone verifies against the real firewall**, independent of this
system's own claim that it worked:

```bash
curl -s -k -u "$OPNSENSE_API_KEY:$OPNSENSE_API_SECRET" \
  https://10.10.10.1/api/firewall/alias_util/list/SOAR_TEMP_BLOCK
```
Confirm the TEST-NET-3 IP is present. This is the same call
`opnsense.Connector.contains` makes internally — running it by hand is the
point: an approver or auditor should be able to independently confirm a claim
this system makes, not just trust its response.

**8. Manual rollback (the documented gap).** There is no HTTP endpoint for
this. Either call `actions.Release` directly (e.g. from a short Go program or
`go test -run` against the real connector, mirroring
`opnsense_lab_test.go`'s cleanup step) or remove the IP via the OPNsense UI/API
directly. *Discussion:* is "no self-service rollback endpoint" acceptable for
a lab, and what would need to exist before a production pilot (M6) could rely
on this?

**9. DFIR reviewer checks the trail.**

```bash
curl -s http://localhost:8080/v1/audit
curl -s -X POST http://localhost:8080/v1/cases/<case_id>/export \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: tabletop-export-001' \
  -d '{"format":"json","redaction_profile":"standard","purpose":"tabletop exercise review","recipient":"dfir-reviewer"}'
```
Confirm the raw IP never appears in **the audit log** (only `target_hash`) —
it legitimately *does* appear in this export's own manifest (the case's own
observable/target values), since `standard` redaction only strips each
evidence entry's `uri`/`command_ref` per `adr/0012`, not case content itself.
Confirm the export's `manifest_sha256` matches a hash you can independently
recompute over the returned manifest (sorted keys, compact separators —
see `services/api/cases/export.go` if the group wants the exact encoding).

## Outputs to capture

Per `tabletop-exercises.md`'s requirement, the scribe records, live, in a
shared document (not in this repo — this is operational exercise output, not
design content):

| Decision point | What was decided | Gap identified | Owner | Due date |
|---|---|---|---|---|
| e.g. step 4 approval review | ... | "no asset-criticality lookup exists" | ... | ... |

At minimum, expect to log the two gaps this script already calls out (no
release/rollback HTTP endpoint; most of the ten safe-response-controls gates
are still human-trusted, not code-enforced) plus whatever the group surfaces
during steps 4 and 8's discussion prompts.

## After the exercise

- Update `docs/10-roadmap/milestones.md`'s M5 row to "Done" only once this
  script (or an equivalent) has actually been run with the roles staffed by
  real people and the decision log above completed — a solo dry run by one
  engineer clicking through the API does not satisfy "tabletop approval."
  Include the exercise date and participants in the commit/PR that flips it.
  Ownership of the sign-off is not this document's or Claude Code's to give.
- File the identified gaps (e.g., the missing manual-release endpoint) as
  follow-up work rather than fixing them inline during the exercise — the
  point of a tabletop is to find gaps under realistic conditions, not to close
  them mid-session.
- If the group decides an ADR-0013 assumption no longer holds (for example,
  the double-NAT masking caveat turns out to matter more than expected for
  M6), record that as a new ADR or an amendment, not a silent edit to
  ADR-0013's already-accepted decision.
