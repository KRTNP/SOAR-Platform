# ADR-0007: Start with SSH brute-force and temporary IP block

## Status

Accepted — design only; implementation not started.

## Decision

The first end-to-end path is five failed SSH logins within one minute from a single
source IP. It starts at Medium severity, escalates for privileged users or critical
assets, automatically creates a case/enriches/notifies, and requires one approver for
a ten-minute OPNsense `SOAR_TEMP_BLOCK` action. The action verifies both alias
membership and blocked traffic, then removes the IP with retry at 5/15/30 seconds.

## Rationale

The scenario exercises real Wazuh ingestion, event correlation, case management,
human approval, provider write action, post-condition verification, rollback, audit,
and closure without needing a broad destructive action.

## Consequences

- Only this OPNsense action is write-enabled in the lab baseline.
- Other actions such as account disable, session revoke and endpoint isolation remain
  approval/dry-run candidates until separately tested.
- Trusted management IPs and the isolated lab scope are mandatory safety controls.

