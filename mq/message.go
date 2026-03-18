package mq

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
)

type Message struct {
	Data []byte
	ack  func() error
}

func NewMessage(data []byte, ack func() error) *Message {
	return &Message{Data: data, ack: ack}
}

func (m *Message) Ack() error {
	return m.ack()
}

func (m *Message) Decode(v any) error {
	if err := json.Unmarshal(m.Data, v); err != nil {
		return ctxerrors.Wrap(err, "decode message")
	}

	return nil
}
