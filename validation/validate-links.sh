#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
python3 - "$root_dir" <<'PY'
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
pattern = re.compile(r"\[[^\]]+\]\(([^)]+)\)")
failed = False
count = 0
for source in sorted(root.rglob("*.md")):
    relative_source = source.relative_to(root)
    # Skills are vendored agent dependencies. Their upstream-relative links
    # may point to skills that are intentionally not installed in this repo.
    if relative_source.parts[:2] == (".agents", "skills"):
        continue
    # node_modules/.next are installed/build artifacts (gitignored, never
    # part of this repo's own content) that ship their own markdown with
    # upstream-relative links this repo has no reason to satisfy.
    if "node_modules" in relative_source.parts or ".next" in relative_source.parts:
        continue
    for raw_target in pattern.findall(source.read_text(encoding="utf-8")):
        target = raw_target.split("#", 1)[0].strip()
        if not target or target.startswith(("http://", "https://", "mailto:")):
            continue
        count += 1
        path = pathlib.Path(target) if target.startswith("/") else source.parent / target
        if not path.exists():
            print(f"{source.relative_to(root)}: missing link target: {target}", file=sys.stderr)
            failed = True
if failed:
    raise SystemExit(1)
print(f"markdown local links: ok ({count} checked)")
PY
