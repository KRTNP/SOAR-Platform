# System overview

## Logical architecture

```mermaid
flowchart LR
  sources[Security Sources] --> ingress[Ingestion API]
  ingress --> normalize[Normalizer]
  normalize --> correlate[Correlation and Dedup]
  correlate --> case[Case Service]
  correlate --> queue[Workflow Queue]
  queue --> engine[Playbook Engine]
  engine --> enrich[Investigation Connectors]
  enrich --> decision[Policy and Approval]
  decision --> response[Response Connectors]
  response --> verify[Post-condition Verification]
  verify --> case
  engine --> audit[Audit and Evidence]
  case --> ui[Analyst UI/API]
  audit --> metrics[Observability]
```

## Boundaries

- **Ingress**: รับและ authenticate input; ไม่เรียก destructive action
- **Domain**: event, alert, observable และ case; ไม่รู้รายละเอียด vendor API
- **Orchestration**: รัน graph, state, retry และ approval
- **Connector**: แปลง domain action เป็น provider call
- **Policy**: ตรวจ scope, risk, asset criticality, confidence และ approval
- **Evidence/Audit**: เก็บ trace ที่ append-only และ reference ไป artifact
- **Control plane**: admin, schema, playbook และ credential configuration

## Source of truth

SIEM/EDR เป็น source ของ raw telemetry, SOAR เป็น source ของ case/execution/decision, และ DFIR case store เป็น source ของ evidence handling เมื่อมี forensic collection

