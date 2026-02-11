package infra

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations applies all pending up-migrations from the given directory.
func RunMigrations(db *sql.DB, driver string, migrationsPath string) error {
	var (
		dbDriver database.Driver
		err      error
	)

	switch driver {
	case "pgx":
		dbDriver, err = postgres.WithInstance(db, &postgres.Config{})
	case "sqlite":
		dbDriver, err = sqlite.WithInstance(db, &sqlite.Config{})
	default:
		return fmt.Errorf("unsupported migration driver: %s", driver)
	}
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		driver,
		dbDriver,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
