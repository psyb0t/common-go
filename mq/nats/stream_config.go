package natsmq

import (
	"strings"
	"time"
)

type StorageType int

const (
	MemoryStorage StorageType = iota
	FileStorage
)

type StreamConfig struct {
	Name        string
	SubjectBase string
	MaxAge      time.Duration
	Storage     StorageType
}

func (c StreamConfig) BuildSubject(parts ...string) string {
	return c.SubjectBase + "." + strings.Join(parts, ".")
}
