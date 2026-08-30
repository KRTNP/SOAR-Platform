# Use cases

## UC-01 Wazuh alert triage

รับ alert → validate signature/source → deduplicate → enrich IP/hash/user/host → correlate event → เปิด case → แจ้ง analyst

## UC-02 Phishing triage

รับ email metadata/header → extract URL/domain/hash → ตรวจ reputation และ authentication → ค้นหา recipients ที่ได้รับ → quarantine เมื่อ policy ผ่าน → เก็บ evidence

## UC-03 Compromised account

ตรวจ sign-in, MFA, device, source IP, privilege change และ mailbox rule → revoke session/token → บังคับ reset หลัง approval → ตรวจ lateral activity

## UC-04 Endpoint malware

ตรวจ process/persistence/network → ดึง targeted artifacts ผ่าน DFIR tool → สร้าง timeline → ขออนุมัติ isolate → verify containment → recovery task

## UC-05 Suspicious IOC campaign

รับ IOC จาก TI หรือ analyst → normalize → search across SIEM/EDR/DNS/email → รวมผลเป็น campaign → ส่ง detection update และ case task

## UC-06 Actively exploited vulnerability

รับ CVE/intelligence → map asset inventory → ตรวจ exposure และ exploitation evidence → จัดลำดับ → patch/mitigate → verify และรายงาน

## UC-07 Ransomware

ยืนยัน signal หลายแหล่ง → activate incident commander → preserve evidence → ประเมิน business dependency → containment แบบมี approval → recovery/communications

