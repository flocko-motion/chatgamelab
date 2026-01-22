# Integration Tests

Go-based integration tests for the ChatGameLab backend API.

## Concept

- **Suite-based**: Each test suite gets its own clean database state
- **Simple setup**: Just Postgres in Docker, backend runs with `go run`
- **Fast**: Postgres stays running between suites, only tables are dropped
- **High-level API**: User-centric test helpers with fluent interface
- **IDE-friendly**: Run any test directly from your IDE

## Quick Start

```bash
# Setup (first time only)
cd testing && go mod tidy

# Run all tests (Postgres auto-starts on first run)
go test -v

# Run specific suite
go test -v -run TestBasicSuite
```

## Writing Tests

```go
import (
    "testing"
    "cgl/obj"
    "cgl/testing/testutil"
    "github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
    testutil.BaseSuite  // Automatic Docker lifecycle
}

func TestMySuite(t *testing.T) {
    s := &MyTestSuite{}
    s.SuiteName = "My Tests"
    suite.Run(t, s)
}

func (s *MyTestSuite) TestExample() {
    // Create users with fluent API
    alice := testutil.CreateUser(s.T(), "alice", "alice@example.com").Role("admin")
    bob := testutil.CreateUser(s.T(), "bob", "bob@example.com")
    
    // Make API calls
    var game obj.Game
    alice.MustPost("games/new", map[string]interface{}{
        "name": "Test Game",
    }, &game)
    
    // Suite assertions
    s.NotEmpty(game.ID)
    s.Equal("Test Game", game.Name)
}
```

## Structure

```
testing/
├── testutil/              # Infrastructure (BaseSuite, UserClient, etc.)
├── integration_test.go    # BasicTestSuite
├── multiuser_test.go      # MultiUserTestSuite
└── go.mod
```

**Architecture:**
- Postgres runs in Docker (port 7104) - stays running between suites
- Backend runs with `go run` (port 7102) - restarted per suite
- Each suite: **SetupSuite** (clean DB + start backend) → **tests** → **TearDownSuite** (stop backend)
