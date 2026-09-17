# Evidence model

The filesystem paths below are relative to the `soar-platform/` project root.

evidence แบ่งเป็น raw reference, derived artifact, analyst observation และ external report

## Required metadata

- evidence ID และ case ID
- artifact type, source, acquisition method และ collector
- original path/reference โดยไม่แก้ต้นฉบับ
- SHA-256, size, created/received time และ timezone
- classification/sensitivity และ retention
- parent/derived-from relationship
- custody events: collected, transferred, analyzed, exported
- tool name/version และ command reference
- analyst, purpose และ limitations

สำหรับ workspace `dfir/` ให้เก็บ original ใน `../dfir/evidence/<case>/` แบบ read-only, derived output ใน `../dfir/cases/<case>/work/` และ delivery output ใน `../dfir/exports/<case>/` ตาม `dfir/README.md` ของ workspace `dfir/` ซึ่งอยู่นอก repository นี้
