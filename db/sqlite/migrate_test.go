package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/require"
)

const migrationsPath = "testdata/migrations"

//go:embed testdata/migrations/*.sql
var migrationsFS embed.FS

func TestMigrateUpDownAndForce(t *testing.T) {
	sqlDB := openTestDatabase(t)

	require.NoError(t, MigrateUp(sqlDB, migrationsPath, &migrationsFS))
	require.NoError(t, MigrateUp(sqlDB, migrationsPath, &migrationsFS))
	assertTableExists(t, sqlDB)

	require.NoError(t, MigrateDown(sqlDB, migrationsPath, 1, &migrationsFS))
	assertTableDoesNotExist(t, sqlDB)

	require.NoError(t, MigrateUp(sqlDB, migrationsPath, &migrationsFS))
	require.NoError(t, MigrateForce(sqlDB, migrationsPath, 1, &migrationsFS))
	require.NoError(t, sqlDB.Ping())
}

func TestMigrateDownRejectsNonPositiveSteps(t *testing.T) {
	sqlDB := openTestDatabase(t)

	err := MigrateDown(sqlDB, migrationsPath, 0, &migrationsFS)
	require.ErrorIs(t, err, commerr.ErrInvalidArgument)
}

func TestMigrateUpRejectsNilDatabase(t *testing.T) {
	err := MigrateUp(nil, migrationsPath, &migrationsFS)
	require.ErrorIs(t, err, commerr.ErrRequiredFieldNotSet)
}

func TestMigrateUpFromFilesystem(t *testing.T) {
	sqlDB := openTestDatabase(t)

	workingDirectory, err := os.Getwd()
	require.NoError(t, err)

	path := filepath.Join(workingDirectory, migrationsPath)
	require.NoError(t, MigrateUp(sqlDB, path, nil))
	assertTableExists(t, sqlDB)
}

func openTestDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	sqlDB, err := sql.Open(migrationDatabaseName, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
	})

	return sqlDB
}

func assertTableExists(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	var tableName string
	err := sqlDB.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'widgets'",
	).Scan(&tableName)
	require.NoError(t, err)
	require.Equal(t, "widgets", tableName)
}

func assertTableDoesNotExist(t *testing.T, sqlDB *sql.DB) {
	t.Helper()

	var tableName string
	err := sqlDB.QueryRow(
		"SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'widgets'",
	).Scan(&tableName)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
