#!/usr/bin/env bash
# Invoke Docker Compose on Windows Git Bash / MSYS without breaking "-f" flags.
docker_compose() {
	# Prevent MSYS from rewriting compose file paths (causes "unknown shorthand flag: 'f'").
	case "${OSTYPE:-}" in
	msys* | cygwin*)
		export MSYS2_ARG_CONV_EXCL="${MSYS2_ARG_CONV_EXCL:-"-f;--file"}"
		;;
	esac

	local plugin=""
	for cand in \
		"/c/Program Files/Docker/cli-plugins/docker-compose.exe" \
		"${ProgramW6432:-/c/Program Files}/Docker/cli-plugins/docker-compose.exe" \
		"${ProgramFiles:-}/Docker/cli-plugins/docker-compose.exe"; do
		if [[ -n "$cand" && -x "$cand" ]]; then
			plugin="$cand"
			break
		fi
	done
	if [[ -n "$plugin" ]]; then
		MSYS_NO_PATHCONV=1 "$plugin" "$@"
		return $?
	fi

	local cli="docker"
	if command -v docker.exe >/dev/null 2>&1; then
		cli="docker.exe"
	fi
	if MSYS_NO_PATHCONV=1 "$cli" compose version >/dev/null 2>&1; then
		MSYS_NO_PATHCONV=1 "$cli" compose "$@"
		return $?
	fi
	if command -v docker-compose >/dev/null 2>&1; then
		MSYS_NO_PATHCONV=1 docker-compose "$@"
		return $?
	fi
	echo "docker compose (plugin) or docker-compose is required" >&2
	return 2
}
