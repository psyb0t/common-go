package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func setupRedisCache(
	t *testing.T,
) (*Redis, func()) {
	t.Helper()

	ctx := context.Background()

	container, err := tcredis.Run(
		ctx, "redis:7-alpine",
	)
	if err != nil {
		t.Skipf("redis container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redis.ParseURL(connStr)
	require.NoError(t, err)

	client := redis.NewClient(opts)

	cleanup := func() {
		_ = client.Close()
		_ = container.Terminate(ctx)
	}

	return NewRedis(client), cleanup
}

func TestRedis_GetSet(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value []byte
	}{
		{
			name:  "simple value",
			key:   "foo",
			value: []byte("bar"),
		},
		{
			name:  "binary data",
			key:   "bin",
			value: []byte{0x00, 0xFF, 0xAB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Set(
				ctx, tt.key, tt.value, time.Minute,
			)
			require.NoError(t, err)

			got, err := c.Get(ctx, tt.key)
			require.NoError(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestRedis_CacheMiss(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	_, err := c.Get(context.Background(), "nope")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestRedis_TTLExpiry(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	ctx := context.Background()

	err := c.Set(
		ctx, "exp", []byte("soon"),
		200*time.Millisecond,
	)
	require.NoError(t, err)

	got, err := c.Get(ctx, "exp")
	require.NoError(t, err)
	assert.Equal(t, []byte("soon"), got)

	time.Sleep(300 * time.Millisecond)

	_, err = c.Get(ctx, "exp")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestRedis_Delete(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	ctx := context.Background()

	_ = c.Set(ctx, "del", []byte("me"), time.Minute)

	err := c.Delete(ctx, "del")
	require.NoError(t, err)

	_, err = c.Get(ctx, "del")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestRedis_Overwrite(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	ctx := context.Background()

	_ = c.Set(ctx, "k", []byte("old"), time.Minute)
	_ = c.Set(ctx, "k", []byte("new"), time.Minute)

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), got)
}

func TestRedis_Close(t *testing.T) {
	c, cleanup := setupRedisCache(t)
	defer cleanup()

	err := c.Close()
	require.NoError(t, err)
}

func TestRedis_KeyPrefix(t *testing.T) {
	ctx := context.Background()

	container, err := tcredis.Run(
		ctx, "redis:7-alpine",
	)
	if err != nil {
		t.Skipf("redis container: %v", err)
	}

	defer func() { _ = container.Terminate(ctx) }()

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	opts, err := redis.ParseURL(connStr)
	require.NoError(t, err)

	client := redis.NewClient(opts)

	defer func() { _ = client.Close() }()

	c := NewRedisWithPrefix(client, "proxq:")

	err = c.Set(
		ctx, "mykey", []byte("val"), time.Minute,
	)
	require.NoError(t, err)

	got, err := c.Get(ctx, "mykey")
	require.NoError(t, err)
	assert.Equal(t, []byte("val"), got)

	rawVal, err := client.Get(
		ctx, "proxq:cache:mykey",
	).Bytes()
	require.NoError(t, err)
	assert.Equal(t, []byte("val"), rawVal)

	_, err = client.Get(ctx, "mykey").Result()
	assert.ErrorIs(t, err, redis.Nil)

	_, err = client.Get(ctx, "cache:mykey").Result()
	assert.ErrorIs(t, err, redis.Nil)
}

func TestRedis_BuildKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "no prefix",
			prefix:   "",
			key:      "foo",
			expected: "cache:foo",
		},
		{
			name:     "with prefix",
			prefix:   "proxq:",
			key:      "bar",
			expected: "proxq:cache:bar",
		},
		{
			name:     "nested prefix",
			prefix:   "app:svc:",
			key:      "key",
			expected: "app:svc:cache:key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Redis{keyPrefix: tt.prefix}
			assert.Equal(t, tt.expected, r.buildKey(tt.key))
		})
	}
}
