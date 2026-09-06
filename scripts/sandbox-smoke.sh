#!/usr/bin/env bash
# Sprint DX / DX18 / DX21 / DX27: sandbox policy + process + landlock (+ optional e2e).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== sandbox unit tests =="
go test ./internal/sandbox/ ./internal/sandbox/process/ ./internal/sandbox/docker/ ./internal/sandbox/landlock/ -count=1
go test ./internal/runs/ -run 'TestSandboxDeniesDangerWhenProfileOff|TestHarnessLoopEmitsRoutedAndCompleted' -count=1

if [[ "${ASH_SANDBOX_LANDLOCK:-}" == "0" ]]; then
  echo "== landlock opted out (ASH_SANDBOX_LANDLOCK=0) =="
elif [[ "${ASH_SANDBOX_LANDLOCK:-}" == "1" ]] || [[ -z "${ASH_SANDBOX_LANDLOCK:-}" ]]; then
  echo "== landlock preference (default-on unless ASH_SANDBOX_LANDLOCK=0) =="
  os_name="$(uname -s 2>/dev/null || echo unknown)"
  if [[ "$os_name" != "Linux" ]]; then
    echo "landlock: skipped on ${os_name} (non-Linux; no e2e)"
  else
    ll_probe="$(mktemp -d 2>/dev/null || mktemp -d -t ash-ll)"
    cat >"$ll_probe/main.go" <<'GOEOF'
package main

import (
	"fmt"

	"github.com/ash-repwiki/ash/internal/sandbox/landlock"
)

func main() {
	fmt.Printf("available=%v seccomp=%v\n", landlock.Available(), landlock.SeccompAvailable())
}
GOEOF
    avail="$(cd "$ll_probe" && go run -mod=readonly . 2>/dev/null || echo "available=false seccomp=false")"
    rm -rf "$ll_probe"
    echo "landlock probe: ${avail}"
    go test ./internal/sandbox/landlock/ -count=1
  fi
fi

# DX27: optional Linux Landlock+seccomp behavioral evidence
if [[ "${ASH_SANDBOX_E2E:-}" == "1" ]]; then
  echo "== landlock/seccomp e2e (ASH_SANDBOX_E2E=1) =="
  os_name="$(uname -s 2>/dev/null || echo unknown)"
  evidence="$ROOT/doc/evidence/sandbox-landlock-e2e-latest.md"
  stamp="$(date -u +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u)"
  if [[ "$os_name" != "Linux" ]]; then
    cat >"$evidence" <<EOF
# Sandbox Landlock + seccomp e2e evidence (DX27)

| Field | Value |
|-------|--------|
| Status | **skipped** (non-Linux host) |
| Platform | ${os_name} |
| Date | ${stamp} |
| Note | Run on Linux CI/host with \`ASH_SANDBOX_E2E=1 make sandbox-smoke\` |

E2B / remote microVM client: **Out** (v2.7). Extension point: \`sandbox.Executor\`.
EOF
    echo "e2e: skipped on ${os_name}; wrote ${evidence}"
  else
    e2e_log="$(mktemp 2>/dev/null || mktemp -t ash-e2e)"
    set +e
    go test ./internal/sandbox/landlock/ -count=1 -v -run 'TestE2E' 2>&1 | tee "$e2e_log"
    e2e_rc=${PIPESTATUS[0]}
    set -e
    allow_st="$(grep -E '--- (PASS|SKIP|FAIL): TestE2ELandlockAllowsRepoRootRead' "$e2e_log" | tail -1 || true)"
    deny_st="$(grep -E '--- (PASS|SKIP|FAIL): TestE2ELandlockDeniesOutsideRepoRoot' "$e2e_log" | tail -1 || true)"
    sec_st="$(grep -E '--- (PASS|SKIP|FAIL): TestE2ESeccompDeniesMountSyscall' "$e2e_log" | tail -1 || true)"
    status="pass"
    if [[ "$e2e_rc" -ne 0 ]]; then
      status="fail"
    fi
    cat >"$evidence" <<EOF
# Sandbox Landlock + seccomp e2e evidence (DX27)

| Field | Value |
|-------|--------|
| Status | **${status}** (exit ${e2e_rc}) |
| Platform | ${os_name} |
| Date | ${stamp} |
| TestE2ELandlockAllowsRepoRootRead | ${allow_st:-"(not run)"} |
| TestE2ELandlockDeniesOutsideRepoRoot | ${deny_st:-"(not run)"} |
| TestE2ESeccompDeniesMountSyscall | ${sec_st:-"(not run)"} |

E2B / remote microVM client: **Out** (v2.7). Extension point: \`sandbox.Executor\`.

## Raw excerpt

\`\`\`
$(tail -n 80 "$e2e_log")
\`\`\`
EOF
    rm -f "$e2e_log"
    echo "e2e: wrote ${evidence}"
    if [[ "$e2e_rc" -ne 0 ]]; then
      exit "$e2e_rc"
    fi
  fi
fi

if [[ "${ASH_SKIP_SANDBOX:-}" != "" ]]; then
  echo "== docker sandbox skipped (ASH_SKIP_SANDBOX set) =="
  echo "OK sandbox-smoke"
  exit 0
fi

if ! command -v docker >/dev/null 2>&1; then
  echo "== docker CLI missing; process-only OK =="
  echo "OK sandbox-smoke"
  exit 0
fi

if ! docker info >/dev/null 2>&1; then
  echo "== docker daemon unavailable; process-only OK =="
  echo "OK sandbox-smoke"
  exit 0
fi

IMAGE="${ASH_SANDBOX_IMAGE:-ash-sandbox-runner:dev}"
echo "== build sandbox image ${IMAGE} =="
docker build -t "$IMAGE" "$ROOT/deploy/sandbox"

TMP="$(mktemp -d 2>/dev/null || mktemp -d -t ash-sbx)"
cleanup() { rm -rf "$TMP"; }
trap cleanup EXIT
echo 'hello-from-docker' >"$TMP/marker.txt"

echo "== docker run echo in workspace =="
out="$(docker run --rm --network none -v "$TMP:/workspace:rw" -w /workspace "$IMAGE" cat marker.txt)"
if [[ "$out" != *hello-from-docker* ]]; then
  echo "docker smoke failed: $out" >&2
  exit 1
fi

echo "OK sandbox-smoke"
