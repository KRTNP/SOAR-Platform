# ADR-0006: Use an isolated KVM lab topology

## Status

Accepted — design only; implementation not started.

## Decision

Run the SOAR stack with Docker Compose on an Ubuntu host using KVM/libvirt, connected
to an isolated host-only network. Use the existing Wazuh instance, an Ubuntu victim VM
with Wazuh agent, a Kali test VM, and an OPNsense firewall VM. Never bridge attacker or
victim interfaces to a production/LAN network.

## Rationale

This permits real telemetry and a real firewall action while preserving a hard lab
boundary. Snapshots and allowlists make the first write-enabled scenario reversible.

## Consequences

- The host needs the recommended 8 vCPU, 32 GB RAM and 500 GB SSD profile.
- The Wazuh central stack is not counted in the local Compose resource budget.
- Network and VM snapshots are part of the test precondition, not optional hygiene.

