package sqlitegorm

import (
	"path/filepath"
	"testing"

	commonsqlite "github.com/psyb0t/common-go/db/sqlite"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	modernsqlite "modernc.org/sqlite"
)

const testSchema = `CREATE TABLE parents (id INTEGER PRIMARY KEY);
CREATE TABLE entries (
 id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE,
 parent_id INTEGER REFERENCES parents(id), CHECK (id > 0)
);
INSERT INTO parents VALUES (1);
INSERT INTO entries VALUES (1, 'first', 1);`

func TestOpenTranslatesConstraints(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		statement string
		wantErr   error
	}{
		{
			"primary key",
			"INSERT INTO entries VALUES (1, 'second', 1)",
			gorm.ErrDuplicatedKey,
		},
		{
			"unique",
			"INSERT INTO entries VALUES (2, 'first', 1)",
			gorm.ErrDuplicatedKey,
		},
		{
			"foreign key",
			"INSERT INTO entries VALUES (2, 'second', 9)",
			gorm.ErrForeignKeyViolated,
		},
		{"check", "INSERT INTO entries VALUES (-1, 'second', 1)", nil},
		{"not null", "INSERT INTO entries VALUES (2, NULL, 1)", nil},
		{"syntax", "invalid SQL", nil},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			database := openTestDatabase(
				t, ":memory:?_pragma=foreign_keys(1)", true,
			)
			require.NoError(t, database.Exec(testSchema).Error)
			err := database.Exec(tc.statement).Error
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			var native *modernsqlite.Error
			require.ErrorAs(t, err, &native)
		})
	}
}

func TestOpenWithoutTranslationKeepsNativeError(t *testing.T) {
	t.Parallel()
	database := openTestDatabase(t, ":memory:", false)
	require.NoError(t, database.Exec(testSchema).Error)
	var native *modernsqlite.Error
	require.ErrorAs(t, database.Exec(
		"INSERT INTO entries VALUES (1, 'second', 1)").Error, &native)
}

func TestTranslatePreservesOtherErrors(t *testing.T) {
	t.Parallel()
	dialector := Open(":memory:")
	wrapped := ctxerrors.Wrap(commerr.ErrInvalidArgument, "caller error")
	for _, err := range []error{nil, commerr.ErrInvalidArgument, wrapped} {
		assert.Equal(t, err, dialector.Translate(err))
	}
}

func TestOpenReopensStoredDataAndUsesMigrationHelpers(t *testing.T) {
	t.Parallel()
	filename := filepath.Join(t.TempDir(), "shared.db")
	database := openTestDatabase(t, filename, true)
	connection, err := database.DB()
	require.NoError(t, err)
	migrations, err := filepath.Abs("../testdata/migrations")
	require.NoError(t, err)
	require.NoError(t, commonsqlite.MigrateUp(connection, migrations, nil))
	require.NoError(t, commonsqlite.MigrateUp(connection, migrations, nil))
	const value = "Unicode ☃ '; DROP TABLE widgets; --"
	require.NoError(t, database.Exec(
		"INSERT INTO widgets (id, name) VALUES (?, ?)", 1, value).Error)
	require.NoError(t, connection.Close())
	reopened := openTestDatabase(t, filename, true)
	var got string
	require.NoError(t, reopened.Raw(
		"SELECT name FROM widgets WHERE id = ?", 1).Scan(&got).Error)
	assert.Equal(t, value, got)
}

func TestOpenRejectsUnavailableDirectory(t *testing.T) {
	t.Parallel()
	_, err := gorm.Open(Open(filepath.Join(t.TempDir(), "missing", "db")),
		&gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.Error(t, err)
}

func openTestDatabase(t *testing.T, dsn string, translate bool) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(Open(dsn), &gorm.Config{
		TranslateError: translate,
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	connection, err := database.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	assert.IsType(t, &modernsqlite.Driver{}, connection.Driver())
	return database
}
