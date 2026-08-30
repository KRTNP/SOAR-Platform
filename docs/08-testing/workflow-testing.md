# Workflow testing

ใช้ fixture ที่ครอบคลุม happy path, branch true/false, missing field, duplicate, partial enrichment, approval reject/expire, worker restart และ provider outage

แต่ละ workflow ต้องมี expected terminal state และไม่อนุญาตให้ path ใดจบแบบเงียบ ๆ โดยไม่มี case task หรือ failure reason

