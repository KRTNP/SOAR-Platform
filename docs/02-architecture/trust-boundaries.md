# Trust boundaries

```mermaid
flowchart LR
  external[External Sources]
  gateway[Authenticated Ingress]
  core[SOAR Core]
  workers[Connector Workers]
  targets[Security Targets]
  evidence[Restricted Evidence Store]
  operator[Analyst and Approver]
  external --> gateway
  gateway --> core
  core --> workers
  workers --> targets
  core --> evidence
  operator --> core
  workers -. no direct control-plane access .-> core
```

Threats at each boundary:

- external → gateway: spoofed webhook, replay, oversized payload, injection
- gateway → core: tenant confusion, schema abuse, duplicate storm
- core → workers: privilege confusion, command injection, secret exposure
- workers → targets: wrong tenant/asset, excessive permission, destructive blast radius
- core → evidence: privacy leakage, tampering, retention violation
- operator → core: approval fraud, privilege escalation, insider misuse

