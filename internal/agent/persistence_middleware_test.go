package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func newTestPersister(t *testing.T) (*messagePersister, *db.MockAgentSessionMessages, *db.MockAgentSessionRuns, *db.MockTransactions, *MockStore) {
	t.Helper()

	mockMessages := db.NewMockAgentSessionMessages(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockTx := db.NewMockTransactions(t)
	mockStore := NewMockStore(t)
	testLogger, _ := logger.NewForTest()

	dbClient := &db.Client{
		AgentSessionMessages: mockMessages,
		AgentSessionRuns:     mockRuns,
		Transactions:         mockTx,
	}

	p := newMessagePersister(dbClient, mockStore, testLogger, "session-1", "run-1", nil, nil)
	return p, mockMessages, mockRuns, mockTx, mockStore
}

// setupSaveMocks sets up the standard mock expectations for a successful save.
func setupSaveMocks(mockMessages *db.MockAgentSessionMessages, mockRuns *db.MockAgentSessionRuns, mockTx *db.MockTransactions, msgID string) {
	mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil).Once()
	mockTx.On("RollbackTx", mock.Anything).Return(nil).Once()
	mockTx.On("CommitTx", mock.Anything).Return(nil).Once()

	mockMessages.On("CreateAgentSessionMessage", mock.Anything, mock.Anything).Return(&models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: msgID},
	}, nil).Once()

	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "run-1").Return(&models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: "run-1", Version: 1},
		SessionID: "session-1",
	}, nil).Once()
	mockRuns.On("UpdateAgentSessionRun", mock.Anything, mock.Anything).Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 2},
	}, nil).Once()
}

func TestPersister_SaveUserInput(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, _ := newTestPersister(t)

	setupSaveMocks(mockMessages, mockRuns, mockTx, "msg-1")

	err := p.saveUserInput(context.Background(), "hello world")
	require.Nil(t, err)
	assert.Equal(t, "msg-1", *p.lastMessageID)

	// Verify the message was created with user role
	call := mockMessages.Calls[0]
	msg := call.Arguments[1].(*models.AgentSessionMessage)
	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "session-1", msg.SessionID)
	assert.Equal(t, "run-1", msg.RunID)
	assert.Nil(t, msg.ParentID) // first message has no parent
}

func TestPersister_SaveChainsSetsParentID(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, _ := newTestPersister(t)

	// First save
	setupSaveMocks(mockMessages, mockRuns, mockTx, "msg-1")
	err := p.saveUserInput(context.Background(), "first")
	require.Nil(t, err)

	// Second save should have parent = msg-1
	setupSaveMocks(mockMessages, mockRuns, mockTx, "msg-2")
	err = p.saveUserInput(context.Background(), "second")
	require.Nil(t, err)

	call := mockMessages.Calls[1]
	msg := call.Arguments[1].(*models.AgentSessionMessage)
	require.NotNil(t, msg.ParentID)
	assert.Equal(t, "msg-1", *msg.ParentID)
}

func TestPersister_ToolContentUploadedToStore(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, mockStore := newTestPersister(t)

	mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil).Once()
	mockTx.On("RollbackTx", mock.Anything).Return(nil).Once()
	mockTx.On("CommitTx", mock.Anything).Return(nil).Once()

	mockMessages.On("CreateAgentSessionMessage", mock.Anything, mock.MatchedBy(func(msg *models.AgentSessionMessage) bool {
		return msg.Role == "tool" && msg.Content == nil // tool content should be nil in DB
	})).Return(&models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "tool-msg-1"},
	}, nil).Once()

	mockStore.On("UploadToolContent", mock.Anything, "session-1", "tool-msg-1", mock.Anything).Return(nil).Once()

	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "run-1").Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 1},
	}, nil).Once()
	mockRuns.On("UpdateAgentSessionRun", mock.Anything, mock.Anything).Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 2},
	}, nil).Once()

	content := json.RawMessage(`[{"type":"tool_response"}]`)
	err := p.save(context.Background(), string(gollem.RoleTool), content)
	require.Nil(t, err)
}

func TestPersister_ContentBlockMiddleware_SavesAssistantText(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, _ := newTestPersister(t)

	setupSaveMocks(mockMessages, mockRuns, mockTx, "assistant-msg-1")

	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			Texts:       []string{"hello from assistant"},
			InputToken:  10,
			OutputToken: 20,
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Equal(t, []string{"hello from assistant"}, resp.Texts)
	assert.Equal(t, "assistant-msg-1", *p.lastMessageID)
}

func TestPersister_ContentBlockMiddleware_SavesFunctionCalls(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, _ := newTestPersister(t)

	setupSaveMocks(mockMessages, mockRuns, mockTx, "fc-msg-1")

	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			FunctionCalls: []*gollem.FunctionCall{
				{ID: "call-1", Name: "get_workspace", Arguments: map[string]any{"id": "ws-1"}},
			},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Len(t, resp.FunctionCalls, 1)
}

func TestPersister_ContentBlockMiddleware_SkipsExcludedTools(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	mockMessages := db.NewMockAgentSessionMessages(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockTx := db.NewMockTransactions(t)
	mockStore := NewMockStore(t)

	p := newMessagePersister(
		&db.Client{AgentSessionMessages: mockMessages, AgentSessionRuns: mockRuns, Transactions: mockTx},
		mockStore, testLogger, "session-1", "run-1", nil,
		[]string{"excluded_tool"},
	)

	// No save mocks — nothing should be saved since the only function call is excluded
	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			FunctionCalls: []*gollem.FunctionCall{
				{ID: "call-1", Name: "excluded_tool", Arguments: map[string]any{}},
			},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Len(t, resp.FunctionCalls, 1)
}

func TestPersister_ContentBlockMiddleware_PassesThroughErrors(t *testing.T) {
	p, _, _, _, _ := newTestPersister(t)

	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return nil, fmt.Errorf("llm error")
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "llm error")
}

func TestPersister_ToolMiddleware_SavesToolResult(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, mockStore := newTestPersister(t)

	// Tool role triggers object store upload
	mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil).Once()
	mockTx.On("RollbackTx", mock.Anything).Return(nil).Once()
	mockTx.On("CommitTx", mock.Anything).Return(nil).Once()
	mockMessages.On("CreateAgentSessionMessage", mock.Anything, mock.Anything).Return(&models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "tool-result-1"},
	}, nil).Once()
	mockStore.On("UploadToolContent", mock.Anything, "session-1", "tool-result-1", mock.Anything).Return(nil).Once()
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "run-1").Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 1},
	}, nil).Once()
	mockRuns.On("UpdateAgentSessionRun", mock.Anything, mock.Anything).Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 2},
	}, nil).Once()

	mw := p.ToolMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Result: map[string]any{"output": "success"},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{ID: "call-1", Name: "get_workspace"},
	})
	require.Nil(t, err)
	assert.Equal(t, map[string]any{"output": "success"}, resp.Result)
}

func TestPersister_ToolMiddleware_SkipsExcludedTools(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	p := newMessagePersister(
		&db.Client{},
		nil, testLogger, "session-1", "run-1", nil,
		[]string{"excluded_tool"},
	)

	mw := p.ToolMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{Result: map[string]any{"ok": true}}, nil
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{ID: "call-1", Name: "excluded_tool"},
	})
	require.Nil(t, err)
	assert.Equal(t, map[string]any{"ok": true}, resp.Result)
}

func TestPersister_ToolMiddleware_PassesThroughErrors(t *testing.T) {
	p, _, _, _, _ := newTestPersister(t)

	mw := p.ToolMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return nil, fmt.Errorf("tool error")
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{ID: "call-1", Name: "some_tool"},
	})
	assert.Nil(t, resp)
	assert.EqualError(t, err, "tool error")
}

func TestPersister_ToolMiddleware_SavesToolError(t *testing.T) {
	p, mockMessages, mockRuns, mockTx, mockStore := newTestPersister(t)

	mockTx.On("BeginTx", mock.Anything).Return(context.Background(), nil).Once()
	mockTx.On("RollbackTx", mock.Anything).Return(nil).Once()
	mockTx.On("CommitTx", mock.Anything).Return(nil).Once()
	mockMessages.On("CreateAgentSessionMessage", mock.Anything, mock.Anything).Return(&models.AgentSessionMessage{
		Metadata: models.ResourceMetadata{ID: "tool-err-1"},
	}, nil).Once()
	mockStore.On("UploadToolContent", mock.Anything, "session-1", "tool-err-1", mock.Anything).Return(nil).Once()
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "run-1").Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 1},
	}, nil).Once()
	mockRuns.On("UpdateAgentSessionRun", mock.Anything, mock.Anything).Return(&models.AgentSessionRun{
		Metadata: models.ResourceMetadata{ID: "run-1", Version: 2},
	}, nil).Once()

	mw := p.ToolMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ToolExecRequest) (*gollem.ToolExecResponse, error) {
		return &gollem.ToolExecResponse{
			Error: fmt.Errorf("tool execution failed"),
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ToolExecRequest{
		Tool: &gollem.FunctionCall{ID: "call-1", Name: "some_tool"},
	})
	require.Nil(t, err)
	assert.NotNil(t, resp.Error)
}

func TestPersister_ContentBlockMiddleware_SavesTextButSkipsExcludedToolCall(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	mockMessages := db.NewMockAgentSessionMessages(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockTx := db.NewMockTransactions(t)
	mockStore := NewMockStore(t)

	p := newMessagePersister(
		&db.Client{AgentSessionMessages: mockMessages, AgentSessionRuns: mockRuns, Transactions: mockTx},
		mockStore, testLogger, "session-1", "run-1", nil,
		[]string{"excluded_tool"},
	)

	// Text content should still be saved even though the tool call is excluded
	setupSaveMocks(mockMessages, mockRuns, mockTx, "mixed-msg-1")

	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			Texts: []string{"some text"},
			FunctionCalls: []*gollem.FunctionCall{
				{ID: "call-1", Name: "excluded_tool", Arguments: map[string]any{}},
			},
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Equal(t, []string{"some text"}, resp.Texts)
	assert.Equal(t, "mixed-msg-1", *p.lastMessageID)
}

func TestPersister_ContentBlockMiddleware_SkipsOnResponseError(t *testing.T) {
	p, _, _, _, _ := newTestPersister(t)

	mw := p.ContentBlockMiddleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{Error: fmt.Errorf("response error")}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.NotNil(t, resp.Error)
	assert.Nil(t, p.lastMessageID) // nothing should have been saved
}

func TestPersister_NewMessagePersisterWithInitialLastMessageID(t *testing.T) {
	testLogger, _ := logger.NewForTest()
	initialID := "existing-msg"
	p := newMessagePersister(&db.Client{}, nil, testLogger, "s", "r", &initialID, nil)
	assert.Equal(t, "existing-msg", *p.lastMessageID)
}
