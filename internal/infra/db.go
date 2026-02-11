package infra

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DBConfig holds database connection parameters.
type DBConfig struct {
	Driver string // "pgx" for PostgreSQL, "sqlite" for SQLite
	DSN    string // connection string or ":memory:" for in-memory SQLite
}

// OpenDB creates and validates a database connection.
func OpenDB(ctx context.Context, cfg DBConfig) (*sql.DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
