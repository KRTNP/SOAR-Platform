# EDR/XDR integration

## Investigation intents

- get host/process/user/network context
- search IOC และ process tree
- collect targeted artifact
- retrieve detection verdict

## Response intents

- isolate host
- release/isolate quarantine artifact
- kill process
- restore connectivity

Response ต้องส่ง asset ID ที่ canonical, ตรวจ host criticality, ใช้ TTL เมื่อ provider รองรับ และ verify state หลัง action

