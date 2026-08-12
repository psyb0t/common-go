package postgresql

import (
	"embed"

	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/psyb0t/common-go/db"
	"github.com/psyb0t/ctxerrors"
)

func (p *Postgresql) migrationDriver() (database.Driver, error) {
	driver, err := pgx.WithInstance(p.SQLDB, &pgx.Config{})
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not create database driver")
	}

	return driver, nil
}

// MigrateUp applies all available migrations.
func (p *Postgresql) MigrateUp(path string, fs *embed.FS) error {
	driver, err := p.migrationDriver()
	if err != nil {
		return ctxerrors.Wrap(err, "create PostgreSQL migration driver")
	}

	return db.MigrateUp(p.config.Database, driver, path, fs)
}

// MigrateDown reverts the specified number of migrations.
func (p *Postgresql) MigrateDown(path string, steps int, fs *embed.FS) error {
	driver, err := p.migrationDriver()
	if err != nil {
		return ctxerrors.Wrap(err, "create PostgreSQL migration driver")
	}

	return db.MigrateDown(p.config.Database, driver, path, fs, steps)
}

// MigrateForce forces migration to a specific version.
func (p *Postgresql) MigrateForce(
	path string,
	version int,
	fs *embed.FS,
) error {
	driver, err := p.migrationDriver()
	if err != nil {
		return ctxerrors.Wrap(err, "create PostgreSQL migration driver")
	}

	return db.MigrateForce(p.config.Database, driver, path, fs, version)
}
