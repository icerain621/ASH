#!/usr/bin/env bash
# Static smoke gate (no Worker): regression-short + H-04..H-09 api/doctor coverage.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
bash scripts/regression-short.sh
