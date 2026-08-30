# Lab deployment profile

## Host target

| Resource | Recommended target |
|---|---|
| Host OS | Ubuntu Server 24.04 LTS |
| Hypervisor | KVM/libvirt |
| CPU | 8 vCPU |
| Memory | 32 GB RAM |
| Storage | 500 GB SSD |
| Network | libvirt isolated/host-only network |
| Deployment | Docker Compose for SOAR services |

This profile assumes the Wazuh central instance already exists outside the local
Compose stack. If Wazuh indexer/server is also hosted on this machine, use a separate
host profile with at least 12-16 vCPU, 48-64 GB RAM and 500 GB-1 TB SSD.

## Lab nodes

| Node | Purpose | Network rule |
|---|---|---|
| SOAR host | Next.js, Go API/workers, Temporal, NATS, PostgreSQL, OpenSearch, Keycloak, Vault and observability | Host-only; controlled egress only |
| Wazuh instance | Real alert source and API | Reachable only from approved SOAR connector path |
| Ubuntu victim VM | SSH service and Wazuh agent | Accepts test traffic only from Kali lab address |
| Kali VM | Authorized attack simulation | No route to production or management LAN |
| OPNsense VM | Lab firewall and temporary block target | API reachable only from response connector |

## Network invariants

- No bridged adapter for Kali or victim.
- No route from the isolated lab subnet to production networks.
- OPNsense management API is not exposed to the attacker subnet.
- Wazuh webhook/API credentials are scoped to the lab tenant and connector operation.
- Snapshot victim and OPNsense before each write-action exercise.

## Service profiles

The Compose design is split into `core`, `workflow`, `security`, `search`, and
`observability` profiles so the operator can start a bounded subset without changing
the contract. The full lab profile is the target; lower profiles are for diagnostics,
not alternate architecture.

