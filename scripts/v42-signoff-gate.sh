#!/usr/bin/env bash
# v4.2 release sign-off: frozen scope + Stage-1 checks (no git tag).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

# Go on Windows is a native binary. A POSIX /tmp path is ignored and
# falls back to C:\WINDOWS. Prefer a path MSYS will rewrite.
if [[ -z "${ASH_GO_TMP_READY:-}" ]]; then
  if command -v cygpath >/dev/null 2>&1; then
    _ash_base="${LOCALAPPDATA:-${USERPROFILE:-$HOME}}"
    _ash_tmp="$(cygpath -u "$_ash_base")/Temp/ash-go"
  else
    _ash_tmp="${TMPDIR:-/tmp}/ash-go"
  fi
  mkdir -p "$_ash_tmp/gotmp" "$_ash_tmp/gocache"
  export GOTMPDIR="$_ash_tmp/gotmp"
  export GOCACHE="$_ash_tmp/gocache"
  export TMPDIR="$_ash_tmp/gotmp"
  export TMP="$TMPDIR"
  export TEMP="$TMPDIR"
  export ASH_GO_TMP_READY=1
fi

FAIL=0

run_step() {
  local name="$1"
  shift
  echo "==> $name"
  if "$@"; then
    echo "OK $name"
    return 0
  fi
  echo "FAIL $name" >&2
  FAIL=1
  return 0
}

run_step scope-freeze-gate bash scripts/scope-freeze-gate.sh
run_step proto-check bash -c '
  set -euo pipefail
  export GOTOOLCHAIN=auto
  go run github.com/bufbuild/buf/cmd/buf@latest lint proto
  tmp="$(mktemp -d)"
  trap "rm -rf \"$tmp\"" EXIT
  template="$(pwd)/proto/buf.gen.yaml"
  cp -R proto "$tmp/proto"
  rm -f "$tmp"/proto/ash/v1/*.pb.go
  (cd "$tmp" && GOTOOLCHAIN=auto go run github.com/bufbuild/buf/cmd/buf@latest generate proto --template "$template")
  diff -ru proto/ash/v1 "$tmp/proto/ash/v1"
'
run_step openapi-check bash scripts/openapi-check.sh
run_step v42-stage1 go test ./internal/pluginabi/ ./internal/rag/ ./internal/api/ ./internal/config/ -count=1 -run 'TestProductionPluginGRPCListen|TestRegistryServerRejectsUnsigned|TestIndexerServer|TestStartIndexer|TestPluginABIProfile|TestLoadRagIndexer'

SCOPE="$ROOT/doc/plan/v4.2-release-scope.md"
if grep -q '已冻结' "$SCOPE"; then
  echo "OK v4.2-release-scope frozen marker"
else
  echo "FAIL v4.2-release-scope.md missing 已冻结" >&2
  FAIL=1
fi

CHECKLIST="$ROOT/doc/checklists/v4.2-signoff.md"
if [[ ! -f "$CHECKLIST" ]]; then
  echo "FAIL missing $CHECKLIST" >&2
  FAIL=1
else
  echo "OK v4.2-signoff checklist present"
fi

TEMPLATE="$ROOT/doc/evidence/v4.2-signatures-template.md"
if [[ ! -f "$TEMPLATE" ]]; then
  echo "FAIL missing $TEMPLATE" >&2
  FAIL=1
else
  echo "OK v4.2 signatures template present"
fi

echo ""
echo "Tag is NOT created by this gate. After human sign-off:"
echo "  git tag -a v4.2.0 -m \"ASH v4.2.0 Stage-1 plugin grpc production path and in-process RAG indexer contract (DX73–DX78 frozen)\""
echo "  git push origin v4.2.0"

if [[ "$FAIL" -ne 0 ]]; then
  echo "" >&2
  echo "v4.2-signoff FAILED — see doc/checklists/v4.2-signoff.md" >&2
  exit 1
fi
echo "OK v4.2-signoff"
