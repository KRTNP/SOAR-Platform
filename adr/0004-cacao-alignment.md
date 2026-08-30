# ADR-0004: CACAO-aligned playbook representation

## Decision

playbook metadata, workflow, agents/targets, versioning, markings และ signatures จะออกแบบให้ map กับ CACAO 2.0 ได้ โดย internal YAML อาจเป็น operational subset

## Rationale

CACAO ให้โครงสร้างมาตรฐานสำหรับสร้าง, document, test และ share cyber security playbook ข้าม technology boundary

## Consequences

มีความสามารถ portability และ governance ที่ดีขึ้น แต่ต้องทำ mapping ระหว่าง internal connector intent กับ CACAO agent/target อย่างชัดเจน

