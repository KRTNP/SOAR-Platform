#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
count=0
for diagram in "$root_dir"/docs/diagrams/*.mmd; do
  [[ -f "$diagram" ]] || continue
  first_line="$(sed -n '1p' "$diagram")"
  case "$first_line" in
    flowchart\ *|sequenceDiagram|stateDiagram-v2) ;;
    *) printf 'invalid Mermaid directive: %s\n' "$diagram" >&2; exit 1 ;;
  esac
  # grep, not rg. `rg` is not installed on a stock CI runner, and because this
  # test sits inside an `if`, a missing binary is not caught by `set -e` — the
  # check silently passed and the script still printed "validated N sources".
  if grep -Eq '(^|[[:space:]])(TODO|TBD)([[:space:]]|$)' "$diagram"; then
    printf 'placeholder found in diagram: %s\n' "$diagram" >&2
    exit 1
  fi
  ((count+=1))
  printf 'diagram source: %s\n' "${diagram#"$root_dir"/}"
done
printf 'validated %s Mermaid sources\n' "$count"
