# Relationship and correlation

## Correlation keys

- same tenant + host
- same user/account
- same source/destination IP
- same domain/url/hash
- same process lineage
- same message/thread ID
- same CVE + asset group

## Correlation controls

- ทุก correlation มี bounded time window
- ระบุ whether key equality, fuzzy match หรือ graph relationship
- ตั้ง maximum group size และ TTL
- preserve original alert; correlation ไม่ overwrite source event
- merge case ต้องมี reason และสามารถ split กลับเป็น child case ได้
- late event ต้อง update timeline โดยไม่เปลี่ยน historical decision เงียบ ๆ

