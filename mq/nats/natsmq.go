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

type Client struct {
	conn  *nats.Conn
	js    nats.JetStreamContext
	group string
}

func New(url, group string, streams ...StreamConfig) (*Client, error) {
	if url == "" {
		return nil, ErrEmptyURL
	}

	if group == "" {
		return nil, ErrEmptyGroup
	}

	conn, err := nats.Connect(url)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "nats connect")
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()

		return nil, ctxerrors.Wrap(err, "jetstream context")
	}

	c := &Client{conn: conn, js: js, group: group}

	for _, s := range streams {
		if err := c.ensureStream(s); err != nil {
			conn.Close()

			return nil, ctxerrors.Wrap(err, "ensure stream "+s.Name)
		}
	}

	return c, nil
}

func (b *Client) Close() error {
	b.conn.Close()

	return nil
}

func (b *Client) Publish(
	ctx context.Context,
	topic string,
	v any,
) error {
	data, err := json.Marshal(v)
	if err != nil {
		return ctxerrors.Wrap(err, "marshal publish data")
	}

	_, err = b.js.Publish(topic, data, nats.Context(ctx))
	if err != nil {
		return ctxerrors.Wrap(err, "jetstream publish")
	}

	return nil
}

func (b *Client) Subscribe(
	ctx context.Context,
	topic string,
	h mq.Handler,
) error {
	sub, err := b.js.QueueSubscribeSync(
		topic,
		b.group,
		nats.DeliverAll(),
		nats.AckExplicit(),
	)
	if err != nil {
		return ctxerrors.Wrap(err, "jetstream subscribe")
	}

	defer func() {
		if err := sub.Unsubscribe(); err != nil {
			slog.Error("unsubscribe failed", "error", err)
		}
	}()

	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if ctx.Err() != nil {
			return nil
		}

		if err != nil {
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

func (b *Client) ensureStream(config StreamConfig) error {
	if config.MaxAge == 0 {
		config.MaxAge = defaultMaxAge
	}

	_, err := b.js.StreamInfo(config.Name)
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

	_, err = b.js.AddStream(&nats.StreamConfig{
		Name:      config.Name,
		Subjects:  []string{config.SubjectBase + ".>"},
		Storage:   storage,
		Retention: nats.WorkQueuePolicy,
		MaxAge:    config.MaxAge,
	})
	if err != nil {
		return ctxerrors.Wrap(err, "create stream")
	}

	return nil
}
