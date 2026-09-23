# Engine subproject. Self-contained on purpose: the engine is its own Go module,
# so its build and test commands live with it rather than in the root Makefile.

.PHONY: help all build test race vet fmt check run dev describe tidy clean \
        port-free web web-dev web-deps web-toolchain web-clean protocol protocol-check

# A bare `make` explains itself rather than doing something unasked.
.DEFAULT_GOAL := help

SPEC ?= examples/bridge-keeper-mock.json
ADDR ?= :8080

help: ## List the targets worth knowing about
	@echo "Engine — a ChatGameLab game engine as a standalone Go module."
	@echo
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[1m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo
	@echo "Variables:"
	@echo "  SPEC=<file>      the game to run              (default: $(SPEC))"
	@echo "  ADDR=<addr>      listen address               (default: $(ADDR))"
	@echo
	@echo "Examples:"
	@echo "  make dev                                  play against the mock platform, no key needed"
	@echo "  make dev SPEC=examples/stopped-clock.json    play against OpenAI, key from --api-key or ~/.chatgamelab/config.yaml"
	@echo "  make describe SPEC=examples/bridge-quest.json  print a genre's wiring and exit"

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

dev: web port-free ## Build the player, then serve it with a session ready to play
	@go run ./cmd/engineserver -addr $(ADDR) $(SPEC)

run: port-free ## Serve a session without rebuilding the player
	@go run ./cmd/engineserver -addr $(ADDR) $(SPEC)

# The port is usually held by a server from the last run, so the useful thing is
# to say which one and offer to end it — rather than fail with an errno and
# leave somebody to find the process themselves.
PORT = $(lastword $(subst :, ,$(ADDR)))

port-free: ## Report whatever holds the dev port, and offer to end it
	@pids=`lsof -ti tcp:$(PORT) -sTCP:LISTEN 2>/dev/null`; \
	if [ -z "$$pids" ]; then exit 0; fi; \
	echo "Port $(PORT) is already in use by:"; \
	ps -o pid=,etime=,command= -p $$pids 2>/dev/null | cut -c1-110 | sed 's/^/  /'; \
	echo; \
	if [ ! -t 0 ]; then \
		echo "Nothing was asked, because this is not a terminal. Free it, or pass ADDR=:8081." >&2; \
		exit 1; \
	fi; \
	printf "Stop it and carry on? [y/N] "; \
	read -r reply; \
	case "$$reply" in \
		[yY]*) ;; \
		*) echo "Left running. Pass ADDR=:8081 to use another port." >&2; exit 1 ;; \
	esac; \
	kill $$pids 2>/dev/null; \
	for _ in 1 2 3 4 5 6 7 8 9 10; do \
		sleep 0.2; \
		still=`lsof -ti tcp:$(PORT) -sTCP:LISTEN 2>/dev/null`; \
		[ -z "$$still" ] && break; \
	done; \
	still=`lsof -ti tcp:$(PORT) -sTCP:LISTEN 2>/dev/null`; \
	if [ -n "$$still" ]; then \
		echo "Asked it to stop and it has not; killing outright." >&2; \
		kill -9 $$still 2>/dev/null; \
		sleep 0.3; \
	fi

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
