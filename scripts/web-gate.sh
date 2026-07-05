#!/usr/bin/env bash
# Frontend lint + unit tests + production build gate (MVP §3).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/frontend"

# npmmirror 等镜像不支持 audit bulk API，会触发 npm ERR_INVALID_ARG_TYPE
export NPM_CONFIG_AUDIT=false

if [[ -x node_modules/.bin/eslint && -x node_modules/.bin/vitest ]]; then
  echo "== reuse node_modules =="
elif [[ -f package-lock.json ]]; then
  npm ci --no-audit --no-fund || npm install --no-audit --no-fund
else
  npm install --no-audit --no-fund
fi

npm run lint
npm run test:run
npm run build

echo "OK web-gate"
