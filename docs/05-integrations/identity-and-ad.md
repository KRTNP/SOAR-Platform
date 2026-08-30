# Identity and Active Directory integration

investigation ควรตรวจ sign-in, MFA, device, privilege, token, mailbox rule และ group change

response ที่มีความเสี่ยง:

- revoke session/token
- force credential reset
- disable account
- remove privilege/group membership

ต้องแยก service account, break-glass account และ privileged account ออกจาก policy ปกติ และห้าม disable บัญชีฉุกเฉินโดยใช้ heuristic เดียว

