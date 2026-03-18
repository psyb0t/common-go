package mq

import (
	"context"

	"github.com/psyb0t/ctxerrors"
)

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

type Publisher interface {
	Publish(ctx context.Context, topic string, v any) error
}

// Client is a message queue client. Subscribe blocks until ctx is cancelled
// or an error occurs.
type Client interface {
	Publisher
	Subscribe(ctx context.Context, topic string, h Handler) error
	Close() error
}

// TopicPublisher publishes to a fixed topic.
type TopicPublisher struct {
	publisher Publisher
	topic     string
}

func NewTopicPublisher(p Publisher, topic string) *TopicPublisher {
	return &TopicPublisher{publisher: p, topic: topic}
}

func (tp *TopicPublisher) Publish(ctx context.Context, v any) error {
	if err := tp.publisher.Publish(ctx, tp.topic, v); err != nil {
		return ctxerrors.Wrap(err, "topic publisher publish")
	}

	return nil
}

// TopicSubscriber subscribes to a fixed topic.
type TopicSubscriber struct {
	client Client
	topic  string
}

func NewTopicSubscriber(c Client, topic string) *TopicSubscriber {
	return &TopicSubscriber{client: c, topic: topic}
}

func (ts *TopicSubscriber) Subscribe(ctx context.Context, h Handler) error {
	if err := ts.client.Subscribe(ctx, ts.topic, h); err != nil {
		return ctxerrors.Wrap(err, "topic subscriber subscribe")
	}

	return nil
}
