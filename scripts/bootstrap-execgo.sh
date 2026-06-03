#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKDIR="${EXECGO_WORKDIR:-"$ROOT_DIR/.ash/execgo"}"
EXECGO_REPO="${EXECGO_REPO:-https://github.com/iammm0/execgo.git}"
RUNTIME_REPO="${EXECGO_RUNTIME_REPO:-https://github.com/iammm0/execgo-runtime.git}"
EXECGO_REF="${EXECGO_REF:-}"
RUNTIME_REF="${EXECGO_RUNTIME_REF:-}"
SKIP_RUNTIME="${ASH_EXECGO_SKIP_RUNTIME:-0}"

EXECGO_DIR="$WORKDIR/execgo"
RUNTIME_DIR="$WORKDIR/execgo-runtime"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

clone_or_update() {
  local repo="$1"
  local dir="$2"
  local ref="$3"

  if [[ -d "$dir/.git" ]]; then
    echo "Updating $dir"
    git -C "$dir" fetch --all --tags --prune
  else
    echo "Cloning $repo -> $dir"
    git clone "$repo" "$dir"
  fi

  if [[ -n "$ref" ]]; then
    git -C "$dir" checkout "$ref"
  fi
}

echo "Bootstrapping ExecGo into $WORKDIR"
mkdir -p "$WORKDIR"

need_cmd git
need_cmd go

clone_or_update "$EXECGO_REPO" "$EXECGO_DIR" "$EXECGO_REF"

echo "Building execgo and execgocli"
(
  cd "$EXECGO_DIR"
  mkdir -p bin
  go build -o ./bin/execgocli ./cmd/execgocli
  go build -o ./bin/execgo ./cmd/execgo
)

if [[ "$SKIP_RUNTIME" != "1" ]]; then
  need_cmd cargo
  clone_or_update "$RUNTIME_REPO" "$RUNTIME_DIR" "$RUNTIME_REF"
  echo "Building execgo-runtime"
  (
    cd "$RUNTIME_DIR"
    cargo build --release
  )
else
  echo "Skipping execgo-runtime build because ASH_EXECGO_SKIP_RUNTIME=1"
fi

cat <<EOF

ExecGo bootstrap complete.

Add these exports in the shell where you run ASH:

export EXECGO_EXECGOCLI="$EXECGO_DIR/bin/execgocli"
export PATH="$EXECGO_DIR/bin:\$PATH"
export EXECGO_URL="\${EXECGO_URL:-http://127.0.0.1:8080}"
export EXECGO_RUNTIME_URL="\${EXECGO_RUNTIME_URL:-http://127.0.0.1:18080}"

Start ExecGo control plane:

cd "$EXECGO_DIR"
EXECGO_RUNTIME_URL="\$EXECGO_RUNTIME_URL" ./bin/execgo

Start execgo-runtime in another shell when runtime.command is required:

cd "$RUNTIME_DIR"
cargo run -- serve --listen-addr 127.0.0.1:18080 --data-dir ./data

Then verify from this ASH repo:

make execgo-health
EOF
