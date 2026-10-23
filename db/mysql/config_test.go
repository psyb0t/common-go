package mysql

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		edit func(*Config)
		want error
	}{
		{"hostname", func(c *Config) { c.Hostname = " " }, commerr.ErrRequiredFieldNotSet},
		{"username", func(c *Config) { c.Username = "" }, commerr.ErrRequiredFieldNotSet},
		{"database", func(c *Config) { c.Database = "" }, commerr.ErrRequiredFieldNotSet},
		{"zero port", func(c *Config) { c.Port = 0 }, commerr.ErrInvalidArgument},
		{"negative port", func(c *Config) { c.Port = -1 }, commerr.ErrInvalidArgument},
		{"large port", func(c *Config) { c.Port = 65536 }, commerr.ErrInvalidArgument},
		{"NUL", func(c *Config) { c.Database = "a\x00b" }, commerr.ErrInvalidArgument},
		{"long name", func(c *Config) { c.Database = strings.Repeat("x", 65) }, commerr.ErrInvalidArgument},
		{"max name", func(c *Config) { c.Database = strings.Repeat("界", 64) }, nil},
		{"empty password", func(c *Config) { c.Password = "" }, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := Config{Hostname: "localhost", Port: 3306, Username: "user", Database: "example"}
			tc.edit(&cfg)
			require.ErrorIs(t, cfg.validate(), tc.want)
			if tc.want != nil {
				_, err := NewWithConfig(t.Context(), cfg, false)
				require.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestParseConfig(t *testing.T) {
	for key, value := range map[string]string{
		"DB_MYSQL_HOSTNAME": "localhost", "DB_MYSQL_USERNAME": "example",
		"DB_MYSQL_DATABASE": "example", "DB_MYSQL_PASSWORD": "", "DB_MYSQL_PORT": "", "DB_MYSQL_ISSSL": "",
	} {
		t.Setenv(key, value)
	}
	require.NoError(t, os.Unsetenv("DB_MYSQL_PORT"))
	require.NoError(t, os.Unsetenv("DB_MYSQL_ISSSL"))
	cfg, err := parseConfig()
	require.NoError(t, err)
	require.Equal(t, 3306, cfg.Port)
	require.True(t, cfg.IsSSL)
	t.Setenv("DB_MYSQL_PORT", "not-an-integer")
	_, err = New(t.Context(), false)
	require.Error(t, err)
	t.Setenv("DB_MYSQL_PORT", "3306")
	t.Setenv("DB_MYSQL_ISSSL", "not-a-boolean")
	_, err = New(t.Context(), false)
	require.Error(t, err)
}

func TestDriverConfig(t *testing.T) {
	t.Parallel()
	cfg := Config{Hostname: "::1", Port: 3306, Username: "example", Password: "@:/?&tls=false", Database: "例子", IsSSL: true}
	parsed, err := mysqldriver.ParseDSN(driverConfig(cfg, cfg.Database, false).FormatDSN())
	require.NoError(t, err)
	require.Equal(t, cfg.Password, parsed.Passwd)
	require.Equal(t, cfg.Database, parsed.DBName)
	require.Equal(t, "[::1]:3306", parsed.Addr)
	require.Equal(t, "true", parsed.TLSConfig)
	require.False(t, parsed.TLS.InsecureSkipVerify)
	require.False(t, parsed.AllowFallbackToPlaintext)
	require.False(t, parsed.MultiStatements)
	require.True(t, parsed.ParseTime)
	require.Equal(t, time.UTC, parsed.Loc)
	require.True(t, driverConfig(cfg, cfg.Database, true).MultiStatements)
}

func TestNilHandles(t *testing.T) {
	t.Parallel()
	var connection *Mysql
	require.NoError(t, connection.Close())
	require.ErrorIs(t, connection.MigrateUp(".", nil), commerr.ErrRequiredFieldNotSet)
	require.ErrorIs(t, MigrateUp(nil, ".", nil), commerr.ErrRequiredFieldNotSet)
	require.ErrorIs(t, MigrateDown(nil, ".", 0, nil), commerr.ErrInvalidArgument)
	require.ErrorIs(t, MigrateDown(nil, ".", -1, nil), commerr.ErrInvalidArgument)
	require.ErrorIs(t, MigrateDown(nil, ".", 1, nil), commerr.ErrRequiredFieldNotSet)
	require.ErrorIs(t, MigrateForce(nil, ".", 0, nil), commerr.ErrRequiredFieldNotSet)
}

func TestCanceledConnection(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := NewWithConfig(ctx, Config{Hostname: "127.0.0.1", Port: 1, Username: "example", Database: "example"}, false)
	require.ErrorIs(t, err, context.Canceled)
}
