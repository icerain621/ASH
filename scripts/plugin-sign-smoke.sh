#!/usr/bin/env bash
# Plugin signing smoke (Sprint CR): unit tests + CLI sign/verify round-trip.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
# shellcheck source=_go_env.sh
source "$ROOT/scripts/_go_env.sh"
_ash_go_env_bootstrap "$ROOT"

KEY="${ASH_PLUGIN_SIGNING_KEY:-plugin-sign-smoke-key}"
export ASH_PLUGIN_SIGNING_KEY="$KEY"

echo "== pluginabi / API signature unit tests =="
go test ./internal/pluginabi/ -count=1
go test ./internal/api/ -run 'TestRegisterPluginRequiresValidSignatureWhenKeySet' -count=1

echo "== CLI plugin-sign round-trip =="
SIG="$(go run ./cmd/cli plugin-sign \
  --name otel --version 1.0.0 --protocol grpc --abi ash.plugin.v1 \
  --endpoint 127.0.0.1:7443 --key "$KEY")"
if [[ ${#SIG} -ne 64 ]]; then
  echo "unexpected signature length: ${#SIG} (want 64 hex)" >&2
  exit 1
fi
CAP="$(go run ./cmd/cli plugin-sign \
  --name otel --version 1.0.0 --endpoint 127.0.0.1:7443 --key "$KEY" --capability)"
if [[ "$CAP" != "ash.sign.hmac=${SIG}" ]]; then
  echo "capability mismatch: $CAP (want ash.sign.hmac=${SIG})" >&2
  exit 1
fi

BAD="$(go run ./cmd/cli plugin-sign \
  --name otel --version 1.0.0 --endpoint 127.0.0.1:7443 --key wrong-key)"
if [[ "$BAD" == "$SIG" ]]; then
  echo "different keys produced identical signatures" >&2
  exit 1
fi

echo "signature=$SIG"
echo "OK plugin-sign-smoke"
