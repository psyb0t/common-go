//nolint:tagliatelle // snake_case to match SSE format
package sse

import "github.com/psyb0t/common-go/llm"

type EventType string

const (
	EventTypeMessageStart      EventType = "message_start"
	EventTypeContentBlockStart EventType = "content_block_start"
	EventTypePing              EventType = "ping"
	EventTypeContentBlockDelta EventType = "content_block_delta"
	EventTypeContentBlockStop  EventType = "content_block_stop"
	EventTypeMessageDelta      EventType = "message_delta"
	EventTypeMessageStop       EventType = "message_stop"
)

type ContentBlockType string

const (
	ContentBlockTypeMessage     ContentBlockType = "message"
	ContentBlockTypeText        ContentBlockType = "text"
	ContentBlockTypeTextDelta   ContentBlockType = "text_delta"
	ContentBlockTypeToolUse     ContentBlockType = "tool_use"
	ContentBlockTypeToolResult  ContentBlockType = "tool_result"
	ContentBlockTypeInputJSON   ContentBlockType = "input_json_delta"
	ContentBlockTypeJSONPartial ContentBlockType = "json_partial"
)

type StopReason string

const (
	StopReasonEndTurn StopReason = "end_turn"
)

type Event struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type MessageStartData struct {
	Type    EventType   `json:"type"`
	Message MessageMeta `json:"message"`
}

type MessageMeta struct {
	ID             string           `json:"id"`
	ConversationID string           `json:"conversation_id"`
	Type           ContentBlockType `json:"type"`
	Role           llm.Role         `json:"role"`
	Content        []any            `json:"content"`
	Model          string           `json:"model"`
	StopReason     *StopReason      `json:"stop_reason"`
	StopSequence   *string          `json:"stop_sequence"`
	Usage          UsageStart       `json:"usage"`
}

type UsageStart struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ContentBlockStartData struct {
	Type         EventType    `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ContentBlock `json:"content_block"`
}

type ContentBlock struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text,omitempty"`
}

type PingData struct {
	Type EventType `json:"type"`
}

type ContentBlockDeltaData struct {
	Type  EventType `json:"type"`
	Index int       `json:"index"`
	Delta TextDelta `json:"delta"`
}

type TextDelta struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text"`
}

type ContentBlockStopData struct {
	Type  EventType `json:"type"`
	Index int       `json:"index"`
}

type MessageDeltaData struct {
	Type  EventType        `json:"type"`
	Delta MessageDeltaInfo `json:"delta"`
	Usage UsageEnd         `json:"usage"`
}

type MessageDeltaInfo struct {
	StopReason   StopReason `json:"stop_reason"`
	StopSequence *string    `json:"stop_sequence"`
}

type UsageEnd struct {
	OutputTokens int `json:"output_tokens"`
}

type MessageStopData struct {
	Type EventType `json:"type"`
}

type ToolUseBlock struct {
	Type  ContentBlockType `json:"type"`
	ID    string           `json:"id"`
	Name  string           `json:"name"`
	Input any              `json:"input"`
}

type ToolResultBlock struct {
	Type      ContentBlockType `json:"type"`
	ToolUseID string           `json:"tool_use_id"`
	Content   string           `json:"content,omitempty"`
}

type ContentBlockStartToolUseData struct {
	Type         EventType    `json:"type"`
	Index        int          `json:"index"`
	ContentBlock ToolUseBlock `json:"content_block"`
}

type ContentBlockStartToolResultData struct {
	Type         EventType       `json:"type"`
	Index        int             `json:"index"`
	ContentBlock ToolResultBlock `json:"content_block"`
}

type InputJSONDelta struct {
	Type        ContentBlockType `json:"type"`
	PartialJSON string           `json:"partial_json"`
}

type ContentBlockDeltaToolInputData struct {
	Type  EventType      `json:"type"`
	Index int            `json:"index"`
	Delta InputJSONDelta `json:"delta"`
}

type ToolResultDelta struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text"`
}

type ContentBlockDeltaToolResultData struct {
	Type  EventType       `json:"type"`
	Index int             `json:"index"`
	Delta ToolResultDelta `json:"delta"`
}
