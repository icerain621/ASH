#!/usr/bin/env bash
# DX65: device mint UI markers + AuthSession device mint API smoke evidence.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/device-session-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-device-session)"

STATUS="pass"
RC=0

echo "== DX65 AuthSession device mint/list/revoke tests =="
set +e
go test ./internal/api/ -count=1 -run 'TestAuthSessionDeviceMintListAndRevoke|TestAuthSessionScopeEnforcement|TestAuthSessionJtiReplayAndGrace' 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

OPENAPI_OK="pass"
if ! grep -q '/api/v1/auth/sessions/device' "$ROOT/doc/api/openapi-ash-v1.yaml" 2>/dev/null; then
  OPENAPI_OK="fail"
  STATUS="fail"
  RC=1
fi

FE_OK="pass"
if ! grep -q 'device-mint-form' "$ROOT/frontend/src/pages/SpacePage.tsx"; then
  FE_OK="fail"
  STATUS="fail"
  RC=1
fi
if ! grep -q 'device-mint-result' "$ROOT/frontend/src/pages/SpacePage.tsx"; then
  FE_OK="fail"
  STATUS="fail"
  RC=1
fi
if ! grep -q 'createDeviceAuthSession' "$ROOT/frontend/src/modules/platform/api/platform.api.ts"; then
  FE_OK="fail"
  STATUS="fail"
  RC=1
fi

{
  echo "# Device session smoke (DX65)"
  echo
  echo "- stamp: \`$STAMP\`"
  echo "- host: \`$OS_NAME\`"
  echo "- status: **$STATUS**"
  echo "- openapi device path: **$OPENAPI_OK**"
  echo "- frontend mint UI: **$FE_OK**"
  echo
  echo "## Checks"
  echo
  echo "| Check | Result |"
  echo "|-------|--------|"
  echo "| go test DeviceMint / Scope / Jti | $([[ $RC -eq 0 ]] && echo pass || echo fail) |"
  echo "| OpenAPI /auth/sessions/device | $OPENAPI_OK |"
  echo "| SpacePage mint form + API client | $FE_OK |"
  echo
  echo "## Test excerpt"
  echo
  echo '```'
  tail -n 50 "$LOG" || true
  echo '```'
} >"$EVIDENCE"

echo "Wrote $EVIDENCE (status=$STATUS)"
rm -f "$LOG"
exit "$RC"
