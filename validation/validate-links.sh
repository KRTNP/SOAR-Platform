#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
python3 - "$root_dir" <<'PY'
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1])
# Two destination forms. The bare form stops at the first ")", which cannot
# express a path containing parentheses — and Next.js route groups
# (web/app/(app)/...) are exactly that. CommonMark's angle-bracket form
# <dest> is how such a destination is written, so both are accepted here.
pattern = re.compile(r"\[[^\]]+\]\(\s*(?:<([^>]*)>|([^)\s]*))\s*\)")
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
    for bracketed, bare in pattern.findall(source.read_text(encoding="utf-8")):
        target = (bracketed or bare).split("#", 1)[0].strip()
        if not target or target.startswith(("http://", "https://", "mailto:")):
            continue
        count += 1
        path = pathlib.Path(target) if target.startswith("/") else source.parent / target
        # A link that climbs out of the repository resolves only on a machine
        # that happens to have the sibling checkout. It passed here and broke
        # in CI, which is the wrong order to find out — so escaping the root
        # is an error even when the target exists locally.
        if not path.resolve().is_relative_to(root.resolve()):
            print(f"{relative_source}: link escapes the repository: {target}", file=sys.stderr)
            failed = True
            continue
        if not path.exists():
            print(f"{source.relative_to(root)}: missing link target: {target}", file=sys.stderr)
            failed = True
if failed:
    raise SystemExit(1)
print(f"markdown local links: ok ({count} checked)")
PY
