// Package sqlite provides SQLite migration helpers backed by golang-migrate.
package sqlite

import (
	"database/sql"
	"embed"

	"github.com/golang-migrate/migrate/v4/database"
	migrationsqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	commondb "github.com/psyb0t/common-go/db"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

const migrationDatabaseName = "sqlite"

// MigrateUp applies every pending migration. sqlDB remains owned by the caller
// and stays open after the migration completes.
func MigrateUp(sqlDB *sql.DB, path string, migrationFS *embed.FS) error {
	driver, err := migrationDriver(sqlDB)
	if err != nil {
		return ctxerrors.Wrap(err, "create SQLite migration driver")
	}

	return commondb.MigrateUp(migrationDatabaseName, driver, path, migrationFS)
}

// MigrateDown reverts the requested number of migrations. sqlDB remains owned
// by the caller and stays open after the migration completes.
func MigrateDown(
	sqlDB *sql.DB,
	path string,
	steps int,
	migrationFS *embed.FS,
) error {
	driver, err := migrationDriver(sqlDB)
	if err != nil {
		return ctxerrors.Wrap(err, "create SQLite migration driver")
	}

	return commondb.MigrateDown(
		migrationDatabaseName,
		driver,
		path,
		migrationFS,
		steps,
	)
}

// MigrateForce marks sqlDB at version without running a migration. sqlDB
// remains owned by the caller and stays open after the operation completes.
func MigrateForce(
	sqlDB *sql.DB,
	path string,
	version int,
	migrationFS *embed.FS,
) error {
	driver, err := migrationDriver(sqlDB)
	if err != nil {
		return ctxerrors.Wrap(err, "create SQLite migration driver")
	}

	return commondb.MigrateForce(
		migrationDatabaseName,
		driver,
		path,
		migrationFS,
		version,
	)
}

func migrationDriver(sqlDB *sql.DB) (database.Driver, error) {
	if sqlDB == nil {
		return nil, ctxerrors.Wrap(commerr.ErrRequiredFieldNotSet, "SQLite database")
	}

	driver, err := migrationsqlite.WithInstance(sqlDB, &migrationsqlite.Config{})
	if err != nil {
		return nil, ctxerrors.Wrap(err, "create SQLite migration driver")
	}

	return driver, nil
}
