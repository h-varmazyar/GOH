#!/usr/bin/env bash
# One-time host bootstrap: Docker + host nginx (+ gettext for envsubst).
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
log() { echo "==> $*"; }

require_root() {
	[[ "${EUID:-$(id -u)}" -eq 0 ]] || die "run as root: sudo $0"
}

detect_os_family() {
	# shellcheck source=/dev/null
	source /etc/os-release
	case "${ID:-}" in
	debian | ubuntu | linuxmint | pop) echo debian ;;
	rocky | rhel | centos | almalinux | fedora | ol) echo rhel ;;
	*)
		if [[ "${ID_LIKE:-}" == *"debian"* ]]; then echo debian
		elif [[ "${ID_LIKE:-}" == *"rhel"* ]] || [[ "${ID_LIKE:-}" == *"fedora"* ]]; then echo rhel
		else echo unknown
		fi
		;;
	esac
}

install_docker() {
	if command -v docker >/dev/null 2>&1; then
		log "docker already installed: $(docker --version)"
		return
	fi
	log "installing Docker"
	curl -fsSL https://get.docker.com | sh
	systemctl enable --now docker
}

ensure_compose() {
	docker compose version >/dev/null 2>&1 || die "docker compose plugin missing"
	log "docker compose ok: $(docker compose version)"
}

install_nginx() {
	local family
	family="$(detect_os_family)"

	if command -v nginx >/dev/null 2>&1; then
		log "nginx already installed: $(nginx -v 2>&1)"
	else
		log "installing nginx ($family)"
		case "$family" in
		debian)
			apt-get update -qq
			DEBIAN_FRONTEND=noninteractive apt-get install -y nginx gettext-base
			;;
		rhel)
			dnf install -y nginx gettext
			;;
		*) die "unsupported OS for auto nginx install" ;;
		esac
	fi

	# envsubst renders the nginx template (scripts/render-nginx.sh).
	if ! command -v envsubst >/dev/null 2>&1; then
		log "installing gettext (envsubst)"
		case "$family" in
		debian) DEBIAN_FRONTEND=noninteractive apt-get install -y gettext-base ;;
		rhel) dnf install -y gettext ;;
		*) die "install gettext (envsubst) manually" ;;
		esac
	fi

	systemctl enable nginx
	systemctl start nginx || true
}

print_next_steps() {
	local deploy_repo base repo t d dirs=""
	deploy_repo="$(basename "$ROOT_DIR")"
	base="/opt/$PROJECT"
	for t in $(target_names); do
		for d in $(target_field "$t" 3); do
			[[ " $dirs " == *" $d "* ]] || dirs+="${dirs:+ }$d"
		done
	done

	echo
	echo "Bootstrap complete."
	echo
	echo "Next steps on this host:"
	echo "  1. Clone repos side by side (once):"
	for repo in "$deploy_repo" $(image_repos); do
		echo "       git clone <$repo-url> $base/$repo"
	done
	echo "  2. cd $base/$deploy_repo"
	echo "  3. cp .env.example .env and fill it in (passwords, ports, build args ...)"
	if [[ -n "$dirs" ]]; then
		echo "  4. Put secrets / config files in: $dirs"
	else
		echo "  4. (no required dirs in deploy.conf)"
	fi
	echo "  5. Open the firewall for the public port(s) in nginx/$PROJECT.conf.template only"
	echo "  6. Passwordless sudo for nginx reload (recommended), e.g.:"
	echo "       echo 'deploy ALL=(root) NOPASSWD: /usr/sbin/nginx, /bin/systemctl reload nginx, /bin/systemctl start nginx, /bin/cp, /bin/mkdir, /usr/bin/tee, /bin/systemctl enable nginx' > /etc/sudoers.d/$PROJECT"
	echo "  7. First deploy (local: git pull apps + docker build + compose):"
	echo "       make deploy APP=all"
	echo "  8. make check"
	echo
	echo "Daily updates: make deploy APP=$(target_names | tr ' ' '|')|all"
	echo
}

main() {
	require_root
	install_docker
	ensure_compose
	install_nginx
	print_next_steps
}

main "$@"
