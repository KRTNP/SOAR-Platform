# Risk scoring model

คะแนนนี้เป็น decision aid ไม่ใช่แทน analyst โดยกำหนดผลรวมแบบ deterministic เพื่อให้ replay ได้

## Dimensions

| Dimension | Range | ตัวอย่าง |
|---|---:|---|
| Detection confidence | 0–30 | quality ของ rule/indicator |
| Evidence strength | 0–25 | independent corroboration |
| Asset criticality | 0–25 | business/service impact |
| Blast radius | 0–15 | จำนวน target และ reversibility |
| Threat urgency | 0–15 | active exploitation/time pressure |

## Decision bands

- 0–24: observe/investigate
- 25–49: analyst task และ enrichment เพิ่ม
- 50–74: gated response พร้อม approval
- 75–100: incident commander escalation และ gated containment

คะแนนต้องเก็บ component, rule version, input references และ reason เพื่อให้ analyst override ได้อย่างอธิบายได้

