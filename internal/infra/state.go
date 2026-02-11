package infra

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// DriverType identifies a supported database driver.
type DriverType int

const (
	Postgres DriverType = iota
	SQLite
)

// NewDB creates a database connection based on the driver type.
func NewDB(ctx context.Context, driver DriverType, dsn string) (*sql.DB, error) {
	switch driver {
	case Postgres:
		db, err := OpenDB(ctx, DBConfig{Driver: "pgx", DSN: dsn})
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(1 * time.Minute)
		return db, nil

	case SQLite:
		db, err := OpenDB(ctx, DBConfig{Driver: "sqlite", DSN: dsn})
		if err != nil {
			return nil, err
		}
		db.SetMaxOpenConns(1) // SQLite does not support concurrent writers
		return db, nil

	default:
		return nil, fmt.Errorf("unknown driver type: %d", driver)
	}
}
