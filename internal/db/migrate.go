package db

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
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

	// Convert to absolute path if it doesn't start with /
	if migrationsPath[0] != '/' && migrationsPath[0] != '\\' {
		// Try to find migrations in common locations
		candidates := []string{
			migrationsPath,                    // Try relative path as-is
			"/app/" + migrationsPath,          // Docker path
			"./" + migrationsPath,             // Explicit relative
		}

		// Try each candidate to find the one that exists
		for _, candidate := range candidates {
			source := "file://" + candidate
			if m, err := migrate.New(source, databaseURL); err == nil {
				// Found a working source
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
		}

		// If we get here, none of the candidates worked
		return fmt.Errorf("migrations not found in any of the expected locations: %v", candidates)
	}

	// Use absolute path
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
