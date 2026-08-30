# Lab scenario: SSH brute-force to controlled temporary block

This is the first end-to-end scenario. It is limited to the isolated lab network and
must use only the designated Kali, Ubuntu victim, Wazuh and OPNsense systems.

## Expected policy

| Input | Rule |
|---|---|
| Detection | 5 failed SSH logins within 1 minute from one source IP |
| Initial severity | Medium |
| Escalation | Medium becomes higher priority only when source targets a privileged user or critical asset |
| Auto actions | Create/update case, enrich IP/user/asset, notify analyst |
| Write action | Add source IP to `SOAR_TEMP_BLOCK` only after one authorized approval |
| TTL | 10 minutes |
| Review SLA | Analyst review within 15 minutes |
| Rollback | Remove IP from alias; retry at 5, 15 and 30 seconds; retain block and escalate if all fail |
| Trusted sources | Management/admin allowlist cannot be blocked; allowlist changes are audited and expire |

## Acceptance sequence

1. Wazuh produces an authenticated event for the designated Ubuntu victim.
2. SOAR stores a redacted raw reference, checksum and normalized event.
3. Duplicate source events do not create duplicate actions.
4. Correlation produces one Medium alert and one case/task.
5. Enrichment failure is visible as partial data and does not silently approve a write.
6. A privileged/critical context raises priority; an ordinary context remains Medium.
7. The UI shows a pending approval with exact IP, case, action hash, TTL and rollback.
8. An authorized approver approves once; a second submission is idempotent.
9. OPNsense contains the IP in `SOAR_TEMP_BLOCK` and the test traffic is blocked.
10. After ten minutes the connector removes the IP and verifies baseline traffic behavior.
11. A simulated rollback failure retries at 5/15/30 seconds, then creates escalation while retaining the block.
12. Case closes only after rollback succeeds, no repeated alert is present, and analyst review is recorded.

## Negative tests

- Source IP is on trusted allowlist: no block action is created.
- Approval is expired or action hash changed: execution is denied.
- OPNsense API timeout: action is `provider_timeout`, not success.
- Alias membership verification fails: post-condition fails and escalation opens.
- Wazuh duplicate event: same idempotency identity is returned.
- Evidence/case scope is missing: workflow stops and creates a manual task.
- SOAR loses connection during TTL cleanup: rollback retry and escalation state remain durable.

## Evidence produced

The test run records fixture IDs, Wazuh event reference, normalized event version,
case ID, policy decision, approval, OPNsense request/response references, alias
verification, rollback attempts, audit hash chain and final case state. No production
credentials or original forensic evidence are used.

