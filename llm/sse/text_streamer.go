package sse

import (
	"strings"

	"github.com/psyb0t/ctxerrors"
)

// TextStreamer accumulates text output and optionally
// publishes content blocks to an SSE stream via Publisher.
// Publisher may be nil for non-streaming use.
type TextStreamer struct {
	publisher      *Publisher
	messageBuilder strings.Builder
	BlockIndex     int
	blockStarted   bool
}

func NewTextStreamer(
	publisher *Publisher,
	startIndex int,
) *TextStreamer {
	return &TextStreamer{
		publisher:  publisher,
		BlockIndex: startIndex,
	}
}

// Text returns the accumulated message text.
func (s *TextStreamer) Text() string {
	return s.messageBuilder.String()
}

// BlockStarted reports whether a text block is currently open.
func (s *TextStreamer) BlockStarted() bool {
	return s.blockStarted
}

// WriteChunk appends text and publishes a content block delta
// if a publisher is set. Automatically starts a new text block
// on first chunk.
func (s *TextStreamer) WriteChunk(text string) error {
	if text == "" {
		return nil
	}

	s.messageBuilder.WriteString(text)

	if s.publisher == nil {
		return nil
	}

	if !s.blockStarted {
		err := s.publisher.SendContentBlockStartText(
			s.BlockIndex,
		)
		if err != nil {
			return ctxerrors.Wrap(
				err, "send content block start",
			)
		}

		s.blockStarted = true
	}

	err := s.publisher.SendContentBlockDeltaText(
		s.BlockIndex, text,
	)
	if err != nil {
		return ctxerrors.Wrap(
			err, "send content block delta",
		)
	}

	return nil
}

// CloseBlock sends a content block stop if a text block
// is currently open. Call this after text streaming ends.
func (s *TextStreamer) CloseBlock() error {
	if !s.blockStarted {
		return nil
	}

	if s.publisher == nil {
		s.blockStarted = false
		s.BlockIndex++

		return nil
	}

	err := s.publisher.SendContentBlockStop(s.BlockIndex)
	if err != nil {
		return ctxerrors.Wrap(err, "send content block stop")
	}

	s.blockStarted = false
	s.BlockIndex++

	return nil
}
