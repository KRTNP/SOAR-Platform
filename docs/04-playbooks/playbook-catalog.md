# Playbook catalog

| ID | Playbook | Default level | Primary output |
|---|---|---:|---|
| PB-SSHBF-001 | SSH brute-force detection and temporary OPNsense block | L3/L4 (single-approval exception per [ADR-0007](../../adr/0007-first-vertical-scenario.md)) | temporary IP block with verified rollback |
| PB-WAZUH-001 | Wazuh alert triage | L1 | enriched case |
| PB-IOC-001 | IOC investigation | L1/L2 | search result and sightings |
| PB-PHISH-001 | Phishing triage | L1/L3 | verdict and mailbox quarantine |
| PB-ACCOUNT-001 | Account compromise | L1/L3 | identity containment task |
| PB-ENDPOINT-001 | Endpoint malware | L1/L4 | evidence and isolation approval |
| PB-VULN-001 | Actively exploited vulnerability | L1/L2 | remediation task |
| PB-RANSOM-001 | Ransomware coordination | L1/L4 | incident commander workflow |
| PB-AUTH-001 | Credential stuffing and brute-force investigation | L1/L3 | containment task |
| PB-EXFIL-001 | Suspected data exfiltration investigation | L1/L3 | containment and notification record |
| PB-INSIDER-001 | Insider-threat signal review | L1/L3 | restricted case summary |
| PB-CLOUD-001 | Cloud identity compromise investigation | L1/L3 | containment and control-plane verification |
| PB-CISA-MAJOR-001 | Major incident coordination and controlled response | L1/L4 | incident commander workflow |
| PB-CISA-VULN-001 | Vulnerability intake, prioritization and verification | L1/L3 | remediation change record |
| PB-NIST-RECOVERY-001 | Evidence-informed service recovery | L1/L3 | validated service restore |
| PB-DFIR-EVIDENCE-001 | Read-only DFIR evidence preservation and custody | L1 | hash-verified evidence manifest |

PB-SSHBF-001 คือ first end-to-end scenario ตาม [ADR-0007](../../adr/0007-first-vertical-scenario.md) และ [SSH brute-force lab scenario](../08-testing/lab-scenario-ssh-bruteforce.md) — เป็น write-enabled playbook เดียวที่ผ่าน tested lab scope ในปัจจุบัน และเป็น playbook เดียวที่มี structured `automation` block ([ADR-0021](../../adr/0021-declarative-signature-registry.md)) เชื่อมกับ `services/api/signatures`'s registry จริง — ทุก playbook อื่นในตารางนี้ยังเป็น documentation เท่านั้น เพราะ `trigger.condition` ของแต่ละไฟล์เป็น prose ไม่มี threshold/window ที่ชัดเจนพอจะ mirror เข้า registry ได้ (สำรวจครบทั้ง 15 ไฟล์ที่เหลือแล้ว)

PB-WAZUH-001, PB-IOC-001 และ PB-VULN-001 เป็นลำดับเริ่มต้นที่ปลอดภัยกว่า response ที่เปลี่ยน network/identity state

PB-CISA-MAJOR-001, PB-CISA-VULN-001, PB-NIST-RECOVERY-001 และ PB-DFIR-EVIDENCE-001 เป็น source-derived pack — ดูรายละเอียด source basis และ adaptation contract ที่ [source-derived playbooks](source-derived-playbooks.md)

Default level อ้างอิงตาม [automation levels](automation-levels.md); L1/L2 หมายถึง playbook สร้างได้แค่ analyst task หรือ ticket ไม่มี approval+action step ที่เปลี่ยน state ปลายทาง, L1/L3 ขึ้นไปหมายถึงมี approval step ตามด้วย action step แยกกัน ([playbook standard](playbook-standard.md#approval-and-action-steps))
