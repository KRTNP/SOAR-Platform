# Ownership and RACI

| Activity | SOC | IR lead | Platform | DFIR | Service owner | Security governance |
|---|---|---|---|---|---|---|
| detection rule | R | C | C | C | I | A |
| investigation playbook | R | A | C | C | I | C |
| high-impact response | C | A | R | C | C | I |
| evidence collection | C | A | R | R | I | C |
| connector credential | I | I | R | I | C | A |
| playbook promotion | C | A | R | C | C | A |
| post-incident review | R | A | C | R | C | C |

R = Responsible, A = Accountable, C = Consulted, I = Informed; production deployment ต้องกำหนดบุคคล/ทีมจริงใน environment configuration ไม่ฝังชื่อใน playbook

