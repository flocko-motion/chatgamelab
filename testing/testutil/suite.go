package testutil

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/stretchr/testify/suite"
)

// testLock ensures only one test suite runs at a time
var testLock sync.Mutex

// BaseSuite provides test environment management for test suites
// Embed this in your specific test suites to get automatic setup/teardown
type BaseSuite struct {
	suite.Suite
	SuiteName     string
	backendCmd    *exec.Cmd
	backendCancel context.CancelFunc
}

// SetupSuite runs once before all tests in the suite
// Starts Postgres in Docker and backend with 'go run'
func (s *BaseSuite) SetupSuite() {
	// Lock to prevent parallel suite execution
	testLock.Lock()

	if s.SuiteName == "" {
		s.SuiteName = "Test Suite"
	}

	fmt.Printf("\n🚀 [%s] Starting test environment...\n", s.SuiteName)

	// Check if Postgres is already running
	checkCmd := exec.Command("docker", "ps", "-q", "-f", "name=chatgamelab-db-test")
	output, _ := checkCmd.Output()

	if len(output) > 0 {
		// Postgres is running, just drop all tables for clean state
		fmt.Printf("♻️  [%s] Postgres already running, cleaning database...\n", s.SuiteName)
		s.dropAllTables()
	} else {
		// Remove any stopped container with same name
		exec.Command("docker", "rm", "-f", "chatgamelab-db-test").Run()

		// Start Postgres with docker run
		fmt.Printf("🐘 [%s] Starting Postgres...\n", s.SuiteName)
		cmd := exec.Command("docker", "run", "-d",
			"--name", "chatgamelab-db-test",
			"-e", "POSTGRES_DB=chatgamelab",
			"-e", "POSTGRES_USER=chatgamelab",
			"-e", "POSTGRES_PASSWORD=testpassword",
			"-p", "7104:5432",
			"postgres:18")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			s.TearDownSuite()
			s.T().Fatalf("Failed to start Postgres: %v", err)
		}
		// Wait for Postgres to be ready
		time.Sleep(3 * time.Second)
	}

	// Start backend with 'go run'
	fmt.Printf("🚀 [%s] Starting backend with 'go run'...\n", s.SuiteName)
	ctx, cancel := context.WithCancel(context.Background())
	s.backendCancel = cancel

	s.backendCmd = exec.CommandContext(ctx, "go", "run", ".", "server")
	s.backendCmd.Dir = "../server"
	s.backendCmd.Env = append(os.Environ(),
		"DB_HOST=localhost",
		"DB_USER=chatgamelab",
		"DB_PASSWORD=testpassword",
		"DB_DATABASE=chatgamelab",
		"PORT_BACKEND=7102",
		"PORT_POSTGRES=7104",
		"DEV_MODE=true",
		"DEV_JWT_SECRET=testsecret123",
		"AUTH0_DOMAIN=test.auth0.domain",
		"AUTH0_AUDIENCE=test.auth0.audience",
		"PUBLIC_URL=http://localhost:7102",
	)
	s.backendCmd.Stdout = os.Stdout
	s.backendCmd.Stderr = os.Stderr

	if err := s.backendCmd.Start(); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Failed to start backend: %v", err)
	}

	// Wait for backend to be ready
	fmt.Printf("⏳ [%s] Waiting for backend to be ready...\n", s.SuiteName)
	if err := s.waitForBackend(30); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Backend not ready: %v", err)
	}
	fmt.Printf("✅ [%s] Backend is ready!\n\n", s.SuiteName)
}

// TearDownSuite runs once after all tests in the suite
func (s *BaseSuite) TearDownSuite() {
	defer testLock.Unlock()

	fmt.Printf("\n🧹 [%s] Cleaning up test environment...\n", s.SuiteName)

	// Stop backend only (keep Postgres running for next suite)
	if s.backendCancel != nil {
		s.backendCancel()
	}
	if s.backendCmd != nil && s.backendCmd.Process != nil {
		s.backendCmd.Process.Kill()
		s.backendCmd.Wait()
	}

	// Note: Postgres stays running for faster next suite startup
	fmt.Printf("✅ [%s] Cleanup complete (Postgres kept running)\n\n", s.SuiteName)
}

// CreateUser creates a user with optional name and email (delegates to testutil.CreateUser)
// Example: user := s.CreateUser() or admin := s.CreateUser("alice", "alice@example.com").Role("admin")
func (s *BaseSuite) CreateUser(nameAndEmail ...string) *UserClient {
	return CreateUser(s.T(), nameAndEmail...)
}

// Public returns a public (unauthenticated) client (delegates to testutil.Public)
func (s *BaseSuite) Public() *PublicClient {
	return Public(s.T())
}

// dropAllTables drops all tables in the test database for a clean state
func (s *BaseSuite) dropAllTables() {
	// Use psql to drop all tables
	cmd := exec.Command("docker", "exec", "chatgamelab-db-test",
		"psql", "-U", "chatgamelab", "-d", "chatgamelab", "-c",
		"DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO chatgamelab;")
	if err := cmd.Run(); err != nil {
		// If drop fails, fall back to restarting container
		fmt.Printf("⚠️  Failed to drop tables, restarting Postgres...\n")
		exec.Command("docker", "rm", "-f", "chatgamelab-db-test").Run()
		restartCmd := exec.Command("docker", "run", "-d",
			"--name", "chatgamelab-db-test",
			"-e", "POSTGRES_DB=chatgamelab",
			"-e", "POSTGRES_USER=chatgamelab",
			"-e", "POSTGRES_PASSWORD=testpassword",
			"-p", "7104:5432",
			"postgres:18")
		restartCmd.Stdout = os.Stdout
		restartCmd.Stderr = os.Stderr
		restartCmd.Run()
		time.Sleep(3 * time.Second)
	}
}

// waitForBackend waits for the backend to be healthy
func (s *BaseSuite) waitForBackend(maxRetries int) error {
	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := TestServerURL + "/api/status"

	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(time.Second)
	}

	return fmt.Errorf("backend not ready after %d retries", maxRetries)
}
