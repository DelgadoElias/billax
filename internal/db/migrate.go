package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations executes all pending database migrations from the given path.
// It uses golang-migrate/migrate to track applied migrations in the schema_migrations table.
// Safe to call multiple times — only unapplied migrations will run.
func RunMigrations(databaseURL, migrationsPath string) error {
	if databaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if migrationsPath == "" {
		migrationsPath = "migrations"
	}

	source := "file://" + migrationsPath
	m, err := migrate.New(source, databaseURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator (source: %s): %w", source, err)
	}
	defer m.Close()

	// Run all pending migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to read migration version: %w", err)
	}

	if dirty {
		return fmt.Errorf("migration state is dirty (version %d), manual intervention required", version)
	}

	return nil
}
