package sse

import (
	"context"
	"encoding/json"

	"github.com/psyb0t/common-go/llm"
	"github.com/psyb0t/common-go/mq"
	"github.com/psyb0t/ctxerrors"
)

//nolint:containedctx // short-lived per-request publisher
type Publisher struct {
	ctx   context.Context
	mq    mq.Publisher
	topic string
}

func NewPublisher(
	ctx context.Context,
	mqPublisher mq.Publisher,
	topic string,
) *Publisher {
	return &Publisher{
		ctx:   ctx,
		mq:    mqPublisher,
		topic: topic,
	}
}

func (p *Publisher) Publish(
	eventType EventType,
	data any,
) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ctxerrors.Wrap(err, "marshal SSE data")
	}

	event := Event{
		Event: string(eventType),
		Data:  string(jsonData),
	}

	if err = p.mq.Publish(p.ctx, p.topic, event); err != nil {
		return ctxerrors.Wrap(err, "publish event")
	}

	return nil
}

func (p *Publisher) SendMessageStart(
	msgID, conversationID, model string,
) error {
	return p.Publish(
		EventTypeMessageStart,
		MessageStartData{
			Type: EventTypeMessageStart,
			Message: MessageMeta{
				ID:             msgID,
				ConversationID: conversationID,
				Type:           ContentBlockTypeMessage,
				Role:           llm.RoleAssistant,
				Content:        []any{},
				Model:          model,
				StopReason:     nil,
				StopSequence:   nil,
				Usage: UsageStart{
					InputTokens:  0,
					OutputTokens: 0,
				},
			},
		},
	)
}

func (p *Publisher) SendPing() error {
	return p.Publish(
		EventTypePing,
		PingData{
			Type: EventTypePing,
		},
	)
}

func (p *Publisher) SendContentBlockStartText(index int) error {
	return p.Publish(
		EventTypeContentBlockStart,
		ContentBlockStartData{
			Type:  EventTypeContentBlockStart,
			Index: index,
			ContentBlock: ContentBlock{
				Type: ContentBlockTypeText,
				Text: "",
			},
		},
	)
}

func (p *Publisher) SendContentBlockDeltaText(
	index int,
	text string,
) error {
	return p.Publish(
		EventTypeContentBlockDelta,
		ContentBlockDeltaData{
			Type:  EventTypeContentBlockDelta,
			Index: index,
			Delta: TextDelta{
				Type: ContentBlockTypeTextDelta,
				Text: text,
			},
		},
	)
}

func (p *Publisher) SendContentBlockStop(index int) error {
	return p.Publish(
		EventTypeContentBlockStop,
		ContentBlockStopData{
			Type:  EventTypeContentBlockStop,
			Index: index,
		},
	)
}

func (p *Publisher) SendMessageDelta(
	stopReason StopReason,
	outputTokens int,
) error {
	return p.Publish(
		EventTypeMessageDelta,
		MessageDeltaData{
			Type: EventTypeMessageDelta,
			Delta: MessageDeltaInfo{
				StopReason:   stopReason,
				StopSequence: nil,
			},
			Usage: UsageEnd{
				OutputTokens: outputTokens,
			},
		},
	)
}

func (p *Publisher) SendMessageStop() error {
	return p.Publish(
		EventTypeMessageStop,
		MessageStopData{
			Type: EventTypeMessageStop,
		},
	)
}

func (p *Publisher) SendToolUseStart(
	index int,
	toolUseID, name string,
) error {
	return p.Publish(
		EventTypeContentBlockStart,
		ContentBlockStartToolUseData{
			Type:  EventTypeContentBlockStart,
			Index: index,
			ContentBlock: ToolUseBlock{
				Type:  ContentBlockTypeToolUse,
				ID:    toolUseID,
				Name:  name,
				Input: map[string]any{},
			},
		},
	)
}

func (p *Publisher) SendToolInputDelta(
	index int,
	inputJSON string,
) error {
	return p.Publish(
		EventTypeContentBlockDelta,
		ContentBlockDeltaToolInputData{
			Type:  EventTypeContentBlockDelta,
			Index: index,
			Delta: InputJSONDelta{
				Type:        ContentBlockTypeInputJSON,
				PartialJSON: inputJSON,
			},
		},
	)
}

func (p *Publisher) SendToolResultStart(
	index int,
	toolUseID string,
) error {
	return p.Publish(
		EventTypeContentBlockStart,
		ContentBlockStartToolResultData{
			Type:  EventTypeContentBlockStart,
			Index: index,
			ContentBlock: ToolResultBlock{
				Type:      ContentBlockTypeToolResult,
				ToolUseID: toolUseID,
			},
		},
	)
}

func (p *Publisher) SendToolResultDelta(
	index int,
	text string,
) error {
	return p.Publish(
		EventTypeContentBlockDelta,
		ContentBlockDeltaToolResultData{
			Type:  EventTypeContentBlockDelta,
			Index: index,
			Delta: ToolResultDelta{
				Type: ContentBlockTypeJSONPartial,
				Text: text,
			},
		},
	)
}

func (p *Publisher) SendStreamPreamble(
	msgID, conversationID, model string,
) error {
	if err := p.SendMessageStart(
		msgID,
		conversationID,
		model,
	); err != nil {
		return err
	}

	return p.SendPing()
}

func (p *Publisher) SendStreamEpilogue(
	outputTokens int,
) error {
	if err := p.SendMessageDelta(
		StopReasonEndTurn,
		outputTokens,
	); err != nil {
		return err
	}

	return p.SendMessageStop()
}

func (p *Publisher) SendToolUseBlock(
	index int,
	toolUseID, name, inputJSON string,
) error {
	if err := p.SendToolUseStart(
		index,
		toolUseID,
		name,
	); err != nil {
		return err
	}

	if err := p.SendToolInputDelta(
		index,
		inputJSON,
	); err != nil {
		return err
	}

	return p.SendContentBlockStop(index)
}

func (p *Publisher) SendToolResultBlock(
	index int,
	toolUseID, resultText string,
) error {
	if err := p.SendToolResultStart(
		index,
		toolUseID,
	); err != nil {
		return err
	}

	if err := p.SendToolResultDelta(
		index,
		resultText,
	); err != nil {
		return err
	}

	return p.SendContentBlockStop(index)
}
