# ADR-0013: Real OPNsense lab connector, env-gated behind the M3 mock

## Status

Accepted — connector code implemented and proven against the real lab VM
(`services/api/opnsense`); M5's other gate condition (tabletop exercise) is
still open. A runnable script for it exists at
`docs/08-testing/tabletop-m5-lab-connector.md`, but has not yet been run with
real people in the roles it defines. `docs/10-roadmap/milestones.md`'s M5 row
stays "Not started" until that happens.

## Decision

Implement `docs/10-roadmap/milestones.md`'s M5 ("lab connector") real half as
`services/api/opnsense`, a `connectors.Connector` that calls a real OPNsense
lab VM's `alias_util` REST API instead of writing to Postgres, and wire it
into `services/api/actions.Execute`/`Release` as an alternative to
`services/api/network` (ADR-0011's mock), selected once at process startup by
`OPNSENSE_ENABLE_REAL` — **default false, so every environment without lab
credentials (CI, another developer's machine) keeps running the mock
unchanged.** `services/api/actions` no longer imports a concrete connector
package at all; it depends on a small `NetworkConnector{New, Release, Target,
Version}` factory that `services/api/main.go` builds from environment
configuration and injects into `actions.NewHandler`.

`examples/opnsense-connector.json` already existed from the design baseline
(manifest `connector-opnsense-lab`, intents `preview_temp_block`/
`add_temp_block`/`remove_temp_block`/`health`, target allowlist
`["opnsense-lab","SOAR_TEMP_BLOCK"]`); `services/api/opnsense`'s
`connectors.Manifest` value is hand-synced to it, the same disclosed
hand-sync convention ADR-0010/0011/0012 already use for every connector in
this codebase.

### Sandbox this was built and proven against

The lab infrastructure that makes this a *real* connector rather than a
second mock: an isolated libvirt network (`labnet`, `10.10.10.0/24`), an
OPNsense VM routing it (LAN `10.10.10.1`, WAN NAT'd out through the host's
existing `default` libvirt network), the existing Kali VM re-homed onto
`labnet` as the fixed attacker (`10.10.10.10`), the existing Wazuh stack
restarted, and a real, already-running, production-adjacent Ubuntu machine
enrolled as a genuine Wazuh-monitored SSH victim — **a deliberate,
user-disclosed deviation from `adr/0006-lab-topology-and-boundaries.md`**'s
"never bridge attacker or victim interfaces to a production/LAN network."
`adr/0006` isolates the *victim*; this lab instead makes OPNsense the
*attacker's only egress path* — every packet Kali sends anywhere, including
to the real victim, must cross OPNsense's LAN interface, so blocking Kali's
source IP on that interface genuinely stops its traffic without ever
touching the victim or the home router. This is a narrower guarantee than
`adr/0006`'s (it does nothing to protect the victim from anything other than
this lab's own attacker VM), but it is the guarantee this connector actually
needs to prove `add_temp_block`/`remove_temp_block` for real.

### OPNsense manifest, concretely

- Alias `SOAR_TEMP_BLOCK` (host/IP list type), with one LAN-interface
  firewall rule blocking any source matching that alias — created once via
  the OPNsense web UI, not by this connector (the connector only ever calls
  `alias_util`, never creates aliases or rules).
- API user `soar-api`, scoped to the single least-privilege permission
  `page-diagnostics-tables` (the page that owns `alias_util`) — not a full
  API-access role.
- Confirmed API contract, empirically, against the real VM:
  - `GET /api/firewall/alias_util/list/{alias}` →
    `{"total":N,"rowCount":N,"current":1,"rows":[{"ip":"x.x.x.x"}, ...]}`.
  - `POST /api/firewall/alias_util/add/{alias}` and
    `POST /api/firewall/alias_util/delete/{alias}`, **both take the address
    as a JSON POST body field (`{"address": "…"}`), never as a URL path
    segment** — an early manual attempt at `.../delete/SOAR_TEMP_BLOCK/<ip>`
    returned `{"status":"failed"}` silently rather than an error, which cost
    real debugging time; `services/api/opnsense`'s `mutate` re-verifies the
    alias's actual state after every add/delete for exactly this reason,
    classifying a mismatch as `postcondition_mismatch` rather than trusting
    `{"status":"done"}` alone.
  - No separate "apply"/"reconfigure" step exists or is needed —
    `alias_util` mutates the live pf table immediately.

## Rationale

`docs/05-integrations/lab-integration-profile.md` already specified this
connector's exact four operations and their approval/risk posture before any
Go code existed. Building the real thing only became possible once the
sandbox itself existed (this session's separate infra build); this ADR
records the code side once that was true, following ADR-0010/0011/0012's
same mock-now/real-when-ready sequencing — the difference here is that
"ready" is a real VM, not just a later milestone.

Env-gating the wiring (rather than replacing `services/api/network` outright)
follows directly from `AGENTS.md`'s "do not... connect to live security
systems without explicit authorization": this repository, cloned anywhere
else, must keep behaving exactly as it did before this ADR unless someone
deliberately opts in with real OPNsense credentials. `OPNSENSE_ENABLE_REAL`
is a distinct switch from merely having `OPNSENSE_BASE_URL`/`OPNSENSE_API_KEY`
set, so a leftover or copy-pasted credential alone can never flip production
behavior.

TLS verification stays on by default; `OPNSENSE_INSECURE_SKIP_VERIFY` exists
only because this specific lab VM presents OPNsense's stock self-signed
certificate and no CA was set up for it. This is disclosed as a real
weakening of transport security, acceptable only because the target is a
lab VM on an isolated network the operator controls end-to-end — anyone
pointing this connector at a non-lab OPNsense instance should provision a
real certificate instead of setting this flag.

## Consequences

- **The mock stays the default everywhere.** `OPNSENSE_ENABLE_REAL=true`
  plus a valid `opnsense.LoadConfig()` are both required before the real
  connector runs at all; a bad config with the flag set fails process
  startup (`services/api/main.go`'s `buildNetworkConnector`) rather than
  silently falling back to the mock — a bad OPNsense config should be loud,
  not quietly no-op.
- **`preview_temp_block` is wired to `POST /v1/actions/preview` via the same
  `NetworkConnector` factory seam as `execute`/`release`.** `actions.
  NetworkConnector` gained an optional `Preview(ctx, target) (bool, error)`
  field; `main.go` wires it to `opnsense.Connector.Preview` only for the real
  connector, leaving it `nil` for the mock (which has no standalone existence
  query outside of an action row). `actions.Handler.Preview` calls it when
  present and reports whether the target is already blocked; when absent
  (the mock, the default) it falls back to the original generic dry-run
  message, so this is additive and does not change M3's existing behavior.
- **Idempotency is checked live, not cached.** Unlike `network.block`'s
  `select exists` against Postgres, `opnsense.Connector` calls `list` before
  every add/remove to decide whether it would have an effect — correct, but
  it means every `execute`/`release` call costs at least two HTTP round
  trips to the lab VM (list, then add/delete, then a postcondition list).
  Acceptable at this lab's volume; would need caching or a different
  idempotency strategy at real production scale.
- **Two independent contract-test surfaces now exist for the same intent
  family**: `services/api/integration/connector_contract_test.go`'s
  Postgres-backed `TestNetworkConnectorContract` (mock, `//go:build
  integration`) and `services/api/opnsense/opnsense_test.go`'s
  `httptest`-backed `TestOpnsenseConnectorContract` (real client shape, fake
  server, no build tag — runs in default CI). A third file,
  `services/api/opnsense/opnsense_lab_test.go` (`//go:build lab`), is the
  only one that touches the actual VM; it `t.Skip`s cleanly without
  `OPNSENSE_*` env vars and must be run manually.
- **Two lab-specific operational fragilities are worth recording** for
  anyone re-running this sandbox, though neither is this connector's
  problem to fix: (1) double-NAT source masking — Kali's traffic to the real
  victim crosses OPNsense's WAN NAT and then the lab host's own Wi-Fi NAT, so
  Wazuh on the victim only ever sees the lab host's own NAT'd IP as the
  "attacker" source, never Kali's or OPNsense's IP; this does not affect the
  block mechanism (which acts upstream of both NAT hops, on OPNsense's LAN
  interface) but does mean this topology cannot correlate a Wazuh alert back
  to Kali's literal address. (2) The lab host's own Wi-Fi DHCP lease can
  change between sessions, silently breaking the victim's Wazuh agent's
  configured manager address (`ossec.conf`) until manually corrected — a
  fragility inherent to hosting the manager on a DHCP client rather than a
  fixed address.
- **No scheduler drives TTL/rollback for the real connector either** —
  same lazy, read-path-triggered expiry ADR-0011 already disclosed for the
  mock; `services/api/opnsense` adds no new capability here, it only changes
  which system the eventual scheduled rollback (M5/M6) will call.
