# Firewall integration

ให้ model action เป็น intent เช่น `block_network_indicator` ไม่ให้ playbook สร้าง raw firewall rule เอง

ก่อน block ต้องตรวจ:

- indicator type และ direction
- allowlist/exception
- source/destination scope
- expiration/TTL
- duplicate existing rule
- business criticality
- approval requirement

หลัง block ให้ query rule state และเก็บ rule ID, previous state, expiration และ rollback reference

## Lab profile

The first real write-capable provider is an OPNsense lab VM. SOAR may only add or
remove an IP in the dedicated `SOAR_TEMP_BLOCK` alias through a typed connector
intent. The action requires one incident-commander approval, has a ten-minute TTL,
verifies both alias membership and blocked SSH traffic, and retries cleanup at 5, 15
and 30 seconds before escalating with the block still active.

รายละเอียดของ connector, network boundary และ acceptance sequence อยู่ใน [lab integration profile](lab-integration-profile.md) และ [SSH brute-force lab scenario](../08-testing/lab-scenario-ssh-bruteforce.md)
