package resolver

import (
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/agent"
)

func TestAgentSessionToolCallStartEventResolver_Timestamp(t *testing.T) {
	type testCase struct {
		name      string
		timestamp *int64
		expect    *Int64
	}

	testCases := []testCase{
		{
			// Epoch milliseconds exceed the 32-bit Int range, so the full value must come through.
			name:      "returns epoch milliseconds without truncation",
			timestamp: ptr.Int64(1_763_500_000_123),
			expect:    int64PtrToInt64ScalarPtr(ptr.Int64(1_763_500_000_123)),
		},
		{
			name:      "returns nil when the event has no timestamp",
			timestamp: nil,
			expect:    nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			r := &AgentSessionToolCallStartEventResolver{
				event: &agent.ToolCallStartEvent{BaseEvent: agent.BaseEvent{Type: agent.EventTypeToolCallStart, Timestamp: test.timestamp}},
			}

			assert.Equal(t, test.expect, r.Timestamp())
		})
	}
}

func TestAgentSessionToolCallResultEventResolver_Timestamp(t *testing.T) {
	type testCase struct {
		name      string
		timestamp *int64
		expect    *Int64
	}

	testCases := []testCase{
		{
			name:      "returns epoch milliseconds without truncation",
			timestamp: ptr.Int64(1_763_500_004_567),
			expect:    int64PtrToInt64ScalarPtr(ptr.Int64(1_763_500_004_567)),
		},
		{
			name:      "returns nil when the event has no timestamp",
			timestamp: nil,
			expect:    nil,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			r := &AgentSessionToolCallResultEventResolver{
				event: &agent.ToolCallResultEvent{BaseEvent: agent.BaseEvent{Type: agent.EventTypeToolCallResult, Timestamp: test.timestamp}},
			}

			assert.Equal(t, test.expect, r.Timestamp())
		})
	}
}
