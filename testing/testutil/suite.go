package testutil

import (
	"cgl/api"
	"cgl/api/client"
	"cgl/api/routes"
	"cgl/config"
	"cgl/db"
	"cgl/obj"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	lorem "github.com/drhodes/golorem"
	"github.com/stretchr/testify/suite"
)

// testLock ensures only one test suite runs at a time
var testLock sync.Mutex

// findAvailablePort finds an available TCP port
func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

// BaseSuite provides test environment management for test suites
// Embed this in your specific test suites to get automatic setup/teardown
type BaseSuite struct {
	suite.Suite
	SuiteName     string
	backendCancel context.CancelFunc
	postgresPort  int
	backendPort   int
	containerName string
	devUser       *UserClient // Default dev admin user for role assignments
	userRegistry  map[string]*UserClient
}

// SetupSuite runs once before all tests in the suite
// Starts Postgres in Docker and backend with 'go run'
func (s *BaseSuite) SetupSuite() {
	// Lock to prevent parallel suite execution
	testLock.Lock()

	if s.SuiteName == "" {
		s.SuiteName = "Test Suite"
	}

	// Initialize user registry
	s.userRegistry = make(map[string]*UserClient)

	// Find available ports for this test suite
	var err error
	s.postgresPort, err = findAvailablePort()
	if err != nil {
		s.T().Fatalf("Failed to find available Postgres port: %v", err)
	}
	s.backendPort, err = findAvailablePort()
	if err != nil {
		s.T().Fatalf("Failed to find available backend port: %v", err)
	}
	s.containerName = fmt.Sprintf("chatgamelab-db-test-%d", s.postgresPort)

	fmt.Printf("\n🚀 [%s] Starting test environment (Postgres:%d, Backend:%d)...\n",
		s.SuiteName, s.postgresPort, s.backendPort)

	// Clean up all stale test containers to prevent port conflicts
	fmt.Printf("🧹 [%s] Cleaning up stale test containers...\n", s.SuiteName)
	cleanupCmd := exec.Command("sh", "-c", "docker ps -a --filter 'name=chatgamelab-db-test' --format '{{.Names}}' | xargs -r docker rm -f")
	cleanupCmd.Run() // Ignore errors if no containers exist

	// Remove any stopped container with same name (redundant but safe)
	exec.Command("docker", "rm", "-f", s.containerName).Run()

	// Start Postgres with docker run on random port
	fmt.Printf("🐘 [%s] Starting Postgres on port %d...\n", s.SuiteName, s.postgresPort)
	cmd := exec.Command("docker", "run", "-d",
		"--name", s.containerName,
		"-e", "POSTGRES_DB=chatgamelab",
		"-e", "POSTGRES_USER=chatgamelab",
		"-e", "POSTGRES_PASSWORD=testpassword",
		"-p", fmt.Sprintf("%d:5432", s.postgresPort),
		"postgres:18")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Failed to start Postgres: %v", err)
	}

	// Wait for Postgres to be ready by pinging it
	fmt.Printf("⏳ [%s] Waiting for Postgres to be ready...\n", s.SuiteName)
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		pingCmd := exec.Command("docker", "exec", s.containerName, "pg_isready", "-U", "chatgamelab")
		if err := pingCmd.Run(); err == nil {
			fmt.Printf("✅ [%s] Postgres is ready!\n", s.SuiteName)
			// Give it a moment to fully stabilize and accept connections
			time.Sleep(3 * time.Second)
			break
		}
		if i == maxRetries-1 {
			s.TearDownSuite()
			s.T().Fatalf("Postgres did not become ready after %d attempts", maxRetries)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Start backend in-process AFTER database is cleaned
	// This ensures backend detects empty DB and initializes schema
	fmt.Printf("🚀 [%s] Starting backend in-process...\n", s.SuiteName)

	// Reset database singleton to clear any stale connections from previous test runs
	// This is critical for tests running in the same process
	db.Reset()

	// Set environment variables for backend
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_USER", "chatgamelab")
	os.Setenv("DB_PASSWORD", "testpassword")
	os.Setenv("DB_DATABASE", "chatgamelab")
	os.Setenv("PORT_BACKEND", fmt.Sprintf("%d", s.backendPort))
	os.Setenv("PORT_POSTGRES", fmt.Sprintf("%d", s.postgresPort))
	os.Setenv("DEV_MODE", "true")
	os.Setenv("DEV_JWT_SECRET", "testsecret123")
	os.Setenv("AUTH0_DOMAIN", "test.auth0.domain")
	os.Setenv("AUTH0_AUDIENCE", "test.auth0.audience")
	os.Setenv("PUBLIC_URL", fmt.Sprintf("http://localhost:%d", s.backendPort))

	// Update TestServerURL for this suite's backend port
	TestServerURL = fmt.Sprintf("http://localhost:%d", s.backendPort)

	// Create context for backend lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	s.backendCancel = cancel

	// Create ready channel
	readyChan := make(chan struct{})

	// Start backend in goroutine
	go func() {
		api.RunServer(ctx, s.backendPort, true, readyChan) // dynamic port, devMode true
	}()

	// Wait for backend to signal it's ready (DB initialized)
	fmt.Printf("⏳ [%s] Waiting for backend to initialize...\n", s.SuiteName)
	select {
	case <-readyChan:
		fmt.Printf("✅ [%s] Backend initialized, waiting for HTTP server...\n", s.SuiteName)
	case <-time.After(30 * time.Second):
		s.TearDownSuite()
		s.T().Fatalf("Backend initialization timed out")
	}

	// Now wait for HTTP server to be responsive
	if err := s.waitForBackend(10); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Backend HTTP server not ready: %v", err)
	}

	// Initialize dev user for role assignments
	// The dev user is created during preseed with UUID 00000000-0000-0000-0000-000000000000
	devUserID := "00000000-0000-0000-0000-000000000001"

	// Ensure client is configured with correct URL
	if err := config.SetServerConfig(TestServerURL, ""); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Failed to set server config: %v", err)
	}

	var jwtResponse struct {
		Token string `json:"token"`
	}
	if err := client.ApiGet("users/"+devUserID+"/jwt", &jwtResponse); err != nil {
		s.TearDownSuite()
		s.T().Fatalf("Failed to get dev user JWT: %v", err)
	}

	s.devUser = &UserClient{
		Name:  "dev",
		ID:    devUserID,
		Email: "dev@chatgamelab.local",
		Token: jwtResponse.Token,
		t:     s.T(),
	}

	fmt.Printf("✅ [%s] Backend ready for tests!\n\n", s.SuiteName)
}

// TearDownSuite runs once after all tests in the suite
func (s *BaseSuite) TearDownSuite() {
	defer testLock.Unlock()

	fmt.Printf("\n🧹 [%s] Cleaning up test environment...\n", s.SuiteName)

	// Stop backend only (keep Postgres running for next suite)
	if s.backendCancel != nil {
		s.backendCancel()
		time.Sleep(100 * time.Millisecond) // Give backend time to shutdown
	}

	// Note: Postgres stays running for faster next suite startup
	fmt.Printf("✅ [%s] Cleanup complete (Postgres kept running)\n\n", s.SuiteName)
}

// CreateUser creates a user with optional name and email
// If name is empty, generates a random name. If email is empty, generates email as <name>@test.local
// Example: user := s.CreateUser() or admin := s.CreateUser("alice").Role("admin")
func (s *BaseSuite) CreateUser(nameAndEmail ...string) *UserClient {
	s.T().Helper()

	// Parse optional name and email
	var name, email string
	if len(nameAndEmail) > 0 && nameAndEmail[0] != "" {
		name = nameAndEmail[0]
	} else {
		// Generate random name
		name = strings.ToLower(lorem.Word(1, 2))
	}

	if len(nameAndEmail) > 1 && nameAndEmail[1] != "" {
		email = nameAndEmail[1]
	} else {
		// Generate email from name
		email = name + "@test.local"
	}

	// Save current token to restore later
	oldToken, _ := config.GetJWT()

	// Clear auth temporarily to call dev endpoints
	if err := config.SetServerConfig(TestServerURL, ""); err != nil {
		s.T().Fatalf("failed to clear auth: %v", err)
	}

	// Create user via dev endpoint
	createPayload := map[string]interface{}{
		"name":  name,
		"email": &email,
	}

	var user struct {
		ID string `json:"id"`
	}
	if err := client.ApiPost("users/new", createPayload, &user); err != nil {
		s.T().Fatalf("failed to create user %q: %v", name, err)
	}

	// Get JWT for the user
	var jwtResponse struct {
		Token string `json:"token"`
	}
	if err := client.ApiGet("users/"+user.ID+"/jwt", &jwtResponse); err != nil {
		s.T().Fatalf("failed to get JWT for user %q: %v", name, err)
	}

	// Restore old token
	if oldToken != "" {
		if err := client.SaveJwt(oldToken); err != nil {
			s.T().Fatalf("failed to restore token: %v", err)
		}
	}

	// Create user client with suite reference
	userClient := &UserClient{
		Name:  name,
		ID:    user.ID,
		Email: email,
		Token: jwtResponse.Token,
		t:     s.T(),
	}

	s.T().Logf("Created user %q (ID: %s)", name, user.ID)
	return userClient
}

// Role sets the user's role
func (s *BaseSuite) Role(user *UserClient, role string) {

	payload := map[string]string{"role": role}
	var result interface{}

	if err := s.devUser.Post(fmt.Sprintf("users/%s/role", user.ID), payload, &result); err != nil {
		s.T().Fatalf("failed to set role %q: %v", role, err)
	}

	s.T().Logf("assigned role %q", role)
}

// Public returns a public (unauthenticated) client (delegates to testutil.Public)
func (s *BaseSuite) Public() *PublicClient {
	return Public(s.T())
}

// DevUser returns the default dev admin user
func (s *BaseSuite) DevUser() *UserClient {
	return s.devUser
}

// User retrieves a previously created user by name from the suite's registry
func (s *BaseSuite) User(name string) *UserClient {
	user, ok := s.userRegistry[name]
	if !ok {
		s.T().Fatalf("User %q not found. Create it first with CreateUser()", name)
	}
	return user
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
	healthURL := fmt.Sprintf("http://localhost:%d/api/status", s.backendPort)

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

// AcceptWorkshopInviteAnonymously accepts a workshop invite anonymously (no auth)
func (s *BaseSuite) AcceptWorkshopInviteAnonymously(token string) (*routes.AcceptInviteResponse, error) {
	s.T().Helper()

	// Use public client (no auth)
	publicClient := s.Public()

	var response routes.AcceptInviteResponse
	err := publicClient.Post("invites/"+token+"/accept", nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CreateUserWithToken creates a UserClient using only an auth token (for participants)
func (s *BaseSuite) CreateUserWithToken(authToken string) *UserClient {
	s.T().Helper()

	// Create client with the token to fetch user info
	client := &UserClient{
		Token: authToken,
		t:     s.T(),
	}

	// Get user info using the token
	var user obj.User
	if err := client.Get("users/me", &user); err != nil {
		s.T().Fatalf("failed to get user info with token: %v", err)
	}

	// Return fully populated client
	return &UserClient{
		Name:  user.Name,
		ID:    user.ID.String(),
		Email: "", // Anonymous users don't have email
		Token: authToken,
		t:     s.T(),
	}
}

// DeleteUser deletes a user by ID (for removing participants)
func (u *UserClient) DeleteUser(userID string) error {
	u.t.Helper()
	return u.Delete("users/" + userID)
}
