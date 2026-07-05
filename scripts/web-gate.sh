#!/usr/bin/env bash
# Frontend lint + unit tests + production build gate (MVP §3).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/frontend"

if [[ -f package-lock.json ]]; then
  npm ci
else
  npm install
fi

npm run lint
npm run test:run
npm run build

echo "OK web-gate"
