# ADR-0002: Internal event schema

## Decision

ใช้ internal canonical model ที่ได้รับแรงบันดาลใจจาก OCSF และเก็บ vendor extensions พร้อม mapping metadata

## Rationale

OCSF ช่วยให้ security event ต่าง vendor ใช้ภาษากลางได้ แต่ระบบ workflow ต้องการ subset ที่เล็กและเน้น case/action จึงไม่ควรอ้าง implementation ว่า OCSF เต็มรูปแบบโดยไม่มี conformance test

## Consequences

adapter ต้องดูแล mapping และ schema version แต่ core playbook ใช้ซ้ำข้าม provider ได้

