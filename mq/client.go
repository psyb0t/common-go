package mq

import "context"

// Client is a message queue client. Subscribe blocks until ctx is cancelled
// or an error occurs.
type Client interface {
	Publisher
	Subscribe(ctx context.Context, topic string, h Handler) error
	Close() error
}
