#!/usr/bin/env bash
# Regenerate OpenAPI docs (Git Bash / Linux / macOS)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
# shellcheck source=_swag.sh
source "$ROOT/scripts/_swag.sh"
_ash_go_env_bootstrap "$ROOT"

_ash_swag_init "$ROOT"

echo "Swagger written to internal/api/docs/"
