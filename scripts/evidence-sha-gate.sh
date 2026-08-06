#!/usr/bin/env bash
# Check doc/evidence/*-latest.md Git SHA vs HEAD (证据一致性).
# Default: WARN on mismatch, exit 0.
# ASH_REQUIRE_EVIDENCE_SHA=1 → mismatch/missing/invalid SHA fails the gate.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

HEAD_SHORT="$(git rev-parse --short HEAD)"
REQUIRE="${ASH_REQUIRE_EVIDENCE_SHA:-0}"
FAIL=0
WARN=0

# Operational evidence that should track recent gates (not process signoff freeze).
DEFAULT_FILES=(
  "doc/evidence/release-window-latest.md"
  "doc/evidence/cloud-h01-h03-latest.md"
  "doc/evidence/rollback-drill-latest.md"
)

if [[ -n "${ASH_EVIDENCE_SHA_FILES:-}" ]]; then
  # shellcheck disable=SC2206
  FILES=(${ASH_EVIDENCE_SHA_FILES})
else
  FILES=("${DEFAULT_FILES[@]}")
fi

fail_or_warn() {
  local msg="$1"
  if [[ "$REQUIRE" == "1" ]]; then
    echo "FAIL $msg" >&2
    FAIL=1
  else
    echo "WARN $msg" >&2
    WARN=1
  fi
}

for rel in "${FILES[@]}"; do
  file="$ROOT/$rel"
  if [[ ! -f "$file" ]]; then
    echo "SKIP $rel (missing)"
    continue
  fi
  line="$(grep -E '^- Git：' "$file" | head -1 || true)"
  if [[ -z "$line" ]]; then
    fail_or_warn "$rel: missing '- Git：' line"
    continue
  fi
  # e.g. "- Git：852726b @ main"
  sha="$(echo "$line" | sed -E 's/^- Git：([0-9a-fA-F]+).*/\1/')"
  if [[ -z "$sha" || "$sha" == "$line" ]]; then
    fail_or_warn "$rel: could not parse Git SHA from: $line"
    continue
  fi
  if ! git cat-file -e "${sha}^{commit}" 2>/dev/null; then
    fail_or_warn "$rel: Git SHA $sha is not a commit in this repo"
    continue
  fi
  if [[ "$sha" == "$HEAD_SHORT" ]] || [[ "$(git rev-parse --short "$sha")" == "$HEAD_SHORT" ]]; then
    echo "OK $rel @ $sha (matches HEAD)"
    continue
  fi
  if git merge-base --is-ancestor "$sha" HEAD 2>/dev/null; then
    fail_or_warn "$rel: Git $sha is behind HEAD $HEAD_SHORT (refresh evidence)"
  else
    fail_or_warn "$rel: Git $sha is not an ancestor of HEAD $HEAD_SHORT"
  fi
done

if [[ "$FAIL" -ne 0 ]]; then
  echo "evidence-sha-gate FAILED (ASH_REQUIRE_EVIDENCE_SHA=1)" >&2
  exit 1
fi
if [[ "$WARN" -ne 0 ]]; then
  echo "OK evidence-sha-gate (warnings only; set ASH_REQUIRE_EVIDENCE_SHA=1 to hard-fail)"
else
  echo "OK evidence-sha-gate"
fi
