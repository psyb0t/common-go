package natsmq

import (
	"strings"
	"time"
)

// StorageType selects NATS JetStream storage backend.
type StorageType int

const (
	MemoryStorage StorageType = iota
	FileStorage
)

// RetentionPolicy selects how the stream retains
// messages.
type RetentionPolicy int

const (
	// InterestRetention keeps messages as long as
	// there are active consumers interested.
	InterestRetention RetentionPolicy = iota

	// WorkQueueRetention delivers each message to
	// exactly one consumer in the queue group.
	WorkQueueRetention

	// LimitsRetention keeps messages up to the
	// configured limits (default NATS behavior).
	LimitsRetention
)

// StreamConfig describes a JetStream stream to
// ensure on client creation.
type StreamConfig struct {
	Name        string
	SubjectBase string
	MaxAge      time.Duration
	Storage     StorageType
	Retention   RetentionPolicy
}

// BuildSubject constructs a dot-separated subject
// under this stream's base.
func (c StreamConfig) BuildSubject(
	parts ...string,
) string {
	return c.SubjectBase + "." + strings.Join(parts, ".")
}
