#!/usr/bin/env bash
# Private signed skill pack smoke (Sprint DX22).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

KEY="${ASH_SKILL_PACK_SIGNING_KEY:-skill-pack-smoke-key}"
export ASH_SKILL_PACK_SIGNING_KEY="$KEY"
export ASH_SKILL_PACK_ALLOWLIST="${ASH_SKILL_PACK_ALLOWLIST:-*}"
export ASH_SKILL_PACK_SPACES="${ASH_SKILL_PACK_SPACES:-*}"

echo "== skills pack unit tests =="
go test ./internal/skills/ -count=1 -run 'TestBuildSignVerifyInstallPack|TestVerifyPackRejects'
go test ./internal/api/ -count=1 -run 'TestSkillPackVerifyAndInstallAPI'

TMP="$(mktemp -d 2>/dev/null || mktemp -d -t ash-skill-pack)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT

mkdir -p "$TMP/skill"
cat >"$TMP/skill/SKILL.md" <<'EOF'
---
name: smoke-pack
description: DX22 skill-pack-smoke fixture
---

# Smoke
EOF

echo "== CLI skill-pack build/sign =="
OUT="$TMP/smoke-pack.zip"
BUILD_OUT="$(go run ./cmd/cli skill-pack build --dir "$TMP/skill" --publisher local --version 0.1.0 --out "$OUT" --key "$KEY")"
echo "$BUILD_OUT"
SIG="$(go run ./cmd/cli skill-pack sign --dir "$TMP/skill" --publisher local --version 0.1.0 --key "$KEY")"
if [[ ${#SIG} -ne 64 ]]; then
  echo "unexpected signature length: ${#SIG}" >&2
  exit 1
fi
echo "signature=$SIG"

echo "== install via package helper =="
# Use a tiny Go program would be heavy; rely on unit tests + CLI artifacts.
test -f "$OUT"

echo "OK skill-pack-smoke"
