# Observable model

observable รองรับชนิดมาตรฐานและ context ที่ใช้ค้น/ตอบสนอง:

| Type | ตัวอย่าง context |
|---|---|
| ipv4/ipv6 | direction, port, ASN, first_seen, last_seen |
| domain/url | registrar, DNS, path, reputation |
| hash | algorithm, file name, signer, malware verdict |
| email | message ID, sender, recipient, attachment |
| user/account | tenant, role, MFA, sign-in history |
| host/device | asset ID, owner, criticality, network zone |
| process/file | parent, command line, path, hash |
| vulnerability | CVE, asset, version, exploit evidence |

ทุก observable ต้องมี `value`, `type`, `normalization`, `first_seen`, `last_seen`, `confidence`, `source_refs` และ `sensitivity`

