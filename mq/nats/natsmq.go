package natsmq

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/psyb0t/common-go/mq"
	"github.com/psyb0t/ctxerrors"
)

// Client implements mq.Client over NATS JetStream.
type Client struct {
	conn  *nats.Conn
	js    nats.JetStreamContext
	group string
}

// New creates a NATS JetStream client. If group is
// empty, Subscribe uses plain fan-out (every
// subscriber gets every message). If group is set,
// Subscribe uses queue groups (each message goes to
// one consumer in the group).
func New(
	url, group string,
	streams ...StreamConfig,
) (*Client, error) {
	if url == "" {
		return nil, ErrEmptyURL
	}

	conn, err := nats.Connect(url)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "nats connect")
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()

		return nil, ctxerrors.Wrap(
			err, "jetstream context",
		)
	}

	c := &Client{conn: conn, js: js, group: group}

	for _, s := range streams {
		if err := c.ensureStream(s); err != nil {
			conn.Close()

			return nil, ctxerrors.Wrap(
				err, "ensure stream "+s.Name,
			)
		}
	}

	return c, nil
}

// Close drains and closes the NATS connection.
func (c *Client) Close() error {
	c.conn.Close()

	return nil
}

// Publish marshals v as JSON and publishes to the
// given topic via JetStream.
func (c *Client) Publish(
	ctx context.Context,
	topic string,
	v any,
) error {
	data, err := json.Marshal(v)
	if err != nil {
		return ctxerrors.Wrap(
			err, "marshal publish data",
		)
	}

	_, err = c.js.Publish(
		topic, data, nats.Context(ctx),
	)
	if err != nil {
		return ctxerrors.Wrap(err, "jetstream publish")
	}

	return nil
}

// Subscribe blocks, delivering messages on the topic
// to the handler until ctx is cancelled. If the
// client was created with a queue group, messages are
// load-balanced across group members. Otherwise every
// subscriber gets every message (fan-out).
func (c *Client) Subscribe(
	ctx context.Context,
	topic string,
	h mq.Handler,
) error {
	sub, err := c.subscribe(topic)
	if err != nil {
		return ctxerrors.Wrap(
			err, "jetstream subscribe",
		)
	}

	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			slog.Error("unsubscribe failed",
				"error", err,
			)
		}
	}()

	return c.consumeLoop(ctx, sub, h)
}

func (c *Client) subscribe(
	topic string,
) (*nats.Subscription, error) {
	opts := []nats.SubOpt{
		nats.DeliverAll(),
		nats.AckExplicit(),
	}

	if c.group != "" {
		sub, err := c.js.QueueSubscribeSync(
			topic, c.group, opts...,
		)
		if err != nil {
			return nil, ctxerrors.Wrap(
				err, "queue subscribe",
			)
		}

		return sub, nil
	}

	sub, err := c.js.SubscribeSync(topic, opts...)
	if err != nil {
		return nil, ctxerrors.Wrap(
			err, "subscribe sync",
		)
	}

	return sub, nil
}

func (c *Client) consumeLoop(
	ctx context.Context,
	sub *nats.Subscription,
	h mq.Handler,
) error {
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) ||
				errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil
			}

			return ctxerrors.Wrap(err, "next message")
		}

		m := mq.NewMessage(msg.Data, func() error {
			return msg.Ack()
		})

		if err := h.Handle(ctx, m); err != nil {
			continue
		}

		if err := m.Ack(); err != nil {
			slog.Error("ack failed", "error", err)
		}
	}
}

const defaultMaxAge = 24 * time.Hour

func (c *Client) ensureStream(
	config StreamConfig,
) error {
	if config.MaxAge == 0 {
		config.MaxAge = defaultMaxAge
	}

	_, err := c.js.StreamInfo(config.Name)
	if err == nil {
		return nil
	}

	if !errors.Is(err, nats.ErrStreamNotFound) {
		return ctxerrors.Wrap(err, "get stream info")
	}

	storage := nats.MemoryStorage
	if config.Storage == FileStorage {
		storage = nats.FileStorage
	}

	retention := retentionToNATS(config.Retention)

	_, err = c.js.AddStream(&nats.StreamConfig{
		Name:      config.Name,
		Subjects:  []string{config.SubjectBase + ".>"},
		Storage:   storage,
		Retention: retention,
		MaxAge:    config.MaxAge,
	})
	if err != nil {
		return ctxerrors.Wrap(err, "create stream")
	}

	return nil
}

func retentionToNATS(
	r RetentionPolicy,
) nats.RetentionPolicy {
	switch r {
	case InterestRetention:
		return nats.InterestPolicy
	case WorkQueueRetention:
		return nats.WorkQueuePolicy
	case LimitsRetention:
		return nats.LimitsPolicy
	}

	return nats.InterestPolicy
}
