# Playbook testing methodology

1. **Static validation** — YAML/JSON schema, forbidden action, missing owner, missing stop condition
2. **Unit tests** — condition, mapping, redaction, idempotency key
3. **Connector mocks** — success, timeout, rate-limit, unauthorized, malformed response
4. **Replay tests** — historical/synthetic fixtures ให้ผล decision ที่คาดไว้
5. **Integration tests** — target sandbox และ test tenant
6. **Tabletop** — คนจริงเดินตามบทบาท การสื่อสาร และ approval
7. **Failure injection** — worker crash, duplicate event, queue backlog, provider outage
8. **Promotion gate** — reviewer ตรวจ diff, scope, permission และ evidence output

ผลทดสอบต้องเก็บ fixture version, playbook version, connector version, policy version, result และ timestamp

