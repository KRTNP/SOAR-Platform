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
  if rg -n '(^|[[:space:]])(TODO|TBD)([[:space:]]|$)' "$diagram" >/dev/null; then
    printf 'placeholder found in diagram: %s\n' "$diagram" >&2
    exit 1
  fi
  ((count+=1))
  printf 'diagram source: %s\n' "${diagram#"$root_dir"/}"
done
printf 'validated %s Mermaid sources\n' "$count"
