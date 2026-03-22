package natsmq

import "errors"

// ErrEmptyURL is returned when New is called with an
// empty NATS URL.
var ErrEmptyURL = errors.New("empty nats url")
