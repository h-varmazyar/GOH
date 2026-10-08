#!/usr/bin/env bash
# Build a Docker image for one app from its sibling repo.
#
# Images (repo, Dockerfile, build args) are declared in deploy.conf → IMAGES.
# Build args that come from .env are read from this repo's .env, never from
# the app repo's own .env (which .dockerignore usually excludes anyway).
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

TAG_LATEST=false
NETWORK="${DOCKER_BUILD_NETWORK:-}"
EXTRA_BUILD_ARGS=()

usage() {
	local first
	first="$(image_names)"
	first="${first%% *}"
	cat <<EOF
Usage: $0 <app> <version> [OPTIONS]

Arguments:
  app       $(image_names | sed 's/ / | /g')   (from $CONFIG_FILE)
  version   Image tag, e.g. 1.2.0 or v1.2.0-rc1

Options:
  --latest              Also tag the image as <app>:latest
  --build-arg KEY=VAL   Extra docker build arg (repeatable)
  --network MODE        Network for RUN steps, e.g. host (env: DOCKER_BUILD_NETWORK)
  -h, --help

Environment:
  <REPO>_DIR     Path to an app repo (default: $WORKSPACE_DIR/<repo>), e.g. $(var_prefix "$(image_field "$first" 1)")_DIR
  ENV_FILE       .env feeding build args (default: $ROOT_DIR/.env)

Examples:
  $0 $first 1.4.0
  $0 $first 1.4.0 --latest
  $0 $first 1.4.0 --build-arg KEY=VAL   # CLI --build-arg overrides deploy.conf/.env
EOF
}

POSITIONAL=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	--latest) TAG_LATEST=true; shift ;;
	--build-arg)
		[[ $# -ge 2 ]] || die "--build-arg needs KEY=VAL"
		EXTRA_BUILD_ARGS+=("$2"); shift 2 ;;
	--network)
		[[ $# -ge 2 ]] || die "--network needs a mode"
		NETWORK="$2"; shift 2 ;;
	-h | --help) usage; exit 0 ;;
	-*) die "unknown option: $1 (see --help)" ;;
	*) POSITIONAL+=("$1"); shift ;;
	esac
done

[[ ${#POSITIONAL[@]} -eq 2 ]] || { usage >&2; exit 1; }
APP="${POSITIONAL[0]}"
VERSION="${POSITIONAL[1]}"

# Docker tag rules: [A-Za-z0-9_][A-Za-z0-9_.-]{0,127}
[[ "$VERSION" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]] || die "invalid version tag: $VERSION"

REPO="$(image_field "$APP" 1)" ||
	die "unknown app: $APP (expected: $(image_names | sed 's/ / | /g'))"
DOCKERFILE="$(image_field "$APP" 2)"
CONTEXT="$(repo_dir "$REPO")"
[[ -n "$DOCKERFILE" ]] || die "no dockerfile for $APP in $CONFIG_FILE"

BUILD_ARGS=()
read -ra spec <<<"$(image_field "$APP" 3)"
for arg in "${spec[@]}"; do
	case "$arg" in
	*=*) BUILD_ARGS+=("$arg") ;;
	*\?)
		# Absent → keep the Dockerfile default; present but empty → pass empty.
		require_env_file
		if value="$(env_get "${arg%\?}")"; then
			BUILD_ARGS+=("${arg%\?}=$value")
		fi
		;;
	*)
		require_env_file
		value="$(env_get "$arg")" && [[ -n "$value" ]] ||
			die "$arg is not set in $ENV_FILE (build arg for $APP)"
		BUILD_ARGS+=("$arg=$value")
		;;
	esac
done

command -v docker >/dev/null 2>&1 || die "docker not found in PATH"
[[ -d "$CONTEXT" ]] || die "repo not found: $CONTEXT"
[[ -f "$CONTEXT/$DOCKERFILE" ]] || die "Dockerfile missing: $CONTEXT/$DOCKERFILE"

IMAGE="$APP:$VERSION"
args=(-t "$IMAGE" -f "$CONTEXT/$DOCKERFILE")
[[ "$TAG_LATEST" == true ]] && args+=(-t "$APP:latest")
[[ -n "$NETWORK" ]] && args+=(--network "$NETWORK")

for a in "${BUILD_ARGS[@]}" "${EXTRA_BUILD_ARGS[@]}"; do
	args+=(--build-arg "$a")
done

if git -C "$CONTEXT" rev-parse --git-dir >/dev/null 2>&1; then
	args+=(--label "org.opencontainers.image.revision=$(git -C "$CONTEXT" rev-parse HEAD)")
fi
args+=(--label "org.opencontainers.image.version=$VERSION")
args+=("$CONTEXT")

log "building $IMAGE from $CONTEXT ($DOCKERFILE)"
[[ ${#BUILD_ARGS[@]} -gt 0 || ${#EXTRA_BUILD_ARGS[@]} -gt 0 ]] &&
	log "build args: ${BUILD_ARGS[*]} ${EXTRA_BUILD_ARGS[*]}"

docker build "${args[@]}"

log "done: $IMAGE$([[ "$TAG_LATEST" == true ]] && echo " (+ $APP:latest)")"
