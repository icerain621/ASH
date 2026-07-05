#!/usr/bin/env bash
# H-09 smoke: static §7 api tests + optional live Worker sampling.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

bash scripts/release-sampling-static.sh

BASE="${ASH_WORKER_URL:-}"
if [[ -n "$BASE" ]]; then
  echo "== H-09 live release sampling @ ${BASE} =="
  bash scripts/release-sampling.sh
else
  echo "== H-09 live sampling skipped (set ASH_WORKER_URL) =="
fi

echo "OK H-09 release sampling smoke"
