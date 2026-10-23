package mysql

import (
	"context"
	"database/sql"
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

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"
)

const testImage = "mysql:8.4.7@sha256:0426ec38c7a10aa45ba383887df7878f74ee70e2fd589c7b69207f3577901903"

//go:embed testdata/*/*.sql
var migrationFS embed.FS

var serverConfig Config

func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	password := uuid.NewString()
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testImage,
			Name:         "common-go-mysql-test-" + uuid.NewString(),
			Env:          map[string]string{"MYSQL_ROOT_PASSWORD": password, "MYSQL_ROOT_HOST": "%"},
			ExposedPorts: []string{"3306/tcp"},
			WaitingFor: wait.ForSQL("3306/tcp", driverName, func(host, port string) string {
				config := driverConfig(Config{Username: "root", Password: password}, "", false)
				config.Addr = net.JoinHostPort(host, strings.TrimSuffix(port, "/tcp"))
				return config.FormatDSN()
			}).WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "start MySQL test server:", err)
		if container != nil {
			if cleanupErr := container.Terminate(context.Background()); cleanupErr != nil {
				fmt.Fprintln(os.Stderr, "clean up MySQL test server:", cleanupErr)
			}
		}
		cancel()
		os.Exit(1)
	}
	serverConfig.Hostname, err = container.Host(ctx)
	if err != nil {
		panic(err)
	}
	port, err := container.MappedPort(ctx, "3306/tcp")
	if err != nil {
		panic(err)
	}
	serverConfig.Port, err = strconv.Atoi(port.Port())
	if err != nil {
		panic(err)
	}
	serverConfig.Username = "root"
	serverConfig.Password = password
	cancel()

	code := m.Run()
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), time.Minute)
	if err := container.Terminate(cleanupCtx); err != nil {
		fmt.Fprintln(os.Stderr, "stop MySQL test server:", err)
		code = 1
	}
	cleanupCancel()
	os.Exit(code)
}

func newTestDatabase(t *testing.T) *Mysql {
	t.Helper()
	cfg := serverConfig
	cfg.Database = "wrapper_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	connection, err := NewWithConfig(t.Context(), cfg, true)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
	})
	return connection
}

func TestConnectionAndMigrations(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	sqlDB, err := connection.GormDB.DB()
	require.NoError(t, err)
	require.Same(t, connection.SQLDB, sqlDB)
	require.NoError(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	require.NoError(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	var row struct {
		ID   int64
		Name string
	}
	require.NoError(t, connection.GormDB.WithContext(t.Context()).Table("widgets").First(&row).Error)
	require.Equal(t, "example", row.Name)

	duplicate := connection.GormDB.WithContext(t.Context()).Table("widgets").Create(&row).Error
	require.ErrorIs(t, duplicate, gorm.ErrDuplicatedKey)
	_, err = connection.SQLDB.ExecContext(t.Context(), "SELECT 1; SELECT 2")
	require.Error(t, err, "normal application pool must not allow multi-statements")
	require.NoError(t, connection.MigrateDown("testdata/migrations", 1, &migrationFS))
	_, err = connection.SQLDB.ExecContext(t.Context(), "SELECT * FROM widgets")
	require.Error(t, err)
	require.NoError(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	require.NoError(t, connection.MigrateForce("testdata/migrations", 1, &migrationFS))
	require.NoError(t, connection.SQLDB.PingContext(t.Context()))

	second, err := NewWithConfig(t.Context(), connection.config, true)
	require.NoError(t, err)
	require.NoError(t, second.Close())
	require.NoError(t, connection.Close())
	require.Error(t, connection.SQLDB.PingContext(t.Context()))
	require.Error(t, connection.MigrateUp("testdata/migrations", &migrationFS))
}

func TestCallerOwnedPoolAndFilesystem(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	connector, err := mysqldriver.NewConnector(driverConfig(connection.config, connection.config.Database, true))
	require.NoError(t, err)
	pool := sql.OpenDB(connector)
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	path, err := filepath.Abs("testdata/migrations")
	require.NoError(t, err)
	require.NoError(t, MigrateUp(pool, path, nil))
	require.NoError(t, MigrateUp(pool, path, nil))
	require.NoError(t, MigrateDown(pool, path, 1, nil))
	require.NoError(t, MigrateUp(pool, path, nil))
	require.NoError(t, MigrateForce(pool, path, 1, nil))
	require.Zero(t, pool.Stats().InUse)
	require.NoError(t, pool.PingContext(t.Context()))
	require.Error(t, MigrateUp(pool, "missing", &migrationFS))
	require.Zero(t, pool.Stats().InUse)
	require.Error(t, MigrateUp(pool, filepath.Join(t.TempDir(), "missing"), nil))
	require.NoError(t, pool.Close())
	require.Error(t, MigrateUp(pool, path, nil))
}

func TestDirtyMigrationRecovery(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	require.Error(t, connection.MigrateUp("testdata/broken", &migrationFS))
	var dirty migrate.ErrDirty
	require.ErrorAs(t, connection.MigrateUp("testdata/migrations", &migrationFS), &dirty)
	require.NoError(t, connection.MigrateForce("testdata/migrations", -1, &migrationFS))
	require.NoError(t, connection.MigrateUp("testdata/migrations", &migrationFS))
	require.NoError(t, connection.SQLDB.PingContext(t.Context()))
}

func TestConcurrentMigrations(t *testing.T) {
	t.Parallel()
	connection := newTestDatabase(t)
	var group sync.WaitGroup
	errors := make(chan error, 3)
	for range cap(errors) {
		group.Go(func() { errors <- connection.MigrateUp("testdata/migrations", &migrationFS) })
	}
	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	var count int
	require.NoError(t, connection.SQLDB.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM widgets").Scan(&count))
	require.Equal(t, 1, count)
}

func TestDatabaseCreationAndFailures(t *testing.T) {
	t.Parallel()
	cfg := serverConfig
	cfg.Database = "quoted`name_" + uuid.NewString()
	connection, err := NewWithConfig(t.Context(), cfg, true)
	require.NoError(t, err)
	require.NoError(t, connection.SQLDB.PingContext(t.Context()))
	require.NoError(t, connection.Close())

	for _, tc := range []struct {
		name   string
		edit   func(*Config)
		ensure bool
	}{
		{"missing database", func(c *Config) { c.Database = "missing_" + uuid.NewString() }, false},
		{"bad password", func(c *Config) { c.Password = uuid.NewString() }, false},
		{"bad creation password", func(c *Config) { c.Password = uuid.NewString() }, true},
		{"untrusted TLS", func(c *Config) { c.IsSSL = true }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			config := cfg
			tc.edit(&config)
			ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
			defer cancel()
			_, err := NewWithConfig(ctx, config, tc.ensure)
			require.Error(t, err)
		})
	}
}

func TestEnvironmentConnection(t *testing.T) {
	connection := newTestDatabase(t)
	for key, value := range map[string]string{
		"DB_MYSQL_HOSTNAME": connection.config.Hostname,
		"DB_MYSQL_PORT":     strconv.Itoa(connection.config.Port),
		"DB_MYSQL_USERNAME": connection.config.Username,
		"DB_MYSQL_PASSWORD": connection.config.Password,
		"DB_MYSQL_DATABASE": connection.config.Database,
		"DB_MYSQL_ISSSL":    "false",
	} {
		t.Setenv(key, value)
	}
	fromEnvironment, err := New(t.Context(), false)
	require.NoError(t, err)
	require.NoError(t, fromEnvironment.Close())
}
