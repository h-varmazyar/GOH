#!/usr/bin/env bash
# Run a one-off tool (deploy.conf → TOOLS) once and exit: `compose run --rm`.
#
# By default the tool's image is built first as <image>:local from the app
# repo's current checkout (no git pull) and <IMAGE>_IMAGE in .env points at it.
# Everything after `--` is passed to the container as its command.
set -euo pipefail

# shellcheck source=lib.sh
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
readonly SCRIPT_DIR="$ROOT_DIR/scripts"

BUILD=true

usage() {
	local first
	first="$(tool_names)"
	first="${first%% *}"
	cat <<EOF
Usage: $0 <tool> [OPTIONS] [-- COMMAND...]

Arguments:
  tool      $(tool_names | sed 's/ / | /g')   (from $CONFIG_FILE)
  COMMAND   passed to the container, e.g. seed -days 7

Options:
  --no-build    Use the image already set in .env (<IMAGE>_IMAGE) instead of building
  -h, --help

Examples:
  $0 $first -- seed -days 7
  $0 $first --no-build -- status
EOF
}

POSITIONAL=()
COMMAND=()
while [[ $# -gt 0 ]]; do
	case "$1" in
	--no-build) BUILD=false; shift ;;
	-h | --help) usage; exit 0 ;;
	--) shift; COMMAND=("$@"); break ;;
	-*) die "unknown option: $1 (see --help)" ;;
	*) POSITIONAL+=("$1"); shift ;;
	esac
done

[[ ${#POSITIONAL[@]} -eq 1 ]] || { usage >&2; exit 1; }
TOOL="${POSITIONAL[0]}"

IMAGE="$(tool_field "$TOOL" 1)" ||
	die "unknown tool: $TOOL (expected: $(tool_names | sed 's/ / | /g'))"

[[ -f "$COMPOSE_FILE" ]] || die "compose not found: $COMPOSE_FILE"
require_env_file
for d in $(tool_field "$TOOL" 2); do
	[[ -d "$ROOT_DIR/$d" ]] || die "missing $ROOT_DIR/$d/ (required by $TOOL)"
done
command -v docker >/dev/null 2>&1 || die "docker not found"
load_env_file

if [[ "$BUILD" == true && -n "$IMAGE" ]]; then
	"$SCRIPT_DIR/build-image.sh" "$IMAGE" local

	prefix="$(var_prefix "$IMAGE")"
	repo_path="$(repo_dir "$(image_field "$IMAGE" 1)")"
	set_env_var "${prefix}_IMAGE" "$IMAGE:local"
	set_env_var "${prefix}_SHA" "$(git -C "$repo_path" rev-parse --short HEAD 2>/dev/null || true)"
	load_env_file
fi

log "running $TOOL ${COMMAND[*]}"
compose run --rm "$TOOL" "${COMMAND[@]}"
