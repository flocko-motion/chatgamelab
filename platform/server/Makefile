# Platform server. Its lint lives with it, as the engine's does in engine/Makefile.

.PHONY: help lint lint-go

.DEFAULT_GOAL := help

help: ## List the targets worth knowing about
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[1m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

lint: lint-go ## Run every linter

# brokkr's full gate reports as a warning for now; only comment length fails
# the build. Route handlers are exempt from comment length: their godoc blocks
# are the OpenAPI spec, so their length is swag's, not prose.
BROKKR := $(shell command -v brokkr 2>/dev/null)

lint-go: ## Run brokkr: every linter as a warning, comment length strict
ifndef BROKKR
	@echo "brokkr is required to lint the sources, and was not found." >&2
	@echo "  install it from https://github.com/flocko-motion/sindri"     >&2
	@exit 1
endif
	-@brokkr lint --tail 20
	brokkr lint comment-length --ignore='api/routes/**'
