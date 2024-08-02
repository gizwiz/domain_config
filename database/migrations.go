package database

import (
	"database/sql"
	"embed"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
)

//go:embed migration/*
var migrationsFS embed.FS

func ApplyLatestDBMigrations(db *sql.DB) error {
	// Create a temporary directory to extract embedded migrations
	tempDir := "temp_migrations"
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return errors.Wrapf(err, "cannot mkdirall %s", tempDir)
	}
	defer os.RemoveAll(tempDir) // Clean up temporary directory

	// Extract embedded migrations to the temporary directory
	if err := extractMigrations(tempDir); err != nil {
		return errors.Wrapf(err, "cannot extractMigrations %s", tempDir)
	}

	// Initialize the SQLite3 driver
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return errors.Wrapf(err, "Error creating driver instance")
	}

	// Create a migrate instance
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+tempDir,
		"sqlite",
		driver,
	)
	if err != nil {
		return errors.Wrapf(err, "Error creating migrate instance")
	}

	// Apply the latest migrations
	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return errors.Wrapf(err, "Error applying migrations")
	}

	log.Println("Database migration completed successfully")
	return nil
}

func extractMigrations(targetDir string) error {
	// List embedded files in the "migration" directory
	entries, err := migrationsFS.ReadDir("migration")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		// Construct the source and target paths
		src := filepath.Join("migration", entry.Name())
		dest := filepath.Join(targetDir, entry.Name())

		// Open the source file from embedded FS
		srcFile, err := migrationsFS.Open(src)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		// Create the destination file
		destFile, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer destFile.Close()

		// Copy the content from source to destination
		if _, err := io.Copy(destFile, srcFile); err != nil {
			return err
		}
	}

	return nil
}
