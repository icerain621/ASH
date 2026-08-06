#!/usr/bin/env bash
# Verify latest (or ASH_BACKUP_FILE) SQLite backup: sha256 + PRAGMA integrity_check.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

DATA_DIR="${ASH_DATA_DIR:-$ROOT/.ash}"
BACKUP_DIR="${ASH_BACKUP_DIR:-$DATA_DIR/backups}"
FILE="${ASH_BACKUP_FILE:-}"

if [[ -z "$FILE" ]]; then
  if [[ ! -d "$BACKUP_DIR" ]]; then
    echo "backup dir missing: $BACKUP_DIR" >&2
    exit 2
  fi
  FILE="$(ls -1t "$BACKUP_DIR"/ash-*.db 2>/dev/null | head -n1 || true)"
fi

if [[ -z "$FILE" || ! -f "$FILE" ]]; then
  echo "no backup file to verify (set ASH_BACKUP_FILE or run make data-backup)" >&2
  exit 2
fi

# Resolve absolute path before go test (subprocess cwd differs on Windows).
FILE="$(cd "$(dirname "$FILE")" && pwd)/$(basename "$FILE")"

echo "Verifying backup: $FILE"

if [[ -f "$FILE.sha256" ]]; then
  expected="$(awk '{print $1}' "$FILE.sha256")"
  if [[ -z "$expected" ]]; then
    echo "empty checksum in $FILE.sha256" >&2
    exit 1
  fi
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$FILE" | awk '{print $1}')"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$FILE" | awk '{print $1}')"
  else
    echo "warning: no sha256sum/shasum; skip checksum re-verify" >&2
    actual=""
  fi
  if [[ -n "${actual:-}" ]]; then
    if [[ "$expected" != "$actual" ]]; then
      echo "sha256 mismatch: expected=$expected actual=$actual" >&2
      exit 1
    fi
    echo "OK sha256 $actual"
  fi
else
  echo "warning: missing $FILE.sha256; skip checksum re-verify" >&2
fi

export ASH_SQLITE_VERIFY_PATH="$FILE"
go test ./internal/store/ -run '^TestEnvSQLiteIntegrityVerify$' -count=1

echo "OK data-backup-verify: $FILE"
