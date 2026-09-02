#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
python3 - "$root_dir" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
files = sorted(root.glob("schemas/*.json")) + sorted(root.glob("examples/*.json"))
for path in files:
    with path.open(encoding="utf-8") as handle:
        json.load(handle)
    print(f"valid json: {path.relative_to(root)}")
print(f"validated {len(files)} JSON files")

def required(schema_name):
    return set(json.loads((root / "schemas" / schema_name).read_text(encoding="utf-8")).get("required", []))

def load_example(name):
    return json.loads((root / "examples" / name).read_text(encoding="utf-8"))

checks = {
    "approval-request.json": ("action.schema.json", load_example("approval-request.json")),
    "case.json": ("case.schema.json", load_example("case.json")),
    "normalized-event.json": ("event.schema.json", load_example("normalized-event.json")),
    "evidence.json": ("evidence.schema.json", load_example("evidence.json")),
    "alert.json": ("alert.schema.json", load_example("alert.json")),
    "policy.default.json": ("policy.schema.json", load_example("policy.default.json")),
}
for example_name, (schema_name, instance) in checks.items():
    missing = required(schema_name) - set(instance)
    if missing:
        raise SystemExit(f"{example_name}: missing schema-required fields: {', '.join(sorted(missing))}")
    print(f"example contract: {example_name} matches required fields in {schema_name}")

connector_required = required("connector.schema.json")
connector_examples = sorted(root.glob("examples/*-connector.json"))
if not connector_examples:
    raise SystemExit("no examples/*-connector.json files found")
for path in connector_examples:
    instance = json.loads(path.read_text(encoding="utf-8"))
    missing = connector_required - set(instance)
    if missing:
        raise SystemExit(f"{path.name}: missing schema-required fields: {', '.join(sorted(missing))}")
    print(f"example contract: {path.name} matches required fields in connector.schema.json")

for example_name in ("normalized-event.json", "wazuh-alert.json"):
    instance = load_example(example_name)
    for observable in instance.get("observables", []):
        missing = required("observable.schema.json") - set(observable)
        if missing:
            raise SystemExit(f"{example_name}: observable missing fields: {', '.join(sorted(missing))}")
print("nested observable contracts: ok")
PY
