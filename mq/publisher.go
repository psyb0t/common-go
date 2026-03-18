package mq

import (
	"context"

	"github.com/psyb0t/ctxerrors"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, v any) error
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
