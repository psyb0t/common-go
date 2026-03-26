package heartbeat

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeenWithin(t *testing.T) {
	t.Parallel()

	require.NoError(t, os.MkdirAll(dir, dirPerms))

	tests := []struct {
		name   string
		setup  func(t *testing.T) string
		within time.Duration
		want   bool
	}{
		{
			name: "no file returns false",
			setup: func(_ *testing.T) string {
				return "nonexistent-svc"
			},
			within: time.Minute,
			want:   false,
		},
		{
			name: "fresh file returns true",
			setup: func(t *testing.T) string {
				t.Helper()

				svc := "test-fresh"
				p := filepath.Join(dir, svc)
				touch(p)

				t.Cleanup(func() { _ = os.Remove(p) })

				return svc
			},
			within: time.Minute,
			want:   true,
		},
		{
			name: "stale file returns false",
			setup: func(t *testing.T) string {
				t.Helper()

				svc := "test-stale"
				p := filepath.Join(dir, svc)
				touch(p)

				stale := time.Now().Add(-5 * time.Minute)

				require.NoError(t,
					os.Chtimes(p, stale, stale),
				)

				t.Cleanup(func() { _ = os.Remove(p) })

				return svc
			},
			within: time.Minute,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := tt.setup(t)

			assert.Equal(t, tt.want,
				SeenWithin(svc, tt.within),
			)
		})
	}
}

func TestStartTouchesFile(t *testing.T) {
	t.Parallel()

	name := "test-start-" + t.Name()
	path := filepath.Join(dir, name)

	t.Cleanup(func() { _ = os.Remove(path) })

	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	done := make(chan struct{})

	go func() {
		Start(ctx, name)
		close(done)
	}()

	// Give Start a moment to touch the file.
	time.Sleep(100 * time.Millisecond)

	assert.True(t, SeenWithin(name, time.Minute))

	cancel()
	<-done
}
