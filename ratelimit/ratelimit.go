package ratelimit

import (
	"context"
	"log/slog"
	"sync"

	"github.com/psyb0t/ctxerrors"
	"golang.org/x/time/rate"
)

// Limiter is a named rate limiter registry. Register
// limiters by name, then Wait on them by name.
type Limiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// New creates an empty Limiter registry.
func New() *Limiter {
	return &Limiter{
		limiters: make(map[string]*rate.Limiter),
	}
}

// Register adds a rate limiter with the given name,
// requests per second, and burst size.
func (l *Limiter) Register(
	name string,
	rps float64,
	burst int,
) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.limiters[name] = rate.NewLimiter(
		rate.Limit(rps), burst,
	)
	slog.Debug("rate limiter registered",
		"name", name,
		"rps", rps,
		"burst", burst,
	)
}

// Wait blocks until the named rate limiter allows an
// event or the context is cancelled.
func (l *Limiter) Wait(
	ctx context.Context,
	name string,
) error {
	l.mu.RLock()
	limiter, ok := l.limiters[name]
	l.mu.RUnlock()

	if !ok {
		return ctxerrors.New("rate limiter not found: " + name)
	}

	slog.Debug("rate limiter waiting", "name", name)

	if err := limiter.Wait(ctx); err != nil {
		return ctxerrors.Wrap(err, "rate limit wait: "+name)
	}

	return nil
}
