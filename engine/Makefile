# Engine subproject. Self-contained on purpose: the engine is its own Go module,
# so its build and test commands live with it rather than in the root Makefile.

.PHONY: help all build test race vet fmt check run dev describe tidy clean \
        web web-dev web-deps web-toolchain web-clean protocol protocol-check

# A bare `make` explains itself rather than doing something unasked.
.DEFAULT_GOAL := help

SPEC ?= examples/npc-live-mock.json
ADDR ?= :8080

help: ## List the targets worth knowing about
	@echo "Engine — a ChatGameLab game engine as a standalone Go module."
	@echo
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[1m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo "Variables:"
	@echo "  SPEC=<file>      session spec to run          (default: $(SPEC))"
	@echo "  ADDR=<addr>      listen address               (default: $(ADDR))"
	@echo
	@echo "Examples:"
	@echo "  make dev                                  play against the mock platform, no key needed"
	@echo "  make dev SPEC=examples/npc-live.json      play against OpenAI, key from --api-key or ~/.chatgamelab/config.yaml"
	@echo "  make describe SPEC=examples/adventure.json   print a genre's wiring and exit"

all: check

check: fmt vet protocol-check test race ## Everything a change must pass before it is committed

build: ## Compile every package
	go build ./...

test: ## Run the tests
	go test ./... -count=1

race: ## Run the tests under the race detector, which a concurrent graph is not optional about
	go test ./... -count=1 -race

vet: ## Report suspicious constructs
	go vet ./...

fmt: ## Format the Go sources in place
	gofmt -l -w .

# --- Running ----------------------------------------------------------------

dev: web ## Build the player, then serve it with a session ready to play
	@echo
	@echo "  player  http://localhost$(ADDR)/player/"
	@echo
	@go run ./cmd/engineserver -addr $(ADDR) $(SPEC)

run: ## Serve a session without rebuilding the player
	go run ./cmd/engineserver -addr $(ADDR) $(SPEC)

describe: ## Print a genre's wiring and exit
	@go run ./cmd/engineserver -describe $(SPEC)

# --- Protocol ---------------------------------------------------------------
# The session protocol is a WebSocket, which OpenAPI cannot describe, so the
# contract that matters is the event message. It is generated from the Go types
# that define it rather than mirrored by hand.

protocol: ## Regenerate the TypeScript view of the wire protocol
	go run ./cmd/gen-protocol

protocol-check: ## Fail if the generated protocol is stale
	@go run ./cmd/gen-protocol -check

# --- Player -----------------------------------------------------------------
# Sources live in web/src and build into web/dist, which is committed: the Go
# module embeds dist, so building the engine never requires a JS toolchain.
# Only changing the player does.

NODE := $(shell command -v node 2>/dev/null)
NPM  := $(shell command -v npm 2>/dev/null)

web: protocol web-deps ## Typecheck and bundle the player into web/dist
	@cd web && npm run build

web-dev: protocol web-deps ## Bundle the player unminified, for reading a stack trace
	@cd web && DEV=1 npm run build

web-deps: web-toolchain
	@cd web && [ -d node_modules ] || npm install

web-toolchain:
ifndef NODE
	@echo "node is required to build the player, and was not found."          >&2
	@echo "  install Node 20 or newer: https://nodejs.org/"                   >&2
	@echo "  the Go build does not need it — web/dist is committed."          >&2
	@exit 1
endif
ifndef NPM
	@echo "npm is required to build the player, and was not found."           >&2
	@echo "  it normally ships with Node: https://nodejs.org/"                >&2
	@exit 1
endif
	@:

# --- Housekeeping -----------------------------------------------------------

tidy: ## Tidy the module's dependencies
	go mod tidy

web-clean: ## Remove the player's build output and installed toolchain
	rm -rf web/node_modules web/dist

clean: ## Clear the Go build and test caches
	go clean -cache -testcache ./...
