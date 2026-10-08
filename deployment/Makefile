# Shortcuts for the deployment scripts in scripts/.
# Project-specific names (images, deploy targets, ...) live in deploy.conf.
# Extra flags for any target can be passed with ARGS="...", e.g.
#   make deploy APP=<target> ARGS="--skip-nginx"

SHELL := /usr/bin/env bash

APP     ?= all
VERSION ?=
ARGS    ?=
SUDO    ?= sudo

SCRIPTS := scripts

.PHONY: help build-image deploy render-nginx install-nginx check

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'
	@echo
	@source $(SCRIPTS)/lib.sh && \
		echo "Images  (build-image APP=): $$(image_names)" && \
		echo "Targets (deploy APP=):      $$(target_names) nginx all"
	@echo "Variables: APP=$(APP) VERSION=$(VERSION) ARGS=$(ARGS)"

build-image: ## Build an app image (APP=<image> VERSION=x.y.z)
	@test -n "$(VERSION)" || { echo "error: VERSION is required" >&2; exit 1; }
	@test "$(APP)" != "all" || { echo "error: APP is required (see make help)" >&2; exit 1; }
	$(SCRIPTS)/build-image.sh $(APP) $(VERSION) $(ARGS)

deploy: ## Deploy apps with compose + refresh host nginx (APP=<target>|nginx|all)
	$(SCRIPTS)/deploy.sh --app $(APP) $(ARGS)

render-nginx: ## Print the rendered host nginx config from its template
	$(SCRIPTS)/render-nginx.sh $(ARGS)

install-nginx: ## Render the nginx config, install it, nginx -t and reload (runs with sudo)
	$(SUDO) $(SCRIPTS)/render-nginx.sh --install $(ARGS)

check: ## Health checks for Docker apps + host nginx + deploy.conf checks
	$(SCRIPTS)/check.sh $(ARGS)
