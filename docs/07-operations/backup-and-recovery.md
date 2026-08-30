# Backup and recovery

- backup domain database, playbook/policy revisions, audit metadata และ evidence index แยกกัน
- encrypt backup และจำกัดผู้กู้คืน
- เก็บ retention ตาม classification/case policy
- ทดสอบ restore ใน isolated environment
- ตรวจ hash หลัง restore
- กู้คืน control plane แบบ fail-closed; ห้าม auto-resume destructive action โดยไม่มี operator review
- บันทึก RPO/RTO ต่อ component และผลทดสอบล่าสุด

