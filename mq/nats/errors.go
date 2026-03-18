package natsmq

import "errors"

var (
	ErrEmptyURL   = errors.New("empty nats url")
	ErrEmptyGroup = errors.New("empty queue group")
)
