#!/usr/bin/env bash
# Smoke: relative backup path must survive sha256 verify (release-window / pre-migrate).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

bash scripts/bootstrap-local-ash-db.sh >/dev/null

# Intentionally relative ASH_BACKUP_DIR (the path shape that broke sha256 -c).
ASH_BACKUP_DIR=.ash/backups ASH_BACKUP_SKIP_VERIFY=0 bash scripts/ash-data-backup.sh

echo "OK ash-data-backup-smoke"
