package mq

import "context"

// Handler processes messages from a topic.
// Return nil to ack, error to skip ack (redelivery).
type Handler interface {
	Handle(ctx context.Context, msg *Message) error
}

type HandlerFunc func(ctx context.Context, msg *Message) error

func (f HandlerFunc) Handle(
	ctx context.Context,
	msg *Message,
) error {
	return f(ctx, msg)
}
