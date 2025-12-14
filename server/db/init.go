package db

import (
	"database/sql"
	"errors"
	"fmt"

	sqlc "cgl/db/sqlc"
	"cgl/functional"
	"cgl/obj"

	_ "github.com/lib/pq" // Postgres driver
)

var (
	sqlDb            *sql.DB       // shared *sql.DB
	queriesSingleton *sqlc.Queries // sqlc-generated Queries (see db/sqlc/db.go)
)

// Init initializes the database connection. Call this at startup.
func Init() {
	_ = queries() // trigger lazy initialization
}

// queries returns the sqlc Queries singleton, initializing if needed.
func queries() *sqlc.Queries {
	if queriesSingleton != nil {
		return queriesSingleton
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		functional.RequireEnv("DB_USER"),
		functional.RequireEnv("DB_PASSWORD"),
		functional.EnvOrDefault("DB_HOST", "127.0.0.1"),
		functional.RequireEnv("PORT_POSTGRES"),
		functional.RequireEnv("DB_DATABASE"))

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
