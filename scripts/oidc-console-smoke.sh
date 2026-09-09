#!/usr/bin/env bash
# DX59: console auth entry + session API smoke evidence (no live IdP required).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

EVIDENCE="$ROOT/doc/evidence/oidc-console-smoke-latest.md"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
OS_NAME="$(uname -s 2>/dev/null || echo unknown)"
LOG="$(mktemp 2>/dev/null || mktemp -t ash-oidc-console)"

STATUS="pass"
RC=0

echo "== DX59 OIDC / AuthSession backend tests =="
set +e
go test ./internal/api/ ./internal/idp/ -count=1 -run 'OIDC|AuthSession|ClientExchange' 2>&1 | tee "$LOG"
RC=${PIPESTATUS[0]}
set -e
if [[ "$RC" -ne 0 ]]; then
  STATUS="fail"
fi

OPENAPI_OK="pass"
if ! grep -q 'name: ui' "$ROOT/doc/api/openapi-ash-v1.yaml" 2>/dev/null; then
  # ui query on oidc login
  if ! grep -q 'ui=1' "$ROOT/doc/api/openapi-ash-v1.yaml"; then
    OPENAPI_OK="fail"
    STATUS="fail"
    RC=1
  fi
fi
if ! grep -q '/api/v1/auth/sessions/refresh' "$ROOT/doc/api/openapi-ash-v1.yaml"; then
  OPENAPI_OK="fail"
  STATUS="fail"
  RC=1
fi

FE_OK="pass"
for f in \
  "$ROOT/frontend/src/pages/LoginPage.tsx" \
  "$ROOT/frontend/src/modules/platform/auth/loginHash.ts" \
  "$ROOT/frontend/src/app/router.tsx"
do
  if [[ ! -f "$f" ]]; then
    FE_OK="fail"
    STATUS="fail"
    RC=1
  fi
done
if ! grep -q 'path: "/login"' "$ROOT/frontend/src/app/router.tsx"; then
  FE_OK="fail"
  STATUS="fail"
  RC=1
fi
if ! grep -q 'auth-sessions-panel' "$ROOT/frontend/src/pages/SpacePage.tsx"; then
  FE_OK="fail"
  STATUS="fail"
  RC=1
fi

LIVE_STATUS="skipped"
LIVE_NOTE="set ASH_OIDC_ENABLED=1 + IdP env for live browser OIDC (optional)"

{
  echo "# OIDC / console auth smoke (DX59)"
  echo
  echo "- stamp: \`$STAMP\`"
  echo "- host: \`$OS_NAME\`"
  echo "- status: **$STATUS**"
  echo "- openapi markers: **$OPENAPI_OK**"
  echo "- frontend markers: **$FE_OK**"
  echo "- live IdP: **$LIVE_STATUS** — $LIVE_NOTE"
  echo
  echo "## Checks"
  echo
  echo "| Check | Result |"
  echo "|-------|--------|"
  echo "| go test OIDC/AuthSession/idp | $([[ $RC -eq 0 ]] && echo pass || echo fail) |"
  echo "| OpenAPI ui + sessions/refresh | $OPENAPI_OK |"
  echo "| LoginPage + /ui/login + Sessions panel | $FE_OK |"
  echo
  echo "## Test excerpt"
  echo
  echo '```'
  tail -n 40 "$LOG" || true
  echo '```'
} >"$EVIDENCE"

echo "Wrote $EVIDENCE (status=$STATUS)"
rm -f "$LOG"
exit "$RC"
