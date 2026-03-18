package mq

import (
	"context"

	"github.com/psyb0t/ctxerrors"
)

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
