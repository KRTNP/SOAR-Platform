# Detection-to-playbook mapping

mapping ต้อง explicit และมี fallback:

| Detection class | Default playbook | Fallback |
|---|---|---|
| 5 failed SSH logins within 1 minute from one source IP | ssh-bruteforce-response | investigate-only |
| authentication anomaly (single identity, privilege change) | account-compromise | investigate-only |
| distributed/repeated login failure | credential-stuffing | investigate-only |
| malicious process/persistence | endpoint-malware | DFIR task |
| suspicious network IOC | suspicious-ioc | analyst task |
| suspicious email | phishing-triage | preserve-only |
| active CVE exploitation | exploited-vulnerability | vulnerability owner |
| multi-host encryption | ransomware-response | incident commander |
| unusual transfer volume/sensitive data movement | data-exfiltration | investigate-only |
| corroborated multi-signal insider risk | insider-threat | legal/privacy review |
| cloud audit/IAM anomaly | cloud-identity-compromise | investigate-only |

Major-incident coordination (`cisa-major-incident-coordination`), vulnerability intake (`cisa-vulnerability-response`), recovery (`nist-recovery`) and evidence preservation (`dfir-evidence-preservation`) are not detection-triggered; they are launched manually by an incident commander, vulnerability owner or DFIR lead once one of the rows above escalates past its fallback.

การ map ต้องตรวจ source, event class, required fields, minimum confidence, tenant และ playbook status ก่อน launch หากหลาย playbook match ให้ใช้ priority ที่ versioned และไม่ execute response สองชุดโดยอัตโนมัติ

