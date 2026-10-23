package postgresql

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testImage = "postgres:16-alpine@sha256:f4610795587492cd4fb887c2dc3de301b813278bb6f616992061141b5f15ac36"

//go:embed testdata/*/*.sql
var migrationFS embed.FS

var serverConfig Config

func TestMain(m *testing.M) {
	os.Exit(runMigrationTests(m))
}

func runMigrationTests(m *testing.M) (code int) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	password := uuid.NewString()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: testImage, Name: "common-go-postgres-test-" + uuid.NewString(),
			Env:          map[string]string{"POSTGRES_PASSWORD": password},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForSQL("5432/tcp", "pgx", func(host, port string) string {
				return "postgres://postgres:" + password + "@" + net.JoinHostPort(host, strings.TrimSuffix(port, "/tcp")) + "/postgres?sslmode=disable"
			}).WithStartupTimeout(2 * time.Minute),
		}, Started: true,
	})
	if container != nil {
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
			defer cleanupCancel()
			if cleanupErr := container.Terminate(cleanupCtx); cleanupErr != nil {
				fmt.Fprintln(os.Stderr, "stop PostgreSQL test server:", cleanupErr)
				code = 1
			}
		}()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "start PostgreSQL test server:", err)
		return 1
	}
	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve PostgreSQL host:", err)
		return 1
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve PostgreSQL port:", err)
		return 1
	}
	number, err := strconv.Atoi(port.Port())
	if err != nil {
		fmt.Fprintln(os.Stderr, "parse PostgreSQL port:", err)
		return 1
	}
	serverConfig = Config{Hostname: host, Port: number, Username: "postgres", Password: password}
	return m.Run()
}

func newTestDatabase(t *testing.T) *Postgresql {
	t.Helper()
	cfg := serverConfig
	cfg.Database = "migration_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	connection, err := NewWithConfig(t.Context(), cfg, true)
	require.NoError(t, err)
	connection.SQLDB.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	return connection
}

func requireReleasedPool(t *testing.T, connection *Postgresql) {
	t.Helper()
	require.Zero(t, connection.SQLDB.Stats().InUse, "migration retained the only connection")
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	require.NoError(t, connection.SQLDB.PingContext(ctx))
}

func TestMigrationPoolLifecycle(t *testing.T) {
	t.Parallel()
	for _, filesystem := range []bool{false, true} {
		t.Run(fmt.Sprintf("filesystem=%t", filesystem), func(t *testing.T) {
			t.Parallel()
			connection := newTestDatabase(t)
			path, source := "testdata/migrations", &migrationFS
			if filesystem {
				var err error
				path, err = filepath.Abs(path)
				require.NoError(t, err)
				source = nil
			}
			require.NoError(t, connection.MigrateUp(path, source))
			requireReleasedPool(t, connection)
			require.NoError(t, connection.MigrateUp(path, source))
			requireReleasedPool(t, connection)
			var name string
			require.NoError(t, connection.SQLDB.QueryRowContext(t.Context(), "SELECT name FROM widgets WHERE id = 1").Scan(&name))
			require.Equal(t, "example", name)
			var widget struct{ Name string }
			require.NoError(t, connection.GormDB.WithContext(t.Context()).Table("widgets").First(&widget).Error)
			require.Equal(t, name, widget.Name)
			require.NoError(t, connection.MigrateDown(path, 1, source))
			requireReleasedPool(t, connection)
			require.NoError(t, connection.MigrateUp(path, source))
			requireReleasedPool(t, connection)
			require.NoError(t, connection.MigrateForce(path, 1, source))
			requireReleasedPool(t, connection)
			for _, steps := range []int{0, -1} {
				require.ErrorIs(t, connection.MigrateDown(path, steps, source), commerr.ErrInvalidArgument)
				requireReleasedPool(t, connection)
			}
		})
	}
}

func TestMigrationPoolFailures(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	require.ErrorIs(t, connection.MigrateUp("", nil), commerr.ErrEmptyMigrationsPath)
	requireReleasedPool(t, connection)
	require.Error(t, connection.MigrateUp(filepath.Join(t.TempDir(), "absent"), nil))
	requireReleasedPool(t, connection)
	require.ErrorIs(t, connection.MigrateDown("", 1, nil), commerr.ErrEmptyMigrationsPath)
	requireReleasedPool(t, connection)
	require.ErrorIs(t, connection.MigrateForce("", 1, nil), commerr.ErrEmptyMigrationsPath)
	requireReleasedPool(t, connection)
	require.Error(t, connection.MigrateUp("testdata/broken", &migrationFS))
	requireReleasedPool(t, connection)
	require.Error(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	requireReleasedPool(t, connection)
	require.NoError(t, connection.MigrateForce("testdata/migrations", 0, &migrationFS))
	requireReleasedPool(t, connection)
	require.Error(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	requireReleasedPool(t, connection)
	require.NoError(t, connection.MigrateForce("testdata/migrations", -1, &migrationFS))
	requireReleasedPool(t, connection)
	require.NoError(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	requireReleasedPool(t, connection)
	_, err := connection.SQLDB.ExecContext(t.Context(), "SET search_path TO unavailable_schema")
	require.NoError(t, err)
	require.Error(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	requireReleasedPool(t, connection)
}

func TestConcurrentMigrationsReleasePool(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	const callers = 4
	results := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Go(func() { results <- connection.MigrateUp("testdata/migrations", &migrationFS) })
	}
	group.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	requireReleasedPool(t, connection)
}

func TestMissingMigrationPool(t *testing.T) {
	t.Parallel()
	connection := &Postgresql{}
	require.ErrorIs(t, connection.MigrateUp("testdata/migrations", &migrationFS), commerr.ErrRequiredFieldNotSet)
	require.ErrorIs(t, connection.MigrateDown("testdata/migrations", 1, &migrationFS), commerr.ErrRequiredFieldNotSet)
	require.ErrorIs(t, connection.MigrateForce("testdata/migrations", 1, &migrationFS), commerr.ErrRequiredFieldNotSet)
}

func TestClosedMigrationPool(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	require.NoError(t, connection.Close())
	require.Error(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	require.Error(t, connection.MigrateDown("testdata/migrations", 1, &migrationFS))
	require.Error(t, connection.MigrateForce("testdata/migrations", 1, &migrationFS))
}
