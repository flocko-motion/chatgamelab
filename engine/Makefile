# Engine subproject. Self-contained on purpose: the engine is its own Go module,
# so its build and test commands live with it rather than in the root Makefile.

.PHONY: all build test race vet fmt check run describe tidy clean

SPEC ?= examples/npc-live-mock.json
ADDR ?= :8080

all: check

# Everything a change has to pass before it is committed.
check: fmt vet test race

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
