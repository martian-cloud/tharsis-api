package agent

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestExtractThinking(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectThinking  string
		expectRemaining string
	}{
		{
			name:            "no thinking tags",
			input:           "just plain text",
			expectRemaining: "just plain text",
		},
		{
			name:            "single thinking block",
			input:           "before<thinking>inner thought</thinking>after",
			expectThinking:  "inner thought",
			expectRemaining: "beforeafter",
		},
		{
			name:            "multiple thinking blocks",
			input:           "<thinking>first</thinking>middle<thinking>second</thinking>end",
			expectThinking:  "firstsecond",
			expectRemaining: "middleend",
		},
		{
			name:            "multiline thinking",
			input:           "start<thinking>\nline1\nline2\n</thinking>end",
			expectThinking:  "\nline1\nline2\n",
			expectRemaining: "startend",
		},
		{
			name:            "empty thinking tags",
			input:           "before<thinking></thinking>after",
			expectRemaining: "beforeafter",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			thinking, remaining := extractThinking(test.input)
			assert.Equal(t, test.expectThinking, thinking)
			assert.Equal(t, test.expectRemaining, remaining)
		})
	}
}

func TestMarshalEvent(t *testing.T) {
	ev := &RunStartedEvent{
		BaseEvent: BaseEvent{Type: EventTypeRunStarted},
		ThreadID:  "thread-1",
		RunID:     "run-1",
	}

	data, err := MarshalEvent(ev)
	require.Nil(t, err)

	var m map[string]any
	require.Nil(t, json.Unmarshal(data, &m))
	assert.Equal(t, "RUN_STARTED", m["type"])
	assert.Equal(t, "thread-1", m["threadId"])
	assert.Equal(t, "run-1", m["runId"])
}

func TestEventsFromMessage_UserRole(t *testing.T) {
	tc, _ := gollem.NewTextContent("hello")
	content, _ := json.Marshal([]gollem.MessageContent{tc})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-1"},
		Role:     "user",
		Content:  content,
	}

	events, err := EventsFromMessage(msg, nil, false)
	require.Nil(t, err)
	require.Len(t, events, 3)

	assert.Equal(t, EventTypeTextMessageStart, events[0].EventType())
	start := events[0].(*TextMessageStartEvent)
	assert.Equal(t, "user", start.Role)
	assert.Equal(t, "msg-1", start.MessageID)

	assert.Equal(t, EventTypeTextMessageContent, events[1].EventType())
	contentEv := events[1].(*TextMessageContentEvent)
	assert.Equal(t, "hello", contentEv.Delta)

	assert.Equal(t, EventTypeTextMessageEnd, events[2].EventType())
}

func TestEventsFromMessage_AssistantText(t *testing.T) {
	tc, _ := gollem.NewTextContent("response text")
	content, _ := json.Marshal([]gollem.MessageContent{tc})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-2"},
		Role:     "assistant",
		Content:  content,
	}

	events, err := EventsFromMessage(msg, nil, false)
	require.Nil(t, err)
	require.Len(t, events, 3)

	start := events[0].(*TextMessageStartEvent)
	assert.Equal(t, "assistant", start.Role)

	contentEv := events[1].(*TextMessageContentEvent)
	assert.Equal(t, "response text", contentEv.Delta)
}

func TestEventsFromMessage_AssistantWithThinking(t *testing.T) {
	tc, _ := gollem.NewTextContent("<thinking>my thoughts</thinking>visible text")
	content, _ := json.Marshal([]gollem.MessageContent{tc})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-3"},
		Role:     "assistant",
		Content:  content,
	}

	// Without reasoning flag — thinking stripped, only text events
	events, err := EventsFromMessage(msg, nil, false)
	require.Nil(t, err)
	require.Len(t, events, 3)
	contentEv := events[1].(*TextMessageContentEvent)
	assert.Equal(t, "visible text", contentEv.Delta)

	// With reasoning flag — reasoning events + text events
	events, err = EventsFromMessage(msg, nil, true)
	require.Nil(t, err)
	require.Len(t, events, 8) // 5 reasoning + 3 text
	assert.Equal(t, EventTypeReasoningStart, events[0].EventType())
	reasoningContent := events[2].(*ReasoningMessageContentEvent)
	assert.Equal(t, "my thoughts", reasoningContent.Delta)
	assert.Equal(t, EventTypeReasoningEnd, events[4].EventType())
	assert.Equal(t, EventTypeTextMessageStart, events[5].EventType())
}

func TestEventsFromMessage_AssistantToolCall(t *testing.T) {
	tc, _ := gollem.NewToolCallContent("call-1", "get_workspace", map[string]any{"id": "ws-1"})
	content, _ := json.Marshal([]gollem.MessageContent{tc})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-4"},
		Role:     "assistant",
		Content:  content,
	}

	events, err := EventsFromMessage(msg, nil, false)
	require.Nil(t, err)
	require.GreaterOrEqual(t, len(events), 3) // start + at least 1 args + end

	start := events[0].(*ToolCallStartEvent)
	assert.Equal(t, "call-1", start.ToolCallID)
	assert.Equal(t, "get_workspace", start.ToolCallName)
	assert.Equal(t, "msg-4", start.ParentMessageID)

	// Last event should be ToolCallEnd
	end := events[len(events)-1].(*ToolCallEndEvent)
	assert.Equal(t, "call-1", end.ToolCallID)
}

func TestEventsFromMessage_ToolRole(t *testing.T) {
	tr, _ := gollem.NewToolResponseContent("call-1", "get_workspace", map[string]any{"name": "ws"}, false)
	content, _ := json.Marshal([]gollem.MessageContent{tr})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-5"},
		Role:     "tool",
		Content:  nil, // tool content loaded from store
	}

	loader := func(_ string) (json.RawMessage, error) {
		return content, nil
	}

	events, err := EventsFromMessage(msg, loader, false)
	require.Nil(t, err)
	require.Len(t, events, 1)

	result := events[0].(*ToolCallResultEvent)
	assert.Equal(t, "call-1", result.ToolCallID)
	assert.Equal(t, "tool", result.Role)
}

func TestEventsFromMessage_NilContent(t *testing.T) {
	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-6"},
		Role:     "user",
		Content:  nil,
	}

	events, err := EventsFromMessage(msg, nil, false)
	assert.NotNil(t, err)
	assert.Nil(t, events)
}

func TestEventsFromMessage_InvalidJSON(t *testing.T) {
	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-bad"},
		Role:     "user",
		Content:  json.RawMessage(`not valid json`),
	}

	events, err := EventsFromMessage(msg, nil, false)
	assert.NotNil(t, err)
	assert.Nil(t, events)
}

func TestEventsFromMessage_ToolLoaderError(t *testing.T) {
	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-tool-err"},
		Role:     "tool",
		Content:  nil,
	}

	loader := func(_ string) (json.RawMessage, error) {
		return nil, fmt.Errorf("store error")
	}

	events, err := EventsFromMessage(msg, loader, false)
	assert.NotNil(t, err)
	assert.Nil(t, events)
}

func TestEventsFromMessage_AssistantOnlyThinking(t *testing.T) {
	tc, _ := gollem.NewTextContent("<thinking>only thoughts</thinking>")
	content, _ := json.Marshal([]gollem.MessageContent{tc})

	msg := &models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "msg-think-only"},
		Role:     "assistant",
		Content:  content,
	}

	// Without reasoning — thinking stripped, empty remaining text → no events
	events, err := EventsFromMessage(msg, nil, false)
	require.Nil(t, err)
	assert.Empty(t, events)

	// With reasoning — reasoning events only, no text events
	events, err = EventsFromMessage(msg, nil, true)
	require.Nil(t, err)
	require.Len(t, events, 5)
	assert.Equal(t, EventTypeReasoningStart, events[0].EventType())
	assert.Equal(t, EventTypeReasoningEnd, events[4].EventType())
}
