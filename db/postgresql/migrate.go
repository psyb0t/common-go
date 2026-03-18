package postgresql

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file" // needed for file:// scheme
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/psyb0t/ctxerrors"
)

func (p *Postgresql) createMigrator(
	path string,
	fs *embed.FS,
) (*migrate.Migrate, error) {
	driver, err := pgx.WithInstance(p.SQLDB, &pgx.Config{})
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not create database driver")
	}

	if fs != nil {
		sourceDriver, err := iofs.New(fs, path)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "could not create iofs driver")
		}

		m, err := migrate.NewWithInstance(
			"iofs",
			sourceDriver,
			p.config.Database,
			driver,
		)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "could not create migrate instance")
		}

		return m, nil
	}

	if path == "" {
		return nil, ctxerrors.Wrap(ErrMigrationsPathRequired, "path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		return nil, ctxerrors.Wrap(err, "error validating migrations path")
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", path),
		p.config.Database,
		driver,
	)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "could not create migrate instance")
	}

	return m, nil
}

// MigrateUp applies all available migrations.
func (p *Postgresql) MigrateUp(path string, fs *embed.FS) error {
	m, err := p.createMigrator(path, fs)
	if err != nil {
		return ctxerrors.Wrap(err, "could not create migrator")
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database is already up to date")

			return nil
		}

		return ctxerrors.Wrap(err, "could not migrate up")
	}

	return nil
}

// MigrateDown reverts the specified number of migrations.
func (p *Postgresql) MigrateDown(path string, steps int, fs *embed.FS) error {
	if steps <= 0 {
		return ctxerrors.Wrap(ErrInvalidSteps, "steps must be greater than 0")
	}

	m, err := p.createMigrator(path, fs)
	if err != nil {
		return ctxerrors.Wrap(err, "could not create migrator")
	}

	if err := m.Steps(-steps); err != nil {
		return ctxerrors.Wrap(err, "could not migrate down")
	}

	return nil
}

// MigrateForce forces migration to a specific version.
func (p *Postgresql) MigrateForce(
	path string,
	version int,
	fs *embed.FS,
) error {
	m, err := p.createMigrator(path, fs)
	if err != nil {
		return ctxerrors.Wrap(err, "could not create migrator")
	}

	if err := m.Force(version); err != nil {
		return ctxerrors.Wrap(err, "could not force version")
	}

	return nil
}
