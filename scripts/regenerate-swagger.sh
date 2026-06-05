#!/usr/bin/env bash
# Regenerate OpenAPI docs (Git Bash / Linux / macOS)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go run github.com/swaggo/swag/cmd/swag@latest init \
  -g cmd/worker/main.go \
  -o internal/api/docs \
  --parseDependency \
  --parseInternal

echo "Swagger written to internal/api/docs/"
