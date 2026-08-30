# Roadmap

## Phase 0 — Design baseline

สร้าง domain contract, playbook contract, threat model, schemas, examples และ diagrams

## Phase 1 — Safe workflow core

สร้าง ingestion, normalization, case, queue, workflow state, audit และ connector boundary โดยใช้ mock เฉพาะ provider ที่ยังไม่มี sandbox/instance พร้อมใช้

## Phase 2 — Operational integrations

เปิดใช้ Wazuh webhook จริงใน lab และเพิ่ม SIEM query, TI, ticketing และ DFIR task orchestration โดย default เป็น read-only/dry-run; provider ที่ยังไม่มี sandbox ใช้ deterministic mock

## Phase 3 — Governed response

เพิ่ม policy engine, approval, TTL, idempotent firewall/EDR/identity action และ post-condition verification

## Phase 4 — Resilience and measurement

เพิ่ม worker isolation, retries, circuit breaker, replay regression, metrics, backup/restore และ tabletop

## Phase 5 — Assistive intelligence

เพิ่ม evidence-backed summarization และ playbook/module recommendation โดย human review ยังคงเป็น gate
