#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
python3 - "$root_dir" <<'PY'
import pathlib
import sys
import yaml

root = pathlib.Path(sys.argv[1])
required = {
    "id", "name", "version", "owner", "type", "status", "references", "trigger",
    "scope", "preconditions", "steps", "safety", "evidence", "metrics", "tests",
}
allowed_status = {"draft", "review", "lab_tested", "staged", "approved", "active", "deprecated"}
allowed_kinds = {"validate", "deduplicate", "investigate", "enrich", "decision", "approval", "action", "verify", "evidence", "notify", "escalate", "close"}
files = sorted(root.glob("playbooks/*.yaml")) + sorted(root.glob("playbooks/templates/*.yaml"))
for path in files:
    data = yaml.safe_load(path.read_text(encoding="utf-8"))
    if not isinstance(data, dict):
        raise SystemExit(f"{path}: YAML root must be an object")
    missing = required - set(data)
    if missing:
        raise SystemExit(f"{path}: missing top-level keys: {', '.join(sorted(missing))}")
    if data["status"] not in allowed_status:
        raise SystemExit(f"{path}: invalid status: {data['status']}")
    if not isinstance(data["references"], list) or not data["references"]:
        raise SystemExit(f"{path}: references must be a non-empty list")
    if not isinstance(data["evidence"], list) or not data["evidence"]:
        raise SystemExit(f"{path}: evidence must be a non-empty list")
    if not isinstance(data["metrics"], list) or not data["metrics"]:
        raise SystemExit(f"{path}: metrics must be a non-empty list")
    if not isinstance(data["steps"], list) or not data["steps"]:
        raise SystemExit(f"{path}: steps must be a non-empty list")
    for step in data["steps"]:
        if not isinstance(step, dict) or not {"id", "kind", "purpose"} <= set(step):
            raise SystemExit(f"{path}: each step needs id, kind and purpose")
        if step["kind"] not in allowed_kinds:
            raise SystemExit(f"{path}: invalid step kind: {step['kind']}")
    if data["safety"].get("default_mode") != "dry_run":
        raise SystemExit(f"{path}: missing safe default mode")
    if not data["safety"].get("stop_conditions"):
        raise SystemExit(f"{path}: missing stop conditions")
    automation = data.get("automation")
    if automation is not None:
        required_automation = {"signature_id", "match", "threshold", "window_seconds", "action_intent"}
        missing_automation = required_automation - set(automation)
        if missing_automation:
            raise SystemExit(f"{path}: automation missing keys: {', '.join(sorted(missing_automation))}")
        if not isinstance(automation["match"], dict) or "event_class" not in automation["match"]:
            raise SystemExit(f"{path}: automation.match must include event_class")
    print(f"playbook contract: {path.relative_to(root)}")
print(f"validated {len(files)} playbooks")
PY
