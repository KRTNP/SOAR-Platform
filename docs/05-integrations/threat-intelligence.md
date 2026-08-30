# Threat intelligence integration

รองรับ STIX/TAXII, provider REST API และ local indicator store

ผล enrichment ต้องเก็บ provider, query time, verdict, confidence, source reliability, sighting time, expiration และ raw reference

ไม่ควรเปลี่ยน verdict ของ provider หนึ่งให้เป็น block decision ทันที ต้องรวมกับ asset context, internal sightings, allowlist และ detection evidence

