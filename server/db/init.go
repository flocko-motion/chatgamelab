package db

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	sqlc "cgl/db/sqlc"
	"cgl/functional"
	"cgl/log"
	"cgl/obj"

	_ "github.com/lib/pq" // Postgres driver
)

//go:embed schema.sql
var schemaSQL string

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
	} else {
		log.Debug("database already initialized")
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
	default:
		return obj.Role("invalid:" + s), errors.New("invalid role")
	}
}
