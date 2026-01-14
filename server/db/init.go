package db

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

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

// Init initializes the database connection. Call this at startup.
// If the database is empty (no tables), it will automatically initialize the schema.
func Init() {
	log.Debug("initializing database connection")
	_ = queries() // trigger lazy initialization

	// Check if database needs initialization
	if isEmpty, err := isDatabaseEmpty(); err != nil {
		log.Error("failed to check database state", "error", err)
		panic("failed to check database state: " + err.Error())
	} else if isEmpty {
		log.Info("database is empty, initializing schema")
		if err := initializeSchema(); err != nil {
			log.Error("failed to initialize database schema", "error", err)
			panic("failed to initialize database schema: " + err.Error())
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
		log.Error("failed to open postgres connection", "error", err)
		panic("failed to open postgres connection: " + err.Error())
	}

	if err = sqlDb.Ping(); err != nil {
		log.Error("failed to connect to postgres", "error", err)
		panic("failed to connect to postgres: " + err.Error())
	}

	log.Debug("postgres connection established")

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
	default:
		return obj.Role("invalid:" + s), errors.New("invalid role")
	}
}
