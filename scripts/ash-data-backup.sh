#!/usr/bin/env bash
# Backup SQLite data dir before Postgres migrate (MVP §6).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DATA_DIR="${ASH_DATA_DIR:-$ROOT/.ash}"
SRC="${ASH_SQLITE_PATH:-$DATA_DIR/ash.db}"
BACKUP_DIR="${ASH_BACKUP_DIR:-$DATA_DIR/backups}"

if [[ ! -f "$SRC" ]]; then
  echo "SQLite db not found: $SRC" >&2
  echo "Nothing to backup; create data with Worker or copy existing .ash/ash.db" >&2
  exit 2
fi

mkdir -p "$BACKUP_DIR"
TS="$(date -u +%Y%m%dT%H%M%SZ)"
DEST="$BACKUP_DIR/ash-${TS}.db"
cp -f "$SRC" "$DEST"

if command -v sha256sum >/dev/null 2>&1; then
  # Basename-only checksum so verify can `cd` into the backup dir safely.
  (cd "$(dirname "$DEST")" && sha256sum "$(basename "$DEST")" | tee "$(basename "$DEST").sha256")
elif command -v shasum >/dev/null 2>&1; then
  (cd "$(dirname "$DEST")" && shasum -a 256 "$(basename "$DEST")" | tee "$(basename "$DEST").sha256")
fi

echo "OK backup: $DEST"

if [[ "${ASH_BACKUP_SKIP_VERIFY:-0}" != "1" ]]; then
  ASH_BACKUP_FILE="$DEST" bash "$ROOT/scripts/ash-data-backup-verify.sh"
fi