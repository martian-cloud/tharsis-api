package agent

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/m-mizutani/gollem"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

var thinkingRe = regexp.MustCompile(`(?s)<thinking>(.*?)</thinking>`)

// extractThinking pulls <thinking>...</thinking> content out of text,
// returning the thinking content and the remaining text separately.
func extractThinking(text string) (thinking, remaining string) {
	matches := thinkingRe.FindAllStringSubmatch(text, -1)
	for _, m := range matches {
		if len(m) > 1 {
			thinking += m[1]
		}
	}
	remaining = thinkingRe.ReplaceAllString(text, "")
	return thinking, remaining
}

// EventType identifies the kind of AG-UI event.
type EventType string

const (
	// EventTypeRunStarted is the event type for a run starting.
	EventTypeRunStarted EventType = "RUN_STARTED"
	// EventTypeRunFinished is the event type for a run finishing.
	EventTypeRunFinished EventType = "RUN_FINISHED"
	// EventTypeRunError is the event type for a run error.
	EventTypeRunError EventType = "RUN_ERROR"
	// EventTypeRunCancelled is the event type for a run being cancelled.
	EventTypeRunCancelled EventType = "RUN_CANCELLED"
	// EventTypeStepStarted is the event type for a step starting.
	EventTypeStepStarted EventType = "STEP_STARTED"
	// EventTypeStepFinished is the event type for a step finishing.
	EventTypeStepFinished EventType = "STEP_FINISHED"

	// EventTypeTextMessageStart is the event type for the start of a text message.
	EventTypeTextMessageStart EventType = "TEXT_MESSAGE_START"
	// EventTypeTextMessageContent is the event type for text message content.
	EventTypeTextMessageContent EventType = "TEXT_MESSAGE_CONTENT"
	// EventTypeTextMessageEnd is the event type for the end of a text message.
	EventTypeTextMessageEnd EventType = "TEXT_MESSAGE_END"
	// EventTypeTextMessageChunk is the event type for a text message chunk.
	EventTypeTextMessageChunk EventType = "TEXT_MESSAGE_CHUNK"

	// EventTypeToolCallStart is the event type for the start of a tool call.
	EventTypeToolCallStart EventType = "TOOL_CALL_START"
	// EventTypeToolCallArgs is the event type for tool call arguments.
	EventTypeToolCallArgs EventType = "TOOL_CALL_ARGS"
	// EventTypeToolCallEnd is the event type for the end of a tool call.
	EventTypeToolCallEnd EventType = "TOOL_CALL_END"
	// EventTypeToolCallResult is the event type for a tool call result.
	EventTypeToolCallResult EventType = "TOOL_CALL_RESULT"
	// EventTypeToolCallChunk is the event type for a tool call chunk.
	EventTypeToolCallChunk EventType = "TOOL_CALL_CHUNK"

	// EventTypeStateSnapshot is the event type for a state snapshot.
	EventTypeStateSnapshot EventType = "STATE_SNAPSHOT"
	// EventTypeStateDelta is the event type for a state delta.
	EventTypeStateDelta EventType = "STATE_DELTA"
	// EventTypeMessagesSnapshot is the event type for a messages snapshot.
	EventTypeMessagesSnapshot EventType = "MESSAGES_SNAPSHOT"

	// EventTypeActivitySnapshot is the event type for an activity snapshot.
	EventTypeActivitySnapshot EventType = "ACTIVITY_SNAPSHOT"
	// EventTypeActivityDelta is the event type for an activity delta.
	EventTypeActivityDelta EventType = "ACTIVITY_DELTA"

	// EventTypeRaw is the event type for a raw event.
	EventTypeRaw EventType = "RAW"
	// EventTypeCustom is the event type for a custom event.
	EventTypeCustom EventType = "CUSTOM"

	// EventTypeReasoningStart is the event type for the start of reasoning.
	EventTypeReasoningStart EventType = "REASONING_START"
	// EventTypeReasoningMessageStart is the event type for the start of a reasoning message.
	EventTypeReasoningMessageStart EventType = "REASONING_MESSAGE_START"
	// EventTypeReasoningMessageContent is the event type for reasoning message content.
	EventTypeReasoningMessageContent EventType = "REASONING_MESSAGE_CONTENT"
	// EventTypeReasoningMessageEnd is the event type for the end of a reasoning message.
	EventTypeReasoningMessageEnd EventType = "REASONING_MESSAGE_END"
	// EventTypeReasoningMessageChunk is the event type for a reasoning message chunk.
	EventTypeReasoningMessageChunk EventType = "REASONING_MESSAGE_CHUNK"
	// EventTypeReasoningEnd is the event type for the end of reasoning.
	EventTypeReasoningEnd EventType = "REASONING_END"
	// EventTypeReasoningEncryptedValue is the event type for an encrypted reasoning value.
	EventTypeReasoningEncryptedValue EventType = "REASONING_ENCRYPTED_VALUE"
)

// Event is the interface implemented by all AG-UI events.
type Event interface {
	EventType() EventType
}

// BaseEvent contains properties shared by all events.
type BaseEvent struct {
	Type      EventType `json:"type"`
	Timestamp *int64    `json:"timestamp,omitempty"`
	RawEvent  any       `json:"rawEvent,omitempty"`
}

// EventType returns the type of the event.
func (e BaseEvent) EventType() EventType { return e.Type }

// MarshalEvent serializes any Event to JSON, injecting the type field.
func MarshalEvent(ev Event) ([]byte, error) {
	return json.Marshal(ev)
}

// RunStartedEvent is emitted when a run starts.
type RunStartedEvent struct {
	BaseEvent
	ThreadID    string `json:"threadId"`
	RunID       string `json:"runId"`
	ParentRunID string `json:"parentRunId,omitempty"`
}

// RunFinishedEvent is emitted when a run finishes.
type RunFinishedEvent struct {
	BaseEvent
	ThreadID string `json:"threadId"`
	RunID    string `json:"runId"`
}

// RunErrorEvent is emitted when a run encounters an error.
type RunErrorEvent struct {
	BaseEvent
	ThreadID string `json:"threadId,omitempty"`
	RunID    string `json:"runId,omitempty"`
	Message  string `json:"message"`
}

// StepStartedEvent is emitted when a step starts.
type StepStartedEvent struct {
	BaseEvent
	StepName string `json:"stepName"`
}

// StepFinishedEvent is emitted when a step finishes.
type StepFinishedEvent struct {
	BaseEvent
	StepName string `json:"stepName"`
}

// TextMessageStartEvent is emitted at the start of a text message.
type TextMessageStartEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
}

// TextMessageContentEvent is emitted for text message content.
type TextMessageContentEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
	Delta     string `json:"delta"`
}

// TextMessageEndEvent is emitted at the end of a text message.
type TextMessageEndEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
}

// TextMessageChunkEvent is emitted for a text message chunk.
type TextMessageChunkEvent struct {
	BaseEvent
	MessageID string `json:"messageId,omitempty"`
	Role      string `json:"role,omitempty"`
	Delta     string `json:"delta,omitempty"`
}

// ToolCallStartEvent is emitted at the start of a tool call.
type ToolCallStartEvent struct {
	BaseEvent
	ToolCallID      string `json:"toolCallId"`
	ToolCallName    string `json:"toolCallName"`
	ParentMessageID string `json:"parentMessageId,omitempty"`
}

// ToolCallArgsEvent is emitted for tool call arguments.
type ToolCallArgsEvent struct {
	BaseEvent
	ToolCallID string `json:"toolCallId"`
	Delta      string `json:"delta"`
}

// ToolCallEndEvent is emitted at the end of a tool call.
type ToolCallEndEvent struct {
	BaseEvent
	ToolCallID string `json:"toolCallId"`
}

// ToolCallResultEvent is emitted for a tool call result.
type ToolCallResultEvent struct {
	BaseEvent
	MessageID  string `json:"messageId"`
	ToolCallID string `json:"toolCallId"`
	Content    string `json:"content"`
	Role       string `json:"role,omitempty"`
}

// ToolCallChunkEvent is emitted for a tool call chunk.
type ToolCallChunkEvent struct {
	BaseEvent
	ToolCallID      string `json:"toolCallId,omitempty"`
	ToolCallName    string `json:"toolCallName,omitempty"`
	ParentMessageID string `json:"parentMessageId,omitempty"`
	Delta           string `json:"delta,omitempty"`
}

// StateSnapshotEvent is emitted for a state snapshot.
type StateSnapshotEvent struct {
	BaseEvent
	Snapshot any `json:"snapshot"`
}

// StateDeltaEvent is emitted for a state delta.
type StateDeltaEvent struct {
	BaseEvent
	Delta []any `json:"delta"`
}

// MessagesSnapshotEvent is emitted for a messages snapshot.
type MessagesSnapshotEvent struct {
	BaseEvent
	Messages []any `json:"messages"`
}

// ActivitySnapshotEvent is emitted for an activity snapshot.
type ActivitySnapshotEvent struct {
	BaseEvent
	MessageID    string `json:"messageId"`
	ActivityType string `json:"activityType"`
	Content      any    `json:"content"`
	Replace      *bool  `json:"replace,omitempty"`
}

// ActivityDeltaEvent is emitted for an activity delta.
type ActivityDeltaEvent struct {
	BaseEvent
	MessageID    string `json:"messageId"`
	ActivityType string `json:"activityType"`
	Patch        []any  `json:"patch"`
}

// RawEvent is emitted for a raw event.
type RawEvent struct {
	BaseEvent
	EventData any    `json:"event"`
	Source    string `json:"source,omitempty"`
}

// CustomEvent is emitted for a custom event.
type CustomEvent struct {
	BaseEvent
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// ReasoningStartEvent is emitted at the start of reasoning.
type ReasoningStartEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
}

// ReasoningMessageStartEvent is emitted at the start of a reasoning message.
type ReasoningMessageStartEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
	Role      string `json:"role"`
}

// ReasoningMessageContentEvent is emitted for reasoning message content.
type ReasoningMessageContentEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
	Delta     string `json:"delta"`
}

// ReasoningMessageEndEvent is emitted at the end of a reasoning message.
type ReasoningMessageEndEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
}

// ReasoningMessageChunkEvent is emitted for a reasoning message chunk.
type ReasoningMessageChunkEvent struct {
	BaseEvent
	MessageID string `json:"messageId,omitempty"`
	Delta     string `json:"delta,omitempty"`
}

// ReasoningEndEvent is emitted at the end of reasoning.
type ReasoningEndEvent struct {
	BaseEvent
	MessageID string `json:"messageId"`
}

// ReasoningEncryptedValueEvent is emitted for an encrypted reasoning value.
type ReasoningEncryptedValueEvent struct {
	BaseEvent
	Subtype        string `json:"subtype"`
	EntityID       string `json:"entityId"`
	EncryptedValue string `json:"encryptedValue"`
}

// EventsFromMessage converts an AgentSessionMessage into a list of AG-UI events.
// For tool role messages, content is loaded from object storage via the provided loader.
func EventsFromMessage(msg *models.AgentSessionMessage, loadToolContent func(messageID string) (json.RawMessage, error), reasoning bool) ([]Event, error) {
	msgID := msg.Metadata.ID

	var ts *int64
	if msg.Metadata.CreationTimestamp != nil {
		v := msg.Metadata.CreationTimestamp.UnixMilli()
		ts = &v
	}
	base := func(t EventType) BaseEvent {
		return BaseEvent{Type: t, Timestamp: ts}
	}

	content := msg.Content
	if msg.Role == string(gollem.RoleTool) {
		loaded, err := loadToolContent(msg.Metadata.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load tool content for message %s: %w", msgID, err)
		}
		content = loaded
	}

	var contents []gollem.MessageContent
	if err := json.Unmarshal(content, &contents); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message content for message %s: %w", msgID, err)
	}

	var events []Event

	switch gollem.MessageRole(msg.Role) {
	case gollem.RoleUser:
		for _, c := range contents {
			if c.Type != gollem.MessageContentTypeText {
				continue
			}
			tc, err := c.GetTextContent()
			if err != nil {
				return nil, fmt.Errorf("failed to get text content for message %s: %w", msgID, err)
			}
			events = append(events,
				&TextMessageStartEvent{BaseEvent: base(EventTypeTextMessageStart), MessageID: msgID, Role: "user"},
				&TextMessageContentEvent{BaseEvent: base(EventTypeTextMessageContent), MessageID: msgID, Delta: tc.Text},
				&TextMessageEndEvent{BaseEvent: base(EventTypeTextMessageEnd), MessageID: msgID},
			)
		}
	case gollem.RoleAssistant:
		for _, c := range contents {
			switch c.Type {
			case gollem.MessageContentTypeText:
				tc, err := c.GetTextContent()
				if err != nil {
					return nil, fmt.Errorf("failed to get text content for message %s: %w", msgID, err)
				}
				thinkingText, cleaned := extractThinking(tc.Text)
				if reasoning && thinkingText != "" {
					events = append(events,
						&ReasoningStartEvent{BaseEvent: base(EventTypeReasoningStart), MessageID: msgID},
						&ReasoningMessageStartEvent{BaseEvent: base(EventTypeReasoningMessageStart), MessageID: msgID, Role: "reasoning"},
						&ReasoningMessageContentEvent{BaseEvent: base(EventTypeReasoningMessageContent), MessageID: msgID, Delta: thinkingText},
						&ReasoningMessageEndEvent{BaseEvent: base(EventTypeReasoningMessageEnd), MessageID: msgID},
						&ReasoningEndEvent{BaseEvent: base(EventTypeReasoningEnd), MessageID: msgID},
					)
				}
				if cleaned != "" {
					events = append(events,
						&TextMessageStartEvent{BaseEvent: base(EventTypeTextMessageStart), MessageID: msgID, Role: "assistant"},
						&TextMessageContentEvent{BaseEvent: base(EventTypeTextMessageContent), MessageID: msgID, Delta: cleaned},
						&TextMessageEndEvent{BaseEvent: base(EventTypeTextMessageEnd), MessageID: msgID},
					)
				}
			case gollem.MessageContentTypeToolCall:
				fc, err := c.GetToolCallContent()
				if err != nil {
					return nil, fmt.Errorf("failed to get tool call content for message %s: %w", msgID, err)
				}
				argsJSON, _ := json.Marshal(fc.Arguments)
				events = append(events, &ToolCallStartEvent{BaseEvent: base(EventTypeToolCallStart), ToolCallID: fc.ID, ToolCallName: fc.Name, ParentMessageID: msgID})
				args := string(argsJSON)
				for i := 0; i < len(args); i += 1024 {
					end := i + 1024
					if end > len(args) {
						end = len(args)
					}
					events = append(events, &ToolCallArgsEvent{BaseEvent: base(EventTypeToolCallArgs), ToolCallID: fc.ID, Delta: args[i:end]})
				}
				events = append(events, &ToolCallEndEvent{BaseEvent: base(EventTypeToolCallEnd), ToolCallID: fc.ID})
			}
		}
	case gollem.RoleTool:
		for _, c := range contents {
			if c.Type != gollem.MessageContentTypeToolResponse {
				continue
			}
			tr, err := c.GetToolResponseContent()
			if err != nil {
				return nil, fmt.Errorf("failed to get tool response content for message %s: %w", msgID, err)
			}
			resultJSON, _ := json.Marshal(tr.Response)
			events = append(events, &ToolCallResultEvent{
				BaseEvent:  base(EventTypeToolCallResult),
				ToolCallID: tr.ToolCallID,
				Content:    string(resultJSON),
				Role:       "tool",
			})
		}
	}

	return events, nil
}
