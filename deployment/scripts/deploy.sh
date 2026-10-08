#!/usr/bin/env bash
# Deploy apps (Docker Compose) + refresh host nginx.
#
# Everything is read from the deployment repo root: deploy.conf (targets and
# images), docker-compose.yml, .env and nginx/<PROJECT>.conf.template.
#
# Modes (auto-detected unless --mode is set):
#   local  — git pull app repos on this host, build images, compose up
#   ci     — pull prebuilt images from a registry, compose up
#   images — use images already on this host (tags from .env), compose up
#
# Auto: GITLAB_CI=true or CI=true → ci; otherwise → local.
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
readonly SCRIPT_DIR="$ROOT_DIR/scripts"

TARGET_APP="${TARGET_APP:-all}" # a target from deploy.conf | nginx | all
SKIP_NGINX=false
MODE="${DEPLOY_MODE:-auto}" # auto | local | ci | images
APP_BRANCH=""               # optional override for git pull branch
IMAGE_OVERRIDES=()          # NAME=REF from --image / --<name>-image

target_choices() { printf '%s' "$(target_names | sed 's/ / | /g') | nginx | all"; }

usage() {
	local first_target first_image
	first_target="$(target_names)"
	first_target="${first_target%% *}"
	first_image="$(image_names)"
	first_image="${first_image%% *}"
	cat <<EOF
Usage: $0 [OPTIONS]

Modes:
  --mode auto|local|ci|images  auto (default): CI env → ci, else local
  local   git pull app repos, build images on this host, compose up
  ci      pull registry images (compose pull), compose up
  images  use images already on this host (<IMAGE>_IMAGE in .env), compose up

Options:
  --app NAME            $(target_choices)
$(for t in $(target_names); do
	printf '                          %-10s %s\n' "$t" "$(target_field "$t" 2)"
done)
                          nginx      only re-render/install host nginx
                          all        every service in docker-compose.yml
  --branch NAME         Git branch to pull in local mode (default: <REPO>_BRANCH
                        from .env, else main)
  --image NAME=REF      Set <NAME>_IMAGE in .env (ci mode; repeatable)
  --<name>-image REF    Same as --image <name>=REF
  --skip-nginx          Do not reinstall host nginx config
  -h, --help

Targets, images and services come from $CONFIG_FILE.

Examples (on the server):
  $0 --app $first_target
  $0 --app all
  $0 --app $first_target --branch develop
  $0 --app all --mode images

Examples (CI / registry):
  $0 --mode ci --app $first_target --$first_image-image registry/.../$first_image:abc1234
EOF
}

is_image()  { image_field "$1" 0 >/dev/null; }
is_target() { target_field "$1" 0 >/dev/null; }

parse_args() {
	while [[ $# -gt 0 ]]; do
		case "$1" in
		--app) TARGET_APP="$2"; shift 2 ;;
		--mode) MODE="$2"; shift 2 ;;
		--branch) APP_BRANCH="$2"; shift 2 ;;
		--image)
			[[ "${2:-}" == *=* ]] || die "--image needs NAME=REF"
			IMAGE_OVERRIDES+=("$2"); shift 2 ;;
		--skip-nginx) SKIP_NGINX=true; shift ;;
		-h | --help) usage; exit 0 ;;
		--*-image)
			local name="${1#--}"
			IMAGE_OVERRIDES+=("${name%-image}=$2"); shift 2 ;;
		*) die "unknown option: $1" ;;
		esac
	done

	case "$TARGET_APP" in
	nginx | all) ;;
	*) is_target "$TARGET_APP" || die "unknown --app: $TARGET_APP ($(target_choices))" ;;
	esac

	local o
	for o in "${IMAGE_OVERRIDES[@]}"; do
		is_image "${o%%=*}" || die "unknown image: ${o%%=*} (expected: $(image_names | sed 's/ / | /g'))"
	done
}

# Targets from deploy.conf covered by --app (none for nginx).
selected_targets() {
	case "$TARGET_APP" in
	nginx) ;;
	all) target_names ;;
	*) printf '%s' "$TARGET_APP" ;;
	esac
}

resolve_mode() {
	case "$MODE" in
	auto)
		if [[ "${GITLAB_CI:-}" == "true" ]] || [[ "${CI:-}" == "true" ]]; then
			MODE=ci
		else
			MODE=local
		fi
		;;
	local | ci | images) ;;
	*) die "invalid --mode: $MODE (use auto|local|ci|images)" ;;
	esac
	log "deploy mode: $MODE (app=$TARGET_APP)"
}

require_files() {
	[[ -f "$COMPOSE_FILE" ]] || die "compose not found: $COMPOSE_FILE"
	require_env_file
	local t d
	for t in $(selected_targets); do
		for d in $(target_field "$t" 3); do
			[[ -d "$ROOT_DIR/$d" ]] || die "missing $ROOT_DIR/$d/ (required by $t)"
		done
	done
}

update_image_tags_ci() {
	local o name ref prefix sha_var
	for o in "${IMAGE_OVERRIDES[@]}"; do
		name="${o%%=*}"
		ref="${o#*=}"
		prefix="$(var_prefix "$name")"
		sha_var="${prefix}_SHA"
		log "${prefix}_IMAGE=$ref"
		set_env_var "${prefix}_IMAGE" "$ref"
		set_env_var "$sha_var" "${!sha_var:-}"
	done
}

# --- local mode: pull sources + build images ---------------------------------

# Each repo is pulled at most once per deploy; REPO_SHA[repo] holds its HEAD.
declare -A REPO_SHA=()
declare -A BUILT=()

git_pull_repo() {
	local repo="$1"
	[[ -n "${REPO_SHA[$repo]:-}" ]] && return 0

	local repo_path branch_var branch
	repo_path="$(repo_dir "$repo")"
	branch_var="$(var_prefix "$repo")_BRANCH"
	branch="${APP_BRANCH:-${!branch_var:-main}}"

	[[ -d "$repo_path/.git" ]] || die "$repo repo not found or not a git clone: $repo_path"

	log "git pull $repo ($repo_path @ $branch)"
	git -C "$repo_path" fetch origin >&2
	git -C "$repo_path" checkout "$branch" >&2
	git -C "$repo_path" pull --ff-only origin "$branch" >&2

	REPO_SHA[$repo]="$(git -C "$repo_path" rev-parse --short HEAD)"
	log "$repo is at ${REPO_SHA[$repo]}"
}

# Build through build-image.sh so local deploys get exactly the same images
# (and build args from .env) as `make build-image`. <REPO>_DIR from .env is
# exported by load_env_file, so build-image.sh sees the same repo paths.
build_image_local() {
	local image="$1"
	[[ -n "${BUILT[$image]:-}" ]] && return 0

	local repo prefix
	repo="$(image_field "$image" 1)"
	git_pull_repo "$repo"
	"$SCRIPT_DIR/build-image.sh" "$image" local

	prefix="$(var_prefix "$image")"
	set_env_var "${prefix}_IMAGE" "$image:local"
	set_env_var "${prefix}_SHA" "${REPO_SHA[$repo]}"
	BUILT[$image]=1
}

deploy_local_build() {
	local t image
	for t in $(selected_targets); do
		for image in $(target_field "$t" 1); do
			build_image_local "$image"
		done
	done
}

# --- compose up --------------------------------------------------------------

deploy_compose() {
	[[ "$TARGET_APP" == "nginx" ]] && return 0

	local services=()
	# all → no service list → every service in the compose file.
	[[ "$TARGET_APP" != "all" ]] && read -ra services <<<"$(target_field "$TARGET_APP" 2)"

	command -v docker >/dev/null 2>&1 || die "docker not found"

	case "$MODE" in
	ci)
		log "pulling registry images ($TARGET_APP)"
		compose pull "${services[@]}"
		;;
	local)  log "using freshly built local images (no registry pull)" ;;
	images) log "using images already on this host (no build, no pull)" ;;
	esac

	# compose starts depends_on services as needed.
	log "compose up ($TARGET_APP)"
	compose up -d --remove-orphans "${services[@]}"
	compose ps
}

deploy_nginx() {
	if [[ "$SKIP_NGINX" == true ]]; then
		log "skipping host nginx (--skip-nginx)"
		return
	fi
	if [[ ! -f "$NGINX_TEMPLATE" ]]; then
		log "no $NGINX_TEMPLATE — skipping host nginx"
		return
	fi

	log "refreshing host nginx ($NGINX_DEST)"
	local sudo=()
	[[ "$EUID" -ne 0 ]] && sudo=(sudo)
	"${sudo[@]}" "$SCRIPT_DIR/render-nginx.sh" --install
}

main() {
	parse_args "$@"
	resolve_mode
	require_files
	load_env_file

	case "$MODE" in
	ci)     update_image_tags_ci ;;
	local)  deploy_local_build ;;
	images) ;;
	esac
	set_env_var DEPLOYED_AT "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	# reload after writes
	load_env_file

	deploy_compose
	deploy_nginx

	log "done (mode=$MODE)"
}

main "$@"
