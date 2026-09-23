# Engine subproject. Self-contained on purpose: the engine is its own Go module,
# so its build and test commands live with it rather than in the root Makefile.

.PHONY: all build test race vet fmt check run describe tidy clean web web-dev web-clean protocol protocol-check

SPEC ?= examples/npc-live-mock.json
ADDR ?= :8080

all: check

# Everything a change has to pass before it is committed. Needs no JavaScript
# toolchain: web/dist is committed.
check: fmt vet protocol-check test race

# --- Protocol ---------------------------------------------------------------
# The session protocol is a WebSocket, which OpenAPI cannot describe, so the
# contract that matters is the event message. It is generated from the Go types
# that define it rather than mirrored by hand.

protocol:
	go run ./cmd/gen-protocol

protocol-check:
	@go run ./cmd/gen-protocol -check

# --- Player -----------------------------------------------------------------
# Sources live in web/src and build into web/dist, which is committed: the Go
# module embeds dist, so building the engine never requires a JS toolchain.
# Only changing the player does.

NODE := $(shell command -v node 2>/dev/null)
NPM  := $(shell command -v npm 2>/dev/null)

web: protocol web-deps
	@cd web && npm run build

# Unminified with sourcemaps, for actually debugging the thing.
web-dev: protocol web-deps
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

web-clean:
	rm -rf web/node_modules web/dist

build:
	go build ./...

test:
	go test ./... -count=1

# The graph is concurrent throughout, so the race detector is not optional here.
race:
	go test ./... -count=1 -race

vet:
	go vet ./...

fmt:
	gofmt -l -w .

# Run a session from a hand-assembled spec. Defaults to the mock platform:
# no key, no network, no cost.
#   make run SPEC=examples/npc-live.json
run:
	go run ./cmd/engineserver -addr $(ADDR) $(SPEC)

# Print a genre's wiring without starting a server.
describe:
	@go run ./cmd/engineserver -describe $(SPEC)

tidy:
	go mod tidy

clean:
	go clean -cache -testcache ./...
