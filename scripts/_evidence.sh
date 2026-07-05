#!/usr/bin/env bash
# Shared helpers for acceptance / MVP evidence archiving.
set -euo pipefail

_ash_evidence_root() {
  local script="${BASH_SOURCE[1]:-${BASH_SOURCE[0]}}"
  cd "$(dirname "$script")/.." && pwd
}

ash_evidence_init() {
  local label="${1:-run}"
  local root
  root="$(_ash_evidence_root)"
  local ts
  ts="$(date -u +%Y%m%dT%H%M%SZ)"
  ASH_EVIDENCE_DIR="${ASH_EVIDENCE_DIR:-$root/.ash/evidence/${label}-${ts}}"
  export ASH_EVIDENCE_DIR
  mkdir -p "$ASH_EVIDENCE_DIR"
  {
    echo "label=$label"
    echo "timestamp_utc=$ts"
    echo "host=$(hostname 2>/dev/null || echo unknown)"
    echo "user=${USER:-unknown}"
    echo "repo=$root"
    echo "git_sha=$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo unknown)"
    echo "git_branch=$(git -C "$root" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)"
  } >"$ASH_EVIDENCE_DIR/meta.txt"
  echo "$ASH_EVIDENCE_DIR"
}

ash_evidence_step() {
  local name="$1"
  shift
  local log="$ASH_EVIDENCE_DIR/${name}.log"
  echo "== evidence: $name =="
  set +e
  "$@" 2>&1 | tee "$log"
  local code="${PIPESTATUS[0]}"
  set -e
  if [[ "$code" -ne 0 ]]; then
    echo "FAIL $name (exit $code)" | tee -a "$ASH_EVIDENCE_DIR/failures.txt"
    return "$code"
  fi
  echo "PASS $name" >>"$ASH_EVIDENCE_DIR/pass.txt"
}

ash_evidence_optional_step() {
  local name="$1"
  shift
  if ash_evidence_step "$name" "$@"; then
    return 0
  fi
  echo "SKIP $name (optional)" >>"$ASH_EVIDENCE_DIR/skipped.txt"
  return 0
}

ash_evidence_has_docker_postgres() {
  docker ps --format '{{.Names}}' 2>/dev/null | grep -qx ash-postgres-dev
}
