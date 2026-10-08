#!/usr/bin/env bash
# Render nginx/<PROJECT>.conf.template with values from .env, so host nginx
# proxies to exactly the address:port docker-compose published each app on.
#
#   scripts/render-nginx.sh                 # print rendered config
#   sudo scripts/render-nginx.sh --install  # write, nginx -t, reload
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

DEST="$NGINX_DEST"
TEMPLATE="$NGINX_TEMPLATE"
INSTALL=false

while [[ $# -gt 0 ]]; do
	case "$1" in
	--dest) DEST="$2"; shift 2 ;;
	--install) INSTALL=true; shift ;;
	-h | --help) sed -n '2,6p' "$0"; exit 0 ;;
	*) die "unknown option: $1" ;;
	esac
done

require_env_file
[[ -f "$TEMPLATE" ]] || die "missing $TEMPLATE"
command -v envsubst >/dev/null 2>&1 || die "envsubst not found (install gettext)"

# Only brace-style ${VAR} references are substituted, and each must be set in
# .env; nginx's own variables ($host, $remote_addr, ...) are left untouched.
mapfile -t VARS < <(grep -oE '\$\{[A-Za-z_][A-Za-z0-9_]*\}' "$TEMPLATE" | tr -d '${}' | sort -u)
shell_format=""
for var in "${VARS[@]}"; do
	value="$(env_get "$var")" && [[ -n "$value" ]] || die "$var is not set in $ENV_FILE (used by $TEMPLATE)"
	export "$var=$value"
	shell_format+="\${$var} "
	log "$var=$value"
done

log "env file: $ENV_FILE"
rendered="$(envsubst "$shell_format" <"$TEMPLATE")"

if [[ "$INSTALL" != true ]]; then
	printf '%s\n' "$rendered"
	exit 0
fi

backup=""
if [[ -f "$DEST" ]]; then
	backup="$DEST.bak"
	cp "$DEST" "$backup"
fi
printf '%s\n' "$rendered" >"$DEST"

if ! nginx -t; then
	if [[ -n "$backup" ]]; then mv "$backup" "$DEST"; else rm -f "$DEST"; fi
	die "nginx -t failed — previous config restored"
fi
rm -f "$backup"
systemctl reload nginx
log "installed $DEST and reloaded nginx"
