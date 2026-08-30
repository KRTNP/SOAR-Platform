# ADR-0001: Documentation-first baseline

## Decision

เริ่มจาก design repository ที่มี contract, examples, testable schemas และ playbook ก่อนเขียน production engine

## Rationale

SOAR มีความเสี่ยงจากการเชื่อมหลายระบบและ action ที่เปลี่ยน state การกำหนด domain, permission, evidence และ failure behavior ก่อนลด rework และลด unsafe automation

## Consequences

ต้องรักษาเอกสารให้สอดคล้องกับ implementation ภายหลัง แต่ได้ baseline ที่ review ได้โดยไม่ต่อระบบจริง

