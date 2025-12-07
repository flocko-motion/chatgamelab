package db

import (
	"database/sql"
	"errors"
	"os"

	sqlc "webapp-server/db/sqlc"
	"webapp-server/obj"

	_ "github.com/lib/pq" // Postgres driver
)

var (
	sqlDb            *sql.DB       // shared *sql.DB
	queriesSingleton *sqlc.Queries // sqlc-generated Queries (see db/sqlc/db.go)
)

// Init initializes the global SQLDB and Queries using a Postgres connection.
//
// It reads the DSN from DATABASE_URL or DB_DSN. If both are empty, it falls
// back to a local development DSN; adjust as needed for your environment.
func queries() *sqlc.Queries {
	if queriesSingleton != nil {
		return queriesSingleton
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		// Fallback for local development; change to match your setup.
		dsn = "postgres://postgres:postgres@localhost:5432/chatgamelab?sslmode=disable"
	}

	var err error
	sqlDb, err = sql.Open("postgres", dsn)
	if err != nil {
		panic("failed to open postgres connection: " + err.Error())
	}

	if err = sqlDb.Ping(); err != nil {
		panic("failed to connect to postgres: " + err.Error())
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
