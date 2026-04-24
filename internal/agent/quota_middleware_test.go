package agent

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestQuotaMiddleware_AllowsWhenUnderLimit(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)
	mockSessions := db.NewMockAgentSessions(t)
	mockLLM := llm.NewMockClient(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1", Version: 1},
		UserID:   "user-1",
	}

	quota := &models.AgentCreditQuota{
		Metadata:     models.ResourceMetadata{ID: "quota-1"},
		UserID:       "user-1",
		TotalCredits: 5.0,
	}

	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(quota, nil)
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(&models.ResourceLimit{Value: 100}, nil)
	mockLLM.On("GetCreditCount", mock.Anything).Return(float64(2.5))
	mockQuotas.On("AddCredits", mock.Anything, "quota-1", 2.5).Return(nil)

	updatedSession := &models.AgentSession{
		Metadata:     models.ResourceMetadata{ID: "session-1", Version: 2},
		UserID:       "user-1",
		TotalCredits: 2.5,
	}
	mockSessions.On("UpdateAgentSession", mock.Anything, mock.Anything).Return(updatedSession, nil)

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
			AgentSessions:     mockSessions,
		},
		llmClient: mockLLM,
		session:   session,
	}

	mw := qm.Middleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{
			Texts:       []string{"response"},
			InputToken:  100,
			OutputToken: 50,
		}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Equal(t, []string{"response"}, resp.Texts)

	// Verify session credits were updated
	assert.Equal(t, updatedSession, qm.session)
}

func TestQuotaMiddleware_BlocksWhenOverLimit(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1"},
		UserID:   "user-1",
	}

	quota := &models.AgentCreditQuota{
		Metadata:     models.ResourceMetadata{ID: "quota-1"},
		UserID:       "user-1",
		TotalCredits: 100.0, // at limit
	}

	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(quota, nil)
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(&models.ResourceLimit{Value: 100}, nil)

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
		},
		session: session,
	}

	mw := qm.Middleware()
	called := false
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		called = true
		return &gollem.ContentResponse{}, nil
	})

	_, err := handler(context.Background(), &gollem.ContentRequest{})
	assert.Equal(t, errors.EForbidden, errors.ErrorCode(err))
	assert.False(t, called, "next handler should not be called when over limit")
}

func TestQuotaMiddleware_NoLimitConfigured(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)
	mockSessions := db.NewMockAgentSessions(t)
	mockLLM := llm.NewMockClient(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1", Version: 1},
		UserID:   "user-1",
	}

	quota := &models.AgentCreditQuota{
		Metadata:     models.ResourceMetadata{ID: "quota-1"},
		TotalCredits: 999.0,
	}

	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(quota, nil)
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(nil, nil) // no limit configured
	mockLLM.On("GetCreditCount", mock.Anything).Return(float64(0))

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
			AgentSessions:     mockSessions,
		},
		llmClient: mockLLM,
		session:   session,
	}

	mw := qm.Middleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{Texts: []string{"ok"}}, nil
	})

	resp, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	assert.Equal(t, []string{"ok"}, resp.Texts)
}

func TestQuotaMiddleware_CreatesQuotaOnFirstUse(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)
	mockSessions := db.NewMockAgentSessions(t)
	mockLLM := llm.NewMockClient(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1", Version: 1},
		UserID:   "user-1",
	}

	newQuota := &models.AgentCreditQuota{
		Metadata: models.ResourceMetadata{ID: "new-quota"},
		UserID:   "user-1",
	}

	// First call returns nil (no quota), then create succeeds
	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(nil, nil).Once()
	mockQuotas.On("CreateAgentCreditQuota", mock.Anything, mock.Anything).Return(newQuota, nil).Once()
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(nil, nil)
	mockLLM.On("GetCreditCount", mock.Anything).Return(float64(0))

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
			AgentSessions:     mockSessions,
		},
		llmClient: mockLLM,
		session:   session,
	}

	mw := qm.Middleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{}, nil
	})

	_, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	mockQuotas.AssertCalled(t, "CreateAgentCreditQuota", mock.Anything, mock.Anything)
}

func TestQuotaMiddleware_ConflictOnCreateRequeriesQuota(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)
	mockSessions := db.NewMockAgentSessions(t)
	mockLLM := llm.NewMockClient(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1", Version: 1},
		UserID:   "user-1",
	}

	existingQuota := &models.AgentCreditQuota{
		Metadata: models.ResourceMetadata{ID: "existing-quota"},
		UserID:   "user-1",
	}

	// First GetAgentCreditQuota returns nil, create returns conflict, second get returns existing
	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(nil, nil).Once()
	mockQuotas.On("CreateAgentCreditQuota", mock.Anything, mock.Anything).Return(nil, errors.New("conflict", errors.WithErrorCode(errors.EConflict))).Once()
	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(existingQuota, nil).Once()
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(nil, nil)
	mockLLM.On("GetCreditCount", mock.Anything).Return(float64(0))

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
			AgentSessions:     mockSessions,
		},
		llmClient: mockLLM,
		session:   session,
	}

	mw := qm.Middleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{}, nil
	})

	_, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
}

func TestQuotaMiddleware_SkipsAddCreditsWhenZero(t *testing.T) {
	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockLimits := db.NewMockResourceLimits(t)
	mockLLM := llm.NewMockClient(t)

	session := &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: "session-1", Version: 1},
		UserID:   "user-1",
	}

	quota := &models.AgentCreditQuota{
		Metadata: models.ResourceMetadata{ID: "quota-1"},
	}

	mockQuotas.On("GetAgentCreditQuota", mock.Anything, "user-1", mock.Anything).Return(quota, nil)
	mockLimits.On("GetResourceLimit", mock.Anything, mock.Anything).Return(nil, nil)
	mockLLM.On("GetCreditCount", mock.Anything).Return(float64(0))
	// AddCredits and UpdateAgentSession should NOT be called

	qm := &quotaMiddleware{
		dbClient: &db.Client{
			AgentCreditQuotas: mockQuotas,
			ResourceLimits:    mockLimits,
		},
		llmClient: mockLLM,
		session:   session,
	}

	mw := qm.Middleware()
	handler := mw(func(_ context.Context, _ *gollem.ContentRequest) (*gollem.ContentResponse, error) {
		return &gollem.ContentResponse{}, nil
	})

	_, err := handler(context.Background(), &gollem.ContentRequest{})
	require.Nil(t, err)
	mockQuotas.AssertNotCalled(t, "AddCredits", mock.Anything, mock.Anything, mock.Anything)
}
