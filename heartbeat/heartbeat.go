package heartbeat

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	dir      = "/tmp/heartbeat"
	dirPerms = 0o750
	interval = 1 * time.Minute
)

// Start touches /tmp/heartbeat/<name> every minute
// until the context is cancelled.
func Start(
	ctx context.Context,
	name string,
) {
	if err := os.MkdirAll(dir, dirPerms); err != nil {
		slog.Error("heartbeat mkdir failed",
			"error", err,
		)

		return
	}

	path := filepath.Join(dir, name)

	touch(path)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			touch(path)
		}
	}
}

// SeenWithin returns true if the heartbeat file for
// the given service was modified within the specified
// duration.
func SeenWithin(name string, d time.Duration) bool {
	path := filepath.Join(dir, name)

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) <= d
}

func touch(path string) {
	f, err := os.Create(path)
	if err != nil {
		slog.Error("heartbeat touch failed",
			"path", path,
			"error", err,
		)

		return
	}

	if err := f.Close(); err != nil {
		slog.Error("heartbeat close failed",
			"path", path,
			"error", err,
		)
	}
}
