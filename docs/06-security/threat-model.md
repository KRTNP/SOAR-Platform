# Threat model

## Assets

- security credentials and connector tokens
- case data, PII, evidence and investigation notes
- playbooks, policies and detection mappings
- response authority over endpoint, network and identity
- audit trail and approval records

## Threat actors

- attacker spoofing source webhook
- compromised connector credential
- malicious or compromised worker
- insider abusing approval or break-glass
- analyst tricked by poisoned email/document
- LLM prompt injection through untrusted incident data

## Primary abuse paths

| Path | Control |
|---|---|
| forged alert → response | signed/authenticated ingress, replay protection, policy gate |
| untrusted IOC → SSRF | type validation, egress allowlist, no arbitrary URL fetch |
| alert field → command injection | intent API, no shell interpolation, allowlisted args |
| low-confidence verdict → block | confidence threshold + human approval |
| connector token theft | short-lived secret, scoped identity, worker isolation |
| evidence tampering | hash, append-only custody, read-only original |
| duplicate storm → repeated action | idempotency, dedup, rate limit |
| LLM output → tool call | schema validation, policy re-check, human gate |

