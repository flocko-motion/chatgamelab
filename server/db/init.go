package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	sqlc "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"

	_ "github.com/lib/pq" // Postgres driver
)

//go:embed schema.sql
var schemaSQL string

//go:embed migrations/*.sql
var migrationsFS embed.FS

var (
	sqlDb            *sql.DB       // shared *sql.DB
	queriesSingleton *sqlc.Queries // sqlc-generated Queries (see db/sqlc/db.go)
)

// Reset clears the database singleton (for testing only)
func Reset() {
	if sqlDb != nil {
		sqlDb.Close()
		sqlDb = nil
	}
	queriesSingleton = nil
}

// Init initializes the database connection. Call this at startup.
// If the database is empty (no tables), it will automatically initialize the schema.
// After initialization, it runs any pending migrations.
func Init() {
	log.Debug("initializing database connection")
	_ = queries() // trigger lazy initialization

	// Check if database needs initialization
	if isEmpty, err := isDatabaseEmpty(); err != nil {
		log.Fatal("failed to check database state", "error", err)
	} else if isEmpty {
		log.Info("database is empty, initializing schema")
		if err := initializeSchema(); err != nil {
			log.Fatal("failed to initialize database schema", "error", err)
		}
		log.Info("database schema initialized successfully")

		// Set schema_version to highest available migration (fresh DB is already up-to-date)
		if err := setInitialSchemaVersion(); err != nil {
			log.Fatal("failed to set initial schema version", "error", err)
		}
	} else {
		log.Debug("database already initialized")

		// Run any pending migrations
		if err := runPendingMigrations(); err != nil {
			log.Fatal("failed to run migrations", "error", err)
		}
	}

	log.Info("database connection initialized")
}

// queries returns the sqlc Queries singleton, initializing if needed.
func queries() *sqlc.Queries {
	if queriesSingleton != nil {
		return queriesSingleton
	}

	host := functional.EnvOrDefault("DB_HOST", "127.0.0.1")
	port := functional.RequireEnv("PORT_POSTGRES")
	dbName := functional.RequireEnv("DB_DATABASE")

	log.Debug("connecting to postgres", "host", host, "port", port, "database", dbName)

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		functional.RequireEnv("DB_USER"),
		functional.RequireEnv("DB_PASSWORD"),
		host,
		port,
		dbName)

	var err error
	sqlDb, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to open postgres connection", "error", err)
	}

	// Retry connection with exponential backoff (max 2 minutes)
	maxRetries := 24 // 25 attempts over ~2 minutes
	retryDelay := 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err = sqlDb.Ping()
		if err == nil {
			log.Debug("postgres connection established")
			break
		}

		if attempt == maxRetries {
			log.Fatal("failed to connect to postgres after retries", "error", err, "attempts", maxRetries)
		}

		log.Info("waiting for postgres to be ready", "attempt", attempt, "max_attempts", maxRetries, "retry_in", retryDelay)
		time.Sleep(retryDelay)
	}

	// New is defined in db/sqlc/db.go and returns *Queries.
	queriesSingleton = sqlc.New(sqlDb)
	return queriesSingleton
}

func sqlNullStringToMaybeString(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func sqlNullBoolToMaybeBool(nb sql.NullBool) *bool {
	if nb.Valid {
		return &nb.Bool
	}
	return nil
}

// isDatabaseEmpty checks if the database has any tables.
func isDatabaseEmpty() (bool, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_type = 'BASE TABLE'
	`
	err := sqlDb.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query table count: %w", err)
	}
	return count == 0, nil
}

// initializeSchema executes the embedded schema.sql to create all tables.
func initializeSchema() error {
	log.Debug("executing schema.sql", "length", len(schemaSQL))
	_, err := sqlDb.Exec(schemaSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema.sql: %w", err)
	}
	return nil
}

func stringToRole(s string) (obj.Role, error) {
	switch s {
	case string(obj.RoleAdmin):
		return obj.RoleAdmin, nil
	case string(obj.RoleHead):
		return obj.RoleHead, nil
	case string(obj.RoleStaff):
		return obj.RoleStaff, nil
	case string(obj.RoleParticipant):
		return obj.RoleParticipant, nil
	case string(obj.RoleIndividual):
		return obj.RoleIndividual, nil
	default:
		return obj.Role("invalid:" + s), errors.New("invalid role")
	}
}

// getAvailableMigrations returns a sorted list of migration files with their version numbers
func getAvailableMigrations() ([]struct {
	version  int
	filename string
}, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var migrations []struct {
		version  int
		filename string
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// Extract version number from filename (e.g., "001_add_feature.sql" -> 1)
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) < 2 {
			log.Warn("skipping migration file with invalid name format", "filename", entry.Name())
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			log.Warn("skipping migration file with invalid version number", "filename", entry.Name(), "error", err)
			continue
		}

		migrations = append(migrations, struct {
			version  int
			filename string
		}{version, entry.Name()})
	}

	// Sort by version number
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})

	return migrations, nil
}

// getHighestMigrationVersion returns the highest migration version available
func getHighestMigrationVersion() (int, error) {
	migrations, err := getAvailableMigrations()
	if err != nil {
		return 0, err
	}

	if len(migrations) == 0 {
		return 0, nil
	}

	return migrations[len(migrations)-1].version, nil
}

// getCurrentSchemaVersion reads the current schema version from system_settings
func getCurrentSchemaVersion() (int, error) {
	var version int
	query := `SELECT schema_version FROM system_settings WHERE id = '00000000-0000-0000-0000-000000000001'::uuid`
	err := sqlDb.QueryRow(query).Scan(&version)
	if err == sql.ErrNoRows {
		// No system_settings row exists yet (shouldn't happen after schema init, but handle gracefully)
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query schema version: %w", err)
	}
	return version, nil
}

// setSchemaVersion updates the schema version in system_settings
func setSchemaVersion(version int) error {
	query := `UPDATE system_settings SET schema_version = $1, modified_at = now() WHERE id = '00000000-0000-0000-0000-000000000001'::uuid`
	_, err := sqlDb.Exec(query, version)
	if err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}
	return nil
}

// setInitialSchemaVersion sets the schema version to the highest available migration
// This is called when initializing a fresh database that already has the full schema
func setInitialSchemaVersion() error {
	highestVersion, err := getHighestMigrationVersion()
	if err != nil {
		return err
	}

	log.Info("setting initial schema version", "version", highestVersion)
	return setSchemaVersion(highestVersion)
}

// runPendingMigrations executes all migrations that haven't been applied yet
func runPendingMigrations() error {
	currentVersion, err := getCurrentSchemaVersion()
	if err != nil {
		return err
	}

	migrations, err := getAvailableMigrations()
	if err != nil {
		return err
	}

	// Filter migrations that need to be applied
	var pendingMigrations []struct {
		version  int
		filename string
	}
	for _, m := range migrations {
		if m.version > currentVersion {
			pendingMigrations = append(pendingMigrations, m)
		}
	}

	if len(pendingMigrations) == 0 {
		log.Debug("no pending migrations")
		return nil
	}

	log.Info("found pending migrations", "count", len(pendingMigrations), "current_version", currentVersion)

	// Execute each migration in order
	for _, m := range pendingMigrations {
		if err := runMigration(m.version, m.filename); err != nil {
			return fmt.Errorf("migration %d (%s) failed: %w", m.version, m.filename, err)
		}
	}

	return nil
}

// runMigration executes a single migration file within a transaction
func runMigration(version int, filename string) error {
	log.Info("running migration", "version", version, "filename", filename)

	// Read migration file
	content, err := migrationsFS.ReadFile(filepath.Join("migrations", filename))
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Start transaction
	tx, err := sqlDb.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback if not committed

	// Execute migration SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Update schema version
	updateQuery := `UPDATE system_settings SET schema_version = $1, modified_at = now() WHERE id = '00000000-0000-0000-0000-000000000001'::uuid`
	if _, err := tx.Exec(updateQuery, version); err != nil {
		return fmt.Errorf("failed to update schema version: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Info("migration completed successfully", "version", version)
	return nil
}
