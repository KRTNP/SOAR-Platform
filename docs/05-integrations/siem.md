# SIEM integration

SIEM connector ต้องรองรับ query แบบ bounded time range, pagination, result count, timeout และ query audit

ห้ามให้ playbook สร้าง query จาก string ที่ไม่ validate โดยตรง ควรใช้ parameterized query template และจำกัด index/data source ตาม tenant

output ควรเป็น normalized sightings พร้อม source event reference, query ID, query hash, time range และ sample count

