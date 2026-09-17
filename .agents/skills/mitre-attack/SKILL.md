---
name: mitre-attack
description: "Query MITRE ATT&CK techniques, groups, software, and mitigations. Requires a one-time ~45 MB download of enterprise-attack.json, which is NOT bundled — see the Dataset section before the first query."
user-invocable: false
metadata:
  version: 1.0.0
---

# MITRE ATT&CK Local Reference

## Dataset

**The dataset is not bundled with this skill.** This install contains `SKILL.md`
only — there is no `enterprise-attack.json` and no `scripts/download-mitre.sh`
(the script an earlier version of this file told you to run first). Every query
below fails until you fetch the bundle once:

```bash
# from the repository root
curl -fsSL https://raw.githubusercontent.com/mitre/cti/master/enterprise-attack/enterprise-attack.json \
  -o .agents/skills/mitre-attack/enterprise-attack.json
```

That is a ~45 MB STIX 2.1 bundle containing every Enterprise ATT&CK object. It
is listed in `.gitignore`: it is a redistributable upstream artifact that would
otherwise add 45 MB to every clone, and it is refreshed by re-running the
command above rather than by a commit.

Check before querying, and say the dataset is missing rather than answering
from recollection — ATT&CK technique IDs and their mappings are exactly the
kind of detail that is easy to state confidently and get wrong:

```bash
test -f .agents/skills/mitre-attack/enterprise-attack.json && echo present || echo MISSING
```

The query examples below use the path `mitre-attack/enterprise-attack.json`,
which assumes the skills directory is the working directory. From the
repository root, prefix them with `.agents/skills/`.

## Querying the Dataset

The dataset is a JSON file with a `objects` array. Each object has a `type` field.

### Find a technique by ID
```bash
cat mitre-attack/enterprise-attack.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
tid = 'T1566'  # Change as needed
for obj in data['objects']:
    refs = obj.get('external_references', [])
    for ref in refs:
        if ref.get('external_id') == tid:
            print(json.dumps(obj, indent=2))
            break
"
```

### List all techniques for a tactic
Tactics are mapped via `kill_chain_phases[].phase_name`:
- `reconnaissance`, `resource-development`, `initial-access`, `execution`, `persistence`
- `privilege-escalation`, `defense-evasion`, `credential-access`, `discovery`
- `lateral-movement`, `collection`, `command-and-control`, `exfiltration`, `impact`

### Find a threat group
```bash
# Groups have type "intrusion-set"
cat mitre-attack/enterprise-attack.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
name = 'APT28'  # Change as needed
for obj in data['objects']:
    if obj.get('type') == 'intrusion-set':
        aliases = obj.get('aliases', [])
        if name in aliases or obj.get('name') == name:
            print(json.dumps(obj, indent=2))
"
```

### Map group to techniques
Groups link to techniques via `relationship` objects with `relationship_type: "uses"`.

### Find software/malware
Software objects have type `malware` or `tool`.

## Key Object Types

| Type | ATT&CK Concept | Key Fields |
|------|----------------|-----------|
| `attack-pattern` | Technique | name, description, kill_chain_phases, external_references (technique ID) |
| `intrusion-set` | Group | name, aliases, description |
| `malware` | Malware | name, description, labels |
| `tool` | Tool | name, description |
| `course-of-action` | Mitigation | name, description |
| `relationship` | Links objects | source_ref, target_ref, relationship_type |
| `x-mitre-tactic` | Tactic | name, x_mitre_shortname |

## Common ATT&CK Technique References

| ID | Name | Common Use |
|----|------|-----------|
| T1566 | Phishing | Initial access via email |
| T1566.001 | Spearphishing Attachment | Malicious attachment |
| T1566.002 | Spearphishing Link | Malicious URL |
| T1059 | Command and Scripting Interpreter | Execution via scripts |
| T1059.001 | PowerShell | PowerShell execution |
| T1078 | Valid Accounts | Using legitimate credentials |
| T1071 | Application Layer Protocol | C2 over HTTP/HTTPS/DNS |
| T1055 | Process Injection | Code injection for evasion |
| T1053 | Scheduled Task/Job | Persistence via scheduled tasks |
| T1547 | Boot or Logon Autostart | Persistence via autostart |
| T1562 | Impair Defenses | Disabling security tools |
| T1486 | Data Encrypted for Impact | Ransomware encryption |
| T1190 | Exploit Public-Facing Application | Vulnerability exploitation |

## Usage in Analysis
- Map observed TTPs to ATT&CK technique IDs
- Cross-reference threat actor profiles with known ATT&CK groups
- Identify detection coverage gaps by comparing ATT&CK heatmap to detection rules
- Use ATT&CK Navigator for visual TTP mapping
