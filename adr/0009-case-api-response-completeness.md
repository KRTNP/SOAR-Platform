# ADR-0009: Bring the Case API response into line with the Case domain schema

## Status

Accepted — implemented in the M1 slice (`services/api`, `web`).

## Decision

Add `hypothesis`, `facts`, `limitations`, `correlation_key`, `root_cause`,
`lessons_learned`, and a minimal `observables` array (`{type, value}` pairs,
not full observable objects) to the `Case` component schema in
`api/openapi.yaml`. All additions are optional properties — a
backward-compatible addition per `CONTRIBUTING.md`, not a breaking change.

## Rationale

`schemas/case.schema.json` already defines `hypothesis`, `facts`,
`limitations`, `correlation_key`, `root_cause`, and `lessons_learned` as part
of the canonical Case domain model, and the M1 Go implementation's internal
`cases.Case` struct and Postgres schema already store all of them. But
`api/openapi.yaml`'s `Case` response schema — the actual wire contract the
Next.js frontend consumes — never carried these fields; it only had the
narrower identity/lifecycle fields (id, title, category, severity, status,
owner, timestamps, `*_refs` arrays). That gap meant the Case Detail page
could not show an investigation summary at all, even though the data existed
one query away, and a case's linked observables (the IPs/hosts an analyst
actually needs to see) were reachable only via `observable_refs` — internal
database IDs with no way to resolve them back to a human-readable value
without a second, undefined endpoint.

This is a straightforward spec-drift fix, not new scope: it makes the
already-approved M1 case-detail feature ("Case Detail shows
summary/facts/observables/timeline") match the API contract to the data
model that was already built for it.

## Consequences

- `GET /v1/cases/{caseId}` (and the list/create responses) now return these
  fields when present; `services/api/cases/handler.go`'s `toAPICase` and the
  Next.js `Case` type in `web/lib/api-client/types.ts` were updated to match.
- `observables` on the Case response is intentionally a minimal
  `{type, value}` projection, not the full `observable.schema.json` object
  (confidence, sensitivity, source_refs, ...) — the case-detail UI only needs
  enough to render an identifying chip. A future milestone that needs the
  full observable record should add a dedicated
  `GET /v1/cases/{caseId}/observables` resource rather than growing this
  field further.
- No case-authoring UI exists yet to let an analyst write a hypothesis or
  facts (M1 has no such editor) — the fields flow through read-only until
  that's built.
