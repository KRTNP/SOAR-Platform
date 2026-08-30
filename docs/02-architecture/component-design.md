# Component design

| Component | หน้าที่ | ห้ามรับผิดชอบ |
|---|---|---|
| Ingestion API | auth, validate, persist raw reference, enqueue | response action |
| Normalizer | map vendor payload เข้า internal event | ตัดสินใจ containment |
| Correlator | key/time-window grouping, duplicate control | แก้ raw evidence |
| Case service | lifecycle, task, owner, observable | เรียก provider โดยตรง |
| Workflow engine | execute graph, state, retries, pause | เก็บ secret แบบ plaintext |
| Policy engine | allow/deny/approval/target scope | เปลี่ยน case โดยไม่มี audit |
| Connector runner | invoke provider with bounded permission | เลือก policy เอง |
| Evidence service | artifact metadata, hash, custody reference | overwrite original |
| Notification | send approved status/summary | ส่ง raw secret |
| Scheduler | scheduled jobs and health checks | bypass approval |
| UI/API | analyst interaction and admin operations | เป็น policy bypass |

การสื่อสารระหว่าง component ต้องมี `request_id`, `tenant_id`, `case_id`, `execution_id`, `schema_version` และ `policy_version` เมื่อ field เหล่านี้เกี่ยวข้อง

