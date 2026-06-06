# shellcheck shell=bash
# Source from bash scripts when GOPATH/GOMODCACHE may be unset (Git Bash + make).
_ash_go_env_bootstrap() {
  local root="${1:?root required}"
  if [[ -z "${GOPATH:-}" ]]; then
    if command -v go >/dev/null 2>&1; then
      GOPATH="$(go env GOPATH 2>/dev/null || true)"
    fi
    if [[ -z "${GOPATH:-}" && -d "$root/../../src" ]]; then
      GOPATH="$(cd "$root/../.." && pwd)"
    fi
    [[ -n "${GOPATH:-}" ]] && export GOPATH
  fi
  if [[ -z "${GOMODCACHE:-}" ]]; then
    if command -v go >/dev/null 2>&1; then
      GOMODCACHE="$(go env GOMODCACHE 2>/dev/null || true)"
    fi
    if [[ -z "${GOMODCACHE:-}" && -n "${GOPATH:-}" ]]; then
      GOMODCACHE="${GOPATH}/pkg/mod"
    fi
    [[ -n "${GOMODCACHE:-}" ]] && export GOMODCACHE
  fi
  if [[ -n "${GOPATH:-}" ]]; then
    local pkg="${GOPATH}/pkg"
    if [[ -z "${GOCACHE:-}" || "${GOCACHE}" == *WINDOWS* ]]; then
      GOCACHE="${pkg}/go-build"
      mkdir -p "$GOCACHE" 2>/dev/null || true
      export GOCACHE
    fi
    if [[ -z "${GOTMPDIR:-}" || "${GOTMPDIR}" == *WINDOWS* ]]; then
      GOTMPDIR="${pkg}/go-tmp"
      mkdir -p "$GOTMPDIR" 2>/dev/null || true
      export GOTMPDIR TMPDIR="$GOTMPDIR" TEMP="$GOTMPDIR" TMP="$GOTMPDIR"
    fi
  fi
}
