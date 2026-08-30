# SOAR playbooks

Playbook ใน directory นี้เป็น operational design examples ที่ default เป็น `dry_run` หรือ investigation-only จนกว่าจะมี authorization และ policy ที่เหมาะสม

## Files

- `templates/playbook-template.yaml` — contract template
- `ssh-bruteforce-response.yaml` — first end-to-end scenario ([ADR-0007](../adr/0007-first-vertical-scenario.md)): Wazuh detection → gated OPNsense temp block → verified rollback
- `wazuh-alert-triage.yaml` — รับ alert และทำ enrichment
- `suspicious-ioc.yaml` — สืบค้น IOC แบบ bounded
- `phishing-triage.yaml` — วิเคราะห์อีเมลต้องสงสัย
- `account-compromise.yaml` — identity investigation และ gated containment
- `endpoint-malware.yaml` — endpoint triage, evidence และ isolation approval
- `exploited-vulnerability.yaml` — map asset, ตรวจ exploitation และ remediation
- `ransomware-response.yaml` — incident commander และ business-aware containment
- `credential-stuffing.yaml` — correlate ความล้มเหลว/สำเร็จของการ login และ gated containment
- `data-exfiltration.yaml` — scope การ transfer ต้องสงสัยและ gated containment
- `insider-threat.yaml` — corroborate สัญญาณหลายแหล่งภายใต้ minimum-necessary scope
- `cloud-identity-compromise.yaml` — cloud audit/IAM investigation และ gated containment

Source-derived pack (ดู [source-derived playbooks](../docs/04-playbooks/source-derived-playbooks.md) สำหรับ source basis และ adaptation contract):

- `cisa-major-incident-coordination.yaml` — declare incident, assign role, coordinate และ gated containment
- `cisa-vulnerability-response.yaml` — intake, prioritize, notify owner และ gated remediation
- `nist-recovery.yaml` — restore service ตาม dependency order พร้อม validation
- `dfir-evidence-preservation.yaml` — read-only evidence acquisition และ custody

มาตรฐานและ lifecycle อยู่ที่ [playbook standard](../docs/04-playbooks/playbook-standard.md)

รายการทั้งหมดอยู่ที่ [playbook catalog](../docs/04-playbooks/playbook-catalog.md) และ
playbook ที่อิง source จริงอยู่ใน [source-derived pack](../docs/04-playbooks/source-derived-playbooks.md)
