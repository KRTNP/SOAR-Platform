# Observability

Stack ที่เลือกไว้คือ [ADR-0005](../../adr/0005-selected-software-stack.md): metrics ผ่าน OpenTelemetry SDK แล้ว export เป็น Prometheus exposition format, log ผ่าน Loki, trace ผ่าน OpenTelemetry (OTLP), และ dashboard/alert ทั้งหมดอยู่ใน Grafana เอกสารด้านล่างอธิบาย field/metric ที่ต้องเก็บแบบ stack-agnostic; ยัง unimplemented เช่นเดียวกับ service อื่นในเฟส design-first นี้

## Logs

เก็บ structured log ที่มี request/case/execution/action ID, component, connector, duration, status และ error class โดย redact sensitive values

## Metrics

- ingestion rate และ rejection rate
- queue depth/lag
- workflow duration และ step duration
- connector success/timeout/rate-limit/auth failure
- approval wait time
- post-condition failure
- duplicate ratio
- evidence/audit write failure

## Traces

trace หนึ่งเส้นควรเชื่อม ingress → normalization → correlation → case → workflow → connector → audit โดยไม่ใส่ payload ลับใน span attribute

