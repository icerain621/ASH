#!/usr/bin/env bash
# Smoke: evidence-sha-gate detects stale SHA when required.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

TMP="$(mktemp -d 2>/dev/null || mktemp -d -t ash-evidence-sha)"
trap 'rm -rf "$TMP"' EXIT

STALE="$TMP/stale-latest.md"
cat >"$STALE" <<'EOF'
# stale evidence
- Date：2099-01-01
- Git：0000000 @ main
EOF

# Relative path from ROOT for the gate.
REL="doc/evidence/.ash-evidence-sha-smoke-tmp.md"
mkdir -p "$(dirname "$ROOT/$REL")"
cp "$STALE" "$ROOT/$REL"
trap 'rm -rf "$TMP"; rm -f "$ROOT/$REL"' EXIT

if ASH_EVIDENCE_SHA_FILES="$REL" ASH_REQUIRE_EVIDENCE_SHA=1 bash scripts/evidence-sha-gate.sh; then
  echo "expected evidence-sha-gate to fail on invalid SHA" >&2
  exit 1
fi

# Warn mode must still exit 0 on the same stale file.
ASH_EVIDENCE_SHA_FILES="$REL" ASH_REQUIRE_EVIDENCE_SHA=0 bash scripts/evidence-sha-gate.sh >/dev/null

echo "OK evidence-sha-smoke"
