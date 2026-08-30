#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
python3 - "$root_dir" <<'PY'
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
catalog = (root / "docs/00-context/source-catalog.md").read_text(encoding="utf-8")
trace = (root / "docs/09-governance/source-to-design-traceability.md").read_text(encoding="utf-8")
ids = set(re.findall(r"\| (SRC-[A-Z0-9-]+) \|", catalog))
if len(ids) < 15:
    raise SystemExit(f"source catalog has too few registered IDs: {len(ids)}")
if catalog.count("https://") < 15:
    raise SystemExit("source catalog is missing canonical URLs")
for source_id in ids:
    if source_id not in trace:
        raise SystemExit(f"source {source_id} is not referenced by traceability matrix")

derived = {"cisa-major-incident-coordination.yaml", "cisa-vulnerability-response.yaml", "nist-recovery.yaml", "dfir-evidence-preservation.yaml"}
for name in derived:
    path = root / "playbooks" / name
    if not path.exists():
        raise SystemExit(f"missing source-derived playbook: {name}")
    text = path.read_text(encoding="utf-8")
    if "source_basis:" not in text or "adaptation:" not in text:
        raise SystemExit(f"{path}: source_basis/adaptation missing")
    referenced = set(re.findall(r"source_id: (SRC-[A-Z0-9-]+)", text))
    missing = referenced - ids
    if missing:
        raise SystemExit(f"{path}: unknown source IDs: {', '.join(sorted(missing))}")
    if "default_mode: dry_run" not in text or "stop_conditions:" not in text:
        raise SystemExit(f"{path}: safety contract missing")
    print(f"traceability contract: {path.relative_to(root)}")
print(f"validated {len(ids)} source IDs and {len(derived)} source-derived playbooks")
PY
