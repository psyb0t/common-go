package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemory_GetSet(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

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
			name:  "empty value",
			key:   "empty",
			value: []byte{},
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

func TestMemory_CacheMiss(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

	_, err := c.Get(ctx, "nonexistent")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestMemory_TTLExpiry(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

	err := c.Set(
		ctx, "expires", []byte("soon"), 50*time.Millisecond,
	)
	require.NoError(t, err)

	got, err := c.Get(ctx, "expires")
	require.NoError(t, err)
	assert.Equal(t, []byte("soon"), got)

	time.Sleep(60 * time.Millisecond)

	_, err = c.Get(ctx, "expires")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestMemory_Eviction(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(2)

	defer func() { _ = c.Close() }()

	_ = c.Set(ctx, "a", []byte("1"), time.Minute)
	_ = c.Set(ctx, "b", []byte("2"), time.Minute)
	_ = c.Set(ctx, "c", []byte("3"), time.Minute)

	// "a" should be evicted (LRU)
	_, err := c.Get(ctx, "a")
	assert.True(t, errors.Is(err, ErrCacheMiss))

	got, err := c.Get(ctx, "b")
	require.NoError(t, err)
	assert.Equal(t, []byte("2"), got)

	got, err = c.Get(ctx, "c")
	require.NoError(t, err)
	assert.Equal(t, []byte("3"), got)
}

func TestMemory_EvictionRespectsAccess(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(2)

	defer func() { _ = c.Close() }()

	_ = c.Set(ctx, "a", []byte("1"), time.Minute)
	_ = c.Set(ctx, "b", []byte("2"), time.Minute)

	// Access "a" to make it recently used.
	_, _ = c.Get(ctx, "a")

	// "b" is now LRU and should be evicted.
	_ = c.Set(ctx, "c", []byte("3"), time.Minute)

	_, err := c.Get(ctx, "b")
	assert.True(t, errors.Is(err, ErrCacheMiss))

	got, err := c.Get(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, []byte("1"), got)
}

func TestMemory_Delete(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

	_ = c.Set(ctx, "del", []byte("me"), time.Minute)

	err := c.Delete(ctx, "del")
	require.NoError(t, err)

	_, err = c.Get(ctx, "del")
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestMemory_DeleteNonexistent(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

	err := c.Delete(ctx, "nope")
	require.NoError(t, err)
}

func TestMemory_Overwrite(t *testing.T) {
	ctx := context.Background()
	c := NewMemory(100)

	defer func() { _ = c.Close() }()

	_ = c.Set(ctx, "k", []byte("old"), time.Minute)
	_ = c.Set(ctx, "k", []byte("new"), time.Minute)

	got, err := c.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, []byte("new"), got)
}
