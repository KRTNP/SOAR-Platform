# Policies

## Minimum policies

- approved source and target inventory
- incident severity and escalation
- action risk classification
- human approval and break-glass
- evidence handling and custody
- data retention and deletion
- connector onboarding/review
- playbook authoring, testing and promotion
- emergency change and rollback
- third-party data transfer

Policy decision ต้อง deterministic, versioned และอ้าง policy ID/version ใน audit record

## Response-authority policy (action risk classification / human approval)

`schemas/policy.schema.json` is the response-authority artifact for the two
bullets above: a per-`intent` table of `allow` vs. `require_approval`
decisions plus the role(s) that should approve. `examples/policy.default.json`
is the current default policy — see [ADR-0016](../../adr/0016-response-authority-policy.md)
for what it covers, what it does not (no enforcement — no identity/authz
system exists yet to check a caller's role against it), and the one place it
is actually wired into `services/api` today (`request_block`'s audit reason).
The [approvals queue](../../web/app/approvals) reads the resulting approvals
across every case.

