#!/usr/bin/env bash
# Shared helpers for the deployment scripts. Source it, don't run it:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
#
# Flat layout — everything lives at the deployment repo root:
#   deploy.conf  .env  docker-compose.yml  nginx/  scripts/
# App repos are siblings of the deployment repo. Everything project-specific
# (project name, images, deploy targets, checks) comes from deploy.conf.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE_DIR="$(cd "$ROOT_DIR/.." && pwd)"
CONFIG_FILE="${CONFIG_FILE:-$ROOT_DIR/deploy.conf}"
ENV_FILE="${ENV_FILE:-$ROOT_DIR/.env}"
COMPOSE_FILE="$ROOT_DIR/docker-compose.yml"
NGINX_DIR="$ROOT_DIR/nginx"

log()  { echo "==> $*" >&2; }
warn() { echo "warning: $*" >&2; }
die()  { echo "error: $*" >&2; exit 1; }

[[ -f "$CONFIG_FILE" ]] || die "missing $CONFIG_FILE (copy one from another project and edit it)"
# shellcheck source=../deploy.conf
source "$CONFIG_FILE"
[[ -n "${PROJECT:-}" ]] || die "PROJECT is not set in $CONFIG_FILE"
declare -p IMAGES >/dev/null 2>&1 || IMAGES=()
declare -p TARGETS >/dev/null 2>&1 || TARGETS=()
declare -p TOOLS >/dev/null 2>&1 || TOOLS=()

NGINX_TEMPLATE="$NGINX_DIR/$PROJECT.conf.template"
NGINX_DEST="${NGINX_DEST:-/etc/nginx/conf.d/$PROJECT.conf}"

require_env_file() {
	[[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE — cp $ROOT_DIR/.env.example $ROOT_DIR/.env"
}

# Strip one pair of surrounding quotes, like docker compose does.
_unquote() {
	local v="$1"
	if [[ "$v" =~ ^\"(.*)\"$ ]] || [[ "$v" =~ ^\'(.*)\'$ ]]; then
		v="${BASH_REMATCH[1]}"
	fi
	printf '%s' "$v"
}

# Export every KEY=VAL from .env without sourcing it: values may contain
# unquoted spaces (e.g. DAY_OTP_MESSAGE_SUBJECT) that bash would run as commands.
load_env_file() {
	require_env_file
	local line
	while IFS= read -r line || [[ -n "$line" ]]; do
		line="${line%$'\r'}"
		[[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]] || continue
		export "${BASH_REMATCH[1]}=$(_unquote "${BASH_REMATCH[2]}")"
	done <"$ENV_FILE"
}

# Print one key from .env. Returns 1 when the key is absent, so
# "set but empty" can be told apart from "missing".
env_get() {
	local line
	line="$(grep -E "^$1=" "$ENV_FILE" | tail -n1)" || return 1
	_unquote "${line#*=}"
}

# Write KEY=VALUE into .env, replacing an existing line or appending.
# Avoids sed — values may contain /, &, | ...
set_env_var() {
	local key="$1" value="$2" file="${3:-$ENV_FILE}"
	local tmp line found=0
	tmp="$(mktemp)"
	if [[ -f "$file" ]]; then
		while IFS= read -r line || [[ -n "$line" ]]; do
			if [[ "$line" == "${key}="* ]]; then
				printf '%s=%s\n' "$key" "$value"
				found=1
			else
				printf '%s\n' "$line"
			fi
		done <"$file" >"$tmp"
	fi
	[[ "$found" -eq 1 ]] || printf '%s=%s\n' "$key" "$value" >>"$tmp"
	# cat (not mv) keeps the file's owner/permissions intact.
	cat "$tmp" >"$file"
	rm -f "$tmp"
}

compose() {
	docker compose --project-directory "$ROOT_DIR" --env-file "$ENV_FILE" -f "$COMPOSE_FILE" "$@"
}

is_true() {
	case "${1,,}" in
	true | 1 | yes | y) return 0 ;;
	*) return 1 ;;
	esac
}

# --- deploy.conf rows -------------------------------------------------------
# IMAGES / TARGETS / TOOLS rows are "a | b | c"; fields are trimmed, the first is the name.

_trim() {
	local s="$1"
	s="${s#"${s%%[![:space:]]*}"}"
	printf '%s' "${s%"${s##*[![:space:]]}"}"
}

# Print field N (0-based) of the row named $2 in the array named $1.
# Returns 1 when no row has that name.
_config_field() {
	local -n _rows="$1"
	local row fields
	for row in "${_rows[@]}"; do
		IFS='|' read -ra fields <<<"$row"
		if [[ "$(_trim "${fields[0]}")" == "$2" ]]; then
			_trim "${fields[$3]:-}"
			return 0
		fi
	done
	return 1
}

_config_names() {
	local -n _rows="$1"
	local row names=()
	for row in "${_rows[@]}"; do
		names+=("$(_trim "${row%%|*}")")
	done
	printf '%s' "${names[*]}"
}

image_names()  { _config_names IMAGES; }
target_names() { _config_names TARGETS; }
tool_names()   { _config_names TOOLS; }

# image_field NAME N: 1=repo 2=dockerfile 3=build args
image_field()  { _config_field IMAGES "$1" "$2"; }
# target_field NAME N: 1=images 2=services 3=dirs
target_field() { _config_field TARGETS "$1" "$2"; }
# tool_field NAME N: 1=image 2=dirs
tool_field()   { _config_field TOOLS "$1" "$2"; }

# Unique repos referenced by IMAGES, in order.
image_repos() {
	local name repo seen=" " out=()
	for name in $(image_names); do
		repo="$(image_field "$name" 1)"
		[[ "$seen" == *" $repo "* ]] && continue
		seen+="$repo "
		out+=("$repo")
	done
	printf '%s' "${out[*]}"
}

# Env var prefix for a name: gozargah-worker → GOZARGAH_WORKER.
var_prefix() {
	local v="${1^^}"
	v="${v//-/_}"
	printf '%s' "${v//./_}"
}

# Source dir of an app repo: <REPO>_DIR if set, else the sibling directory.
repo_dir() {
	local var
	var="$(var_prefix "$1")_DIR"
	printf '%s' "${!var:-$WORKSPACE_DIR/$1}"
}
