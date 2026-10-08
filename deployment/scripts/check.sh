#!/usr/bin/env bash
# Health checks: every compose service running, host nginx config installed,
# valid and up to date, then the project's own checks (project_checks in deploy.conf).
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

VERBOSE=false

PASS=0
FAIL=0

usage() {
	cat <<EOF
Usage: $0 [--verbose]

Reads $ENV_FILE, $COMPOSE_FILE and project_checks from $CONFIG_FILE.
EOF
}

log()  { echo "==> $*"; }
pass() { PASS=$((PASS + 1)); echo "[PASS] $*"; }
fail() { FAIL=$((FAIL + 1)); echo "[FAIL] $*" >&2; }
info() { [[ "$VERBOSE" == true ]] && echo "    $*" || true; }

while [[ $# -gt 0 ]]; do
	case "$1" in
	--verbose) VERBOSE=true; shift ;;
	-h|--help) usage; exit 0 ;;
	*) echo "unknown option: $1" >&2; exit 1 ;;
	esac
done

[[ -f "$ENV_FILE" ]] && load_env_file

# --- helpers for project_checks ---------------------------------------------

# check_http LABEL URL EXPECT_REGEX [CURL_ARGS...]
check_http() {
	local label="$1" url="$2" expect="$3"
	shift 3

	command -v curl >/dev/null 2>&1 || { fail "curl not installed"; return; }

	local code
	code="$(curl -s -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 10 "$@" "$url" 2>/dev/null || true)"
	code="${code:-000}"
	info "$label -> HTTP $code ($url)"
	if [[ "$code" =~ $expect ]]; then
		pass "$label HTTP $code"
	else
		fail "$label HTTP $code (expected $expect, $url)"
	fi
}

# check_tcp LABEL HOST PORT
check_tcp() {
	local label="$1" host="$2" port="$3"
	if timeout 5 bash -c 'exec 3<>"/dev/tcp/$0/$1"' "$host" "$port" 2>/dev/null; then
		pass "$label ($host:$port open)"
	else
		fail "$label ($host:$port not reachable)"
	fi
}

# --- generic checks ----------------------------------------------------------

check_containers() {
	log "docker compose services"
	local wanted running svc
	wanted="$(compose config --services 2>&1)" || { fail "compose config failed"; echo "$wanted" >&2; return; }
	running="$(compose ps --status running --services 2>/dev/null || true)"
	info "running: ${running//$'\n'/ }"
	for svc in $wanted; do
		if grep -qx "$svc" <<<"$running"; then
			pass "container running: $svc"
		else
			fail "container not running: $svc"
		fi
	done
}

check_nginx_host() {
	if [[ ! -f "$NGINX_TEMPLATE" ]]; then
		log "host nginx: no $NGINX_TEMPLATE — skipped"
		return
	fi
	log "host nginx"
	if ! command -v nginx >/dev/null 2>&1; then
		fail "nginx binary not found on host"
		return
	fi
	if systemctl is-active --quiet nginx 2>/dev/null; then
		pass "systemd: nginx is active"
	else
		fail "systemd: nginx is not active"
	fi

	local out
	if out="$(nginx -t 2>&1)" || out="$(sudo nginx -t 2>&1)"; then
		pass "nginx config valid"
		info "$out"
	else
		fail "nginx config invalid"
		echo "$out" >&2
	fi

	if [[ ! -f "$NGINX_DEST" ]]; then
		fail "missing $NGINX_DEST (run make install-nginx)"
	elif diff -q <("$ROOT_DIR/scripts/render-nginx.sh" 2>/dev/null) "$NGINX_DEST" >/dev/null 2>&1; then
		pass "$NGINX_DEST matches template + .env"
	else
		fail "$NGINX_DEST differs from template + .env (run make install-nginx)"
	fi
}

main() {
	[[ -f "$COMPOSE_FILE" ]] || { echo "missing $COMPOSE_FILE" >&2; exit 1; }

	check_containers
	check_nginx_host

	if declare -F project_checks >/dev/null; then
		log "project checks ($CONFIG_FILE)"
		project_checks
	fi

	echo
	echo "PASS=$PASS FAIL=$FAIL"
	[[ "$FAIL" -eq 0 ]]
}

main "$@"
