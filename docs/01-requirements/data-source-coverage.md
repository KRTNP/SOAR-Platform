# Data-source coverage matrix

การออกแบบ playbook ต้องผูกกับ data source จริง ไม่ใช่ดูจากชื่อ alert เพียงอย่างเดียว

| Investigation question | Required source | Minimum fields | Playbook use |
|---|---|---|---|
| ใคร login จากไหน | identity/sign-in | user, time, IP, device, MFA, result | account/credential |
| process ใดรัน | EDR/Sysmon | host, PID, parent, command line, hash, time | endpoint malware |
| มี network sighting หรือไม่ | firewall/proxy/DNS | src, dst, domain, port, action, time | IOC/exfil |
| email ไปถึงใครบ้าง | mail security | message ID, sender, recipients, URLs, attachments | phishing |
| asset สำคัญแค่ไหน | asset/CMDB | owner, criticality, service, zone | approval/blast radius |
| CVE กระทบอะไร | vulnerability inventory | asset, product, version, exposure, owner | exploited vulnerability |
| หลักฐานอยู่ที่ไหน | DFIR case | artifact, URI, hash, custody, tool | malware/ransomware |

## Data quality gates

- ไม่มี timezone: ใช้ timeline-critical decision ไม่ได้จนกว่าจะ resolve
- ไม่มี asset owner/criticality: ห้าม high-impact action
- ไม่มี source event ID: ต้องสร้าง deterministic fingerprint ก่อน dedup
- มี field แต่ไม่มี retention: แจ้ง limitation ไม่ถือว่าไม่มีเหตุการณ์
- provider result ไม่มี confidence/source timestamp: ใช้เป็น context เท่านั้น ไม่ใช้เป็น sole block reason

