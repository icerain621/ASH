#!/usr/bin/env bash
# Sprint DX: sandbox policy + process executor smoke (Docker optional).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

echo "== sandbox unit tests =="
go test ./internal/sandbox/ ./internal/sandbox/process/ ./internal/sandbox/docker/ -count=1
go test ./internal/runs/ -run 'TestSandboxDeniesDangerWhenProfileOff|TestHarnessLoopEmitsRoutedAndCompleted' -count=1

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
