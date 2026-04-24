package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/agent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

var (
	sampleUserID    = "user-1"
	otherUserID     = "user-2"
	sampleSessionID = "session-1"
	sampleRunID     = "run-1"

	sampleUser = &models.User{
		Metadata: models.ResourceMetadata{ID: sampleUserID},
		Username: "testuser",
		Email:    "test@example.com",
	}

	otherUser = &models.User{
		Metadata: models.ResourceMetadata{ID: otherUserID},
		Username: "otheruser",
		Email:    "other@example.com",
	}

	adminUser = &models.User{
		Metadata: models.ResourceMetadata{ID: sampleUserID},
		Username: "adminuser",
		Email:    "admin@example.com",
		Admin:    true,
	}

	sampleSession = &models.AgentSession{
		Metadata: models.ResourceMetadata{ID: sampleSessionID, TRN: "trn:tharsis:agent_session:" + sampleSessionID},
		UserID:   sampleUserID,
	}

	sampleRun = &models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: sampleRunID, TRN: "trn:tharsis:agent_session_run:" + sampleSessionID + "/" + sampleRunID},
		SessionID: sampleSessionID,
		Status:    models.AgentSessionRunRunning,
	}
)

func userCaller(user *models.User) auth.Caller {
	return &auth.UserCaller{User: user}
}

func withCaller(ctx context.Context, caller auth.Caller) context.Context {
	return auth.WithCaller(ctx, caller)
}

// TestGetAgentSessionByID tests authorization: only the session owner can access it.
func TestGetAgentSessionByID(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "owner can access session",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot access session",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "service account caller is forbidden",
			caller:          &auth.ServiceAccountCaller{},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
				}
			}

			svc := &service{aiEnabled: true,
				dbClient: &db.Client{AgentSessions: mockSessions},
			}

			if test.caller != nil {
				ctx = withCaller(ctx, test.caller)
			}

			result, err := svc.GetAgentSessionByID(ctx, sampleSessionID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleSession, result)
		})
	}
}

// TestGetAgentSessionByID_NotFound tests that a non-existent session returns not found.
func TestGetAgentSessionByID_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockSessions := db.NewMockAgentSessions(t)
	mockSessions.On("GetAgentSessionByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

	result, err := svc.GetAgentSessionByID(ctx, "nonexistent")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestGetAgentSessionByTRN tests authorization for TRN-based access.
func TestGetAgentSessionByTRN(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "owner can access session by TRN",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot access session by TRN",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)

			if test.caller != nil {
				mockSessions.On("GetAgentSessionByTRN", mock.Anything, sampleSession.Metadata.TRN).Return(sampleSession, nil)
				mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil).Maybe()
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

			result, err := svc.GetAgentSessionByTRN(ctx, sampleSession.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleSession, result)
		})
	}
}

// TestGetAgentSessionRunByID tests that run access requires owning the parent session.
func TestGetAgentSessionRunByID(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "session owner can access run",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot access run",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "service account caller is forbidden",
			caller:          &auth.ServiceAccountCaller{},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRuns := db.NewMockAgentSessionRuns(t)
			mockSessions := db.NewMockAgentSessions(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockRuns.On("GetAgentSessionRunByID", mock.Anything, sampleRunID).Return(sampleRun, nil)
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{
				AgentSessions:    mockSessions,
				AgentSessionRuns: mockRuns,
			}}

			result, err := svc.GetAgentSessionRunByID(ctx, sampleRunID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleRun, result)
		})
	}
}

// TestGetAgentSessionRunByID_NotFound tests that a non-existent run returns not found.
func TestGetAgentSessionRunByID_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockRuns := db.NewMockAgentSessionRuns(t)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessionRuns: mockRuns}}

	result, err := svc.GetAgentSessionRunByID(ctx, "nonexistent")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestGetAgentSessionRunByTRN tests authorization for TRN-based run access.
func TestGetAgentSessionRunByTRN(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "session owner can access run by TRN",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot access run by TRN",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := withCaller(t.Context(), test.caller)

			mockRuns := db.NewMockAgentSessionRuns(t)
			mockSessions := db.NewMockAgentSessions(t)

			mockRuns.On("GetAgentSessionRunByTRN", mock.Anything, sampleRun.Metadata.TRN).Return(sampleRun, nil)
			mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

			svc := &service{aiEnabled: true, dbClient: &db.Client{
				AgentSessions:    mockSessions,
				AgentSessionRuns: mockRuns,
			}}

			result, err := svc.GetAgentSessionRunByTRN(ctx, sampleRun.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleRun, result)
		})
	}
}

// TestCreateAgentSession tests session creation authorization.
func TestCreateAgentSession(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "user can create session",
			caller: userCaller(sampleUser),
		},
		{
			name:            "service account cannot create session",
			caller:          &auth.ServiceAccountCaller{},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)
			mockTx := db.NewMockTransactions(t)

			if test.caller != nil {
				if uc, ok := test.caller.(*auth.UserCaller); ok {
					mockTx.On("BeginTx", mock.Anything).Return(ctx, nil)
					mockTx.On("RollbackTx", mock.Anything).Return(nil)
					mockTx.On("CommitTx", mock.Anything).Return(nil)
					mockSessions.On("DeleteOldestSessionsByUserID", mock.Anything, uc.User.Metadata.ID, 100).Return(nil)
					mockSessions.On("CreateAgentSession", mock.Anything, mock.Anything).Return(sampleSession, nil)
				}
				ctx = withCaller(ctx, test.caller)
			}

			testLogger, _ := logger.NewForTest()
			svc := &service{aiEnabled: true,
				logger:   testLogger,
				dbClient: &db.Client{AgentSessions: mockSessions, Transactions: mockTx},
			}

			result, err := svc.CreateAgentSession(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleSession, result)
		})
	}
}

// TestCreateAgentRun tests run creation authorization and validation.
func TestCreateAgentRun(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		input           *CreateAgentRunInput
		expectErrorCode errors.CodeType
	}{
		{
			name:   "session owner can create run",
			caller: userCaller(sampleUser),
			input:  &CreateAgentRunInput{SessionID: sampleSessionID, Message: "hello"},
		},
		{
			name:            "different user cannot create run in another user's session",
			caller:          userCaller(otherUser),
			input:           &CreateAgentRunInput{SessionID: sampleSessionID, Message: "hello"},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "empty message is invalid",
			caller:          userCaller(sampleUser),
			input:           &CreateAgentRunInput{SessionID: sampleSessionID, Message: ""},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "no caller returns unauthorized",
			input:           &CreateAgentRunInput{SessionID: sampleSessionID, Message: "hello"},
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)
			mockRuns := db.NewMockAgentSessionRuns(t)
			mockTaskManager := asynctask.NewMockManager(t)
			mockLimitChecker := limits.NewMockLimitChecker(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

					// Only set up run creation mocks for the owner with a valid message
					if test.expectErrorCode == "" {
						mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.Anything).Return(&db.AgentSessionRunsResult{
							PageInfo: &pagination.PageInfo{TotalCount: 0},
						}, nil)
						mockLimitChecker.On("CheckLimit", mock.Anything, limits.ResourceLimitAgentSessionRunsPerSession, int32(0)).Return(nil)
						mockRuns.On("CreateAgentSessionRun", mock.Anything, mock.Anything).Return(sampleRun, nil)
						mockTaskManager.On("StartTask", mock.Anything).Return()
					}
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true,
				dbClient: &db.Client{
					AgentSessions:    mockSessions,
					AgentSessionRuns: mockRuns,
				},
				taskManager:  mockTaskManager,
				limitChecker: mockLimitChecker,
				toolSetFactory: func(_ context.Context) (gollem.ToolSet, error) {
					return nil, nil
				},
			}

			result, err := svc.CreateAgentRun(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, sampleRun, result)
		})
	}
}

// TestCancelAgentRun tests cancel authorization: only the session owner can cancel.
func TestCancelAgentRun(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "session owner can cancel run",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot cancel run",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRuns := db.NewMockAgentSessionRuns(t)
			mockSessions := db.NewMockAgentSessions(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockRuns.On("GetAgentSessionRunByID", mock.Anything, sampleRunID).Return(sampleRun, nil)
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

					if test.expectErrorCode == "" {
						updatedRun := *sampleRun
						updatedRun.CancelRequested = true
						mockRuns.On("UpdateAgentSessionRun", mock.Anything, mock.Anything).Return(&updatedRun, nil)
					}
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{
				AgentSessions:    mockSessions,
				AgentSessionRuns: mockRuns,
			}}

			result, err := svc.CancelAgentRun(ctx, &CancelAgentRunInput{RunID: sampleRunID})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.True(t, result.CancelRequested)
		})
	}
}

// TestCancelAgentRun_NotRunning tests that cancelling a finished run fails.
func TestCancelAgentRun_NotRunning(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	finishedRun := &models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: sampleRunID},
		SessionID: sampleSessionID,
		Status:    models.AgentSessionRunFinished,
	}

	mockRuns := db.NewMockAgentSessionRuns(t)
	mockSessions := db.NewMockAgentSessions(t)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, sampleRunID).Return(finishedRun, nil)
	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{
		AgentSessions:    mockSessions,
		AgentSessionRuns: mockRuns,
	}}

	_, err := svc.CancelAgentRun(ctx, &CancelAgentRunInput{RunID: sampleRunID})
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
}

// TestGetAgentSessionRuns tests that listing runs requires owning the session.
func TestGetAgentSessionRuns(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "session owner can list runs",
			caller: userCaller(sampleUser),
		},
		{
			name:            "different user cannot list runs",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)
			mockRuns := db.NewMockAgentSessionRuns(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

					if test.expectErrorCode == "" {
						mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.MatchedBy(func(input *db.GetAgentSessionRunsInput) bool {
							return input.Filter != nil && input.Filter.SessionID != nil && *input.Filter.SessionID == sampleSessionID
						})).Return(&db.AgentSessionRunsResult{
							AgentSessionRuns: []models.AgentSessionRun{*sampleRun},
						}, nil)
					}
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{
				AgentSessions:    mockSessions,
				AgentSessionRuns: mockRuns,
			}}

			result, err := svc.GetAgentSessionRuns(ctx, &GetAgentSessionRunsInput{SessionID: sampleSessionID})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Len(t, result.AgentSessionRuns, 1)
		})
	}
}

// TestGetAgentTrace tests that only admins can access trace data.
func TestGetAgentTrace(t *testing.T) {
	traceData := []byte(`{"spans":[]}`)

	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:   "admin can access trace",
			caller: userCaller(adminUser),
		},
		{
			name:            "non-admin cannot access trace",
			caller:          userCaller(sampleUser),
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockRuns := db.NewMockAgentSessionRuns(t)
			mockStore := agent.NewMockStore(t)
			testLogger, _ := logger.NewForTest()

			if test.caller != nil {
				if uc, ok := test.caller.(*auth.UserCaller); ok && uc.IsAdmin() {
					mockRuns.On("GetAgentSessionRunByID", mock.Anything, sampleRunID).Return(sampleRun, nil)
					mockStore.On("GetTrace", mock.Anything, sampleSessionID, sampleRunID).Return(traceData, nil)
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true,
				dbClient:   &db.Client{AgentSessionRuns: mockRuns},
				agentStore: mockStore,
				logger:     testLogger,
			}

			result, err := svc.GetAgentTrace(ctx, sampleRunID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, result)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, json.RawMessage(traceData), result)
		})
	}
}

// TestGetAgentCreditUsage tests credit usage authorization.
func TestGetAgentCreditUsage(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		queryUserID     string
		expectErrorCode errors.CodeType
		expectCredits   float64
	}{
		{
			name:          "user can view own credit usage",
			caller:        userCaller(sampleUser),
			queryUserID:   sampleUserID,
			expectCredits: 42.5,
		},
		{
			name:            "user cannot view another user's credit usage",
			caller:          userCaller(sampleUser),
			queryUserID:     otherUserID,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:          "admin can view any user's credit usage",
			caller:        userCaller(adminUser),
			queryUserID:   otherUserID,
			expectCredits: 42.5,
		},
		{
			name:            "no caller returns unauthorized",
			queryUserID:     sampleUserID,
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockQuotas := db.NewMockAgentCreditQuotas(t)

			if test.caller != nil && test.expectErrorCode == "" {
				mockQuotas.On("GetAgentCreditQuota", mock.Anything, test.queryUserID, mock.Anything).Return(&models.AgentCreditQuota{
					TotalCredits: 42.5,
				}, nil)
				ctx = withCaller(ctx, test.caller)
			} else if test.caller != nil {
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{AgentCreditQuotas: mockQuotas}}

			result, err := svc.GetAgentCreditUsage(ctx, test.queryUserID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, test.expectCredits, result)
		})
	}
}

// TestGetAgentCreditUsage_NoQuota tests that missing quota returns 0.
func TestGetAgentCreditUsage_NoQuota(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockQuotas := db.NewMockAgentCreditQuotas(t)
	mockQuotas.On("GetAgentCreditQuota", mock.Anything, sampleUserID, mock.Anything).Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentCreditQuotas: mockQuotas}}

	result, err := svc.GetAgentCreditUsage(ctx, sampleUserID)
	assert.Nil(t, err)
	assert.Equal(t, float64(0), result)
}

// TestGetAgentSessionByTRN_NotFound tests that a non-existent session TRN returns not found.
func TestGetAgentSessionByTRN_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockSessions := db.NewMockAgentSessions(t)
	mockSessions.On("GetAgentSessionByTRN", mock.Anything, "trn:tharsis:agent_session:nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

	result, err := svc.GetAgentSessionByTRN(ctx, "trn:tharsis:agent_session:nonexistent")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestGetAgentSessionRunByTRN_NotFound tests that a non-existent run TRN returns not found.
func TestGetAgentSessionRunByTRN_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockRuns := db.NewMockAgentSessionRuns(t)
	mockRuns.On("GetAgentSessionRunByTRN", mock.Anything, "trn:tharsis:agent_session_run:nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessionRuns: mockRuns}}

	result, err := svc.GetAgentSessionRunByTRN(ctx, "trn:tharsis:agent_session_run:nonexistent")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCancelAgentRun_NotFound tests that cancelling a non-existent run returns not found.
func TestCancelAgentRun_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockRuns := db.NewMockAgentSessionRuns(t)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessionRuns: mockRuns}}

	result, err := svc.CancelAgentRun(ctx, &CancelAgentRunInput{RunID: "nonexistent"})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestGetAgentTrace_RunNotFound tests that trace for a non-existent run returns not found.
func TestGetAgentTrace_RunNotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(adminUser))

	testLogger, _ := logger.NewForTest()
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessionRuns: mockRuns}, logger: testLogger}

	result, err := svc.GetAgentTrace(ctx, "nonexistent")
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCreateAgentRun_PreviousRunNotFound tests that referencing a non-existent previous run fails.
func TestCreateAgentRun_PreviousRunNotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockSessions := db.NewMockAgentSessions(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockLimitChecker := limits.NewMockLimitChecker(t)

	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.Anything).Return(&db.AgentSessionRunsResult{
		PageInfo: &pagination.PageInfo{TotalCount: 0},
	}, nil)
	mockLimitChecker.On("CheckLimit", mock.Anything, limits.ResourceLimitAgentSessionRunsPerSession, int32(0)).Return(nil)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true,
		dbClient: &db.Client{
			AgentSessions:    mockSessions,
			AgentSessionRuns: mockRuns,
		},
		limitChecker: mockLimitChecker,
	}

	result, err := svc.CreateAgentRun(ctx, &CreateAgentRunInput{
		SessionID:     sampleSessionID,
		Message:       "hello",
		PreviousRunID: ptr.String("nonexistent"),
	})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCreateAgentRun_PreviousRunStillRunning tests that referencing a still-running previous run fails.
func TestCreateAgentRun_PreviousRunStillRunning(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	runningPrevRun := &models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: "prev-run"},
		SessionID: sampleSessionID,
		Status:    models.AgentSessionRunRunning,
	}

	mockSessions := db.NewMockAgentSessions(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockLimitChecker := limits.NewMockLimitChecker(t)

	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.Anything).Return(&db.AgentSessionRunsResult{
		PageInfo: &pagination.PageInfo{TotalCount: 0},
	}, nil)
	mockLimitChecker.On("CheckLimit", mock.Anything, limits.ResourceLimitAgentSessionRunsPerSession, int32(0)).Return(nil)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "prev-run").Return(runningPrevRun, nil)

	svc := &service{aiEnabled: true,
		dbClient: &db.Client{
			AgentSessions:    mockSessions,
			AgentSessionRuns: mockRuns,
		},
		limitChecker: mockLimitChecker,
	}

	result, err := svc.CreateAgentRun(ctx, &CreateAgentRunInput{
		SessionID:     sampleSessionID,
		Message:       "hello",
		PreviousRunID: ptr.String("prev-run"),
	})
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCreateAgentRun_PreviousRunAlreadyReferenced tests that a previous run already referenced by another run fails.
func TestCreateAgentRun_PreviousRunAlreadyReferenced(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	finishedPrevRun := &models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: "prev-run"},
		SessionID: sampleSessionID,
		Status:    models.AgentSessionRunFinished,
	}

	mockSessions := db.NewMockAgentSessions(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockLimitChecker := limits.NewMockLimitChecker(t)

	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
	// First call: limit check (filter by SessionID)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.MatchedBy(func(input *db.GetAgentSessionRunsInput) bool {
		return input.Filter != nil && input.Filter.SessionID != nil
	})).Return(&db.AgentSessionRunsResult{
		PageInfo: &pagination.PageInfo{TotalCount: 0},
	}, nil).Once()
	mockLimitChecker.On("CheckLimit", mock.Anything, limits.ResourceLimitAgentSessionRunsPerSession, int32(0)).Return(nil)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "prev-run").Return(finishedPrevRun, nil)
	// Second call: previous run chain check (filter by PreviousRunID)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.MatchedBy(func(input *db.GetAgentSessionRunsInput) bool {
		return input.Filter != nil && input.Filter.PreviousRunID != nil
	})).Return(&db.AgentSessionRunsResult{
		PageInfo: &pagination.PageInfo{TotalCount: 1},
	}, nil).Once()

	svc := &service{aiEnabled: true,
		dbClient: &db.Client{
			AgentSessions:    mockSessions,
			AgentSessionRuns: mockRuns,
		},
		limitChecker: mockLimitChecker,
	}

	result, err := svc.CreateAgentRun(ctx, &CreateAgentRunInput{
		SessionID:     sampleSessionID,
		Message:       "hello",
		PreviousRunID: ptr.String("prev-run"),
	})
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCreateAgentRun_PreviousRunFromDifferentSession tests that referencing a run from a different session fails.
func TestCreateAgentRun_PreviousRunFromDifferentSession(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	otherSessionRun := &models.AgentSessionRun{
		Metadata:  models.ResourceMetadata{ID: "other-run"},
		SessionID: "other-session",
		Status:    models.AgentSessionRunFinished,
	}

	mockSessions := db.NewMockAgentSessions(t)
	mockRuns := db.NewMockAgentSessionRuns(t)
	mockLimitChecker := limits.NewMockLimitChecker(t)

	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.Anything).Return(&db.AgentSessionRunsResult{
		PageInfo: &pagination.PageInfo{TotalCount: 0},
	}, nil)
	mockLimitChecker.On("CheckLimit", mock.Anything, limits.ResourceLimitAgentSessionRunsPerSession, int32(0)).Return(nil)
	mockRuns.On("GetAgentSessionRunByID", mock.Anything, "other-run").Return(otherSessionRun, nil)

	svc := &service{aiEnabled: true,
		dbClient: &db.Client{
			AgentSessions:    mockSessions,
			AgentSessionRuns: mockRuns,
		},
		limitChecker: mockLimitChecker,
	}

	result, err := svc.CreateAgentRun(ctx, &CreateAgentRunInput{
		SessionID:     sampleSessionID,
		Message:       "hello",
		PreviousRunID: ptr.String("other-run"),
	})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestCreateAgentRun_SessionNotFound tests that creating a run in a non-existent session fails.
func TestCreateAgentRun_SessionNotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockSessions := db.NewMockAgentSessions(t)
	mockSessions.On("GetAgentSessionByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

	result, err := svc.CreateAgentRun(ctx, &CreateAgentRunInput{
		SessionID: "nonexistent",
		Message:   "hello",
	})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, result)
}

// TestSubscribeToAgentSession tests authorization for subscribing to a session.
func TestSubscribeToAgentSession(t *testing.T) {
	testCases := []struct {
		name            string
		caller          auth.Caller
		expectErrorCode errors.CodeType
	}{
		{
			name:            "different user cannot subscribe to session",
			caller:          userCaller(otherUser),
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "service account cannot subscribe",
			caller:          &auth.ServiceAccountCaller{},
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "no caller returns unauthorized",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSessions := db.NewMockAgentSessions(t)

			if test.caller != nil {
				if _, ok := test.caller.(*auth.UserCaller); ok {
					mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)
				}
				ctx = withCaller(ctx, test.caller)
			}

			svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

			ch, err := svc.SubscribeToAgentSession(ctx, &SubscribeToAgentSessionInput{SessionID: sampleSessionID})
			assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			assert.Nil(t, ch)
		})
	}
}

// TestSubscribeToAgentSession_NotFound tests subscribing to a non-existent session.
func TestSubscribeToAgentSession_NotFound(t *testing.T) {
	ctx := withCaller(t.Context(), userCaller(sampleUser))

	mockSessions := db.NewMockAgentSessions(t)
	mockSessions.On("GetAgentSessionByID", mock.Anything, "nonexistent").Return(nil, nil)

	svc := &service{aiEnabled: true, dbClient: &db.Client{AgentSessions: mockSessions}}

	ch, err := svc.SubscribeToAgentSession(ctx, &SubscribeToAgentSessionInput{SessionID: "nonexistent"})
	assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
	assert.Nil(t, ch)
}

// TestSubscribeToAgentSession_OwnerSuccess tests that the session owner can subscribe and receive events.
func TestSubscribeToAgentSession_OwnerSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(withCaller(t.Context(), userCaller(sampleUser)))
	defer cancel()

	mockSessions := db.NewMockAgentSessions(t)
	mockSessions.On("GetAgentSessionByID", mock.Anything, sampleSessionID).Return(sampleSession, nil)

	mockRuns := db.NewMockAgentSessionRuns(t)
	mockRuns.On("GetAgentSessionRuns", mock.Anything, mock.MatchedBy(func(input *db.GetAgentSessionRunsInput) bool {
		return input.Filter != nil && input.Filter.SessionID != nil && *input.Filter.SessionID == sampleSessionID
	})).Return(&db.AgentSessionRunsResult{
		AgentSessionRuns: []models.AgentSessionRun{},
	}, nil)

	mockEventChan := make(chan db.Event, 10)
	mockErrChan := make(chan error)
	mockEvents := db.NewMockEvents(t)
	mockEvents.On("Listen", mock.Anything).Return((<-chan db.Event)(mockEventChan), (<-chan error)(mockErrChan)).Maybe()

	testLogger, _ := logger.NewForTest()
	dbClient := &db.Client{
		AgentSessions:    mockSessions,
		AgentSessionRuns: mockRuns,
		Events:           mockEvents,
	}

	eventManager := events.NewEventManager(dbClient, testLogger)
	eventManager.Start(ctx)

	svc := &service{aiEnabled: true,
		logger:       testLogger,
		dbClient:     dbClient,
		eventManager: eventManager,
	}

	ch, err := svc.SubscribeToAgentSession(ctx, &SubscribeToAgentSessionInput{SessionID: sampleSessionID})
	assert.Nil(t, err)
	assert.NotNil(t, ch)

	// Send a CREATE event for a new run
	createData, _ := json.Marshal(db.AgentSessionRunEventData{
		ID:        "new-run-1",
		SessionID: sampleSessionID,
		Status:    string(models.AgentSessionRunRunning),
	})
	mockEventChan <- db.Event{
		Table:  "agent_session_runs",
		Action: "INSERT",
		ID:     "new-run-1",
		Data:   createData,
	}

	// Should receive a RunStartedEvent
	ev := <-ch
	runStarted, ok := ev.(*agent.RunStartedEvent)
	assert.True(t, ok)
	assert.Equal(t, agent.EventTypeRunStarted, runStarted.Type)
	assert.Equal(t, "new-run-1", runStarted.RunID)
	assert.Equal(t, sampleSessionID, runStarted.ThreadID)

	// Send an UPDATE event with finished status
	finishData, _ := json.Marshal(db.AgentSessionRunEventData{
		ID:        "new-run-1",
		SessionID: sampleSessionID,
		Status:    string(models.AgentSessionRunFinished),
	})
	mockEventChan <- db.Event{
		Table:  "agent_session_runs",
		Action: "UPDATE",
		ID:     "new-run-1",
		Data:   finishData,
	}

	// Should receive a RunFinishedEvent
	ev = <-ch
	runFinished, ok := ev.(*agent.RunFinishedEvent)
	assert.True(t, ok)
	assert.Equal(t, agent.EventTypeRunFinished, runFinished.Type)
	assert.Equal(t, "new-run-1", runFinished.RunID)

	// Send an UPDATE event with error status
	errMsg := "something went wrong"
	errorData, _ := json.Marshal(db.AgentSessionRunEventData{
		ID:           "new-run-2",
		SessionID:    sampleSessionID,
		Status:       string(models.AgentSessionRunErrored),
		ErrorMessage: &errMsg,
	})
	mockEventChan <- db.Event{
		Table:  "agent_session_runs",
		Action: "UPDATE",
		ID:     "new-run-2",
		Data:   errorData,
	}

	// Should receive a RunErrorEvent
	ev = <-ch
	runError, ok := ev.(*agent.RunErrorEvent)
	assert.True(t, ok)
	assert.Equal(t, agent.EventTypeRunError, runError.Type)
	assert.Equal(t, "something went wrong", runError.Message)

	cancel()
}
