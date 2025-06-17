package runner

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

func TestInternalTokenProvider_GetToken(t *testing.T) {
	// Test cases
	testCases := []struct {
		name                string
		runnerName          string
		runnerID            string
		mockTokenResponse   []byte
		mockTokenError      error
		expectErrorMessage  string
		validateTokenOutput func(t *testing.T, token string)
	}{
		{
			name:              "success",
			runnerName:        "test-runner",
			runnerID:          "runner-123",
			mockTokenResponse: []byte("test-token"),
			validateTokenOutput: func(t *testing.T, token string) {
				assert.Equal(t, "test-token", token)
			},
		},
		{
			name:               "error generating token",
			runnerName:         "test-runner",
			runnerID:           "runner-123",
			mockTokenError:     assert.AnError,
			expectErrorMessage: assert.AnError.Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			// Create mock identity provider
			mockIDP := auth.NewMockIdentityProvider(t)

			// Set up expectations for the mock
			mockIDP.On("GenerateToken", ctx, mock.MatchedBy(func(input *auth.TokenInput) bool {
				return input.Subject == tc.runnerName &&
					input.Audience == internalRunnerJWTAudience &&
					input.Typ == internalRunnerJWTType &&
					input.Claims["runner_id"] == gid.ToGlobalID(types.RunnerModelType, tc.runnerID) &&
					input.Expiration != nil
			})).Return(tc.mockTokenResponse, tc.mockTokenError)

			// Create the token provider
			tokenProvider := NewInternalTokenProvider(tc.runnerName, tc.runnerID, mockIDP)

			// Call the method under test
			token, err := tokenProvider.GetToken(ctx)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				if tc.validateTokenOutput != nil {
					tc.validateTokenOutput(t, token)
				}
			}

			// Test token caching - should not call GenerateToken again
			if tc.mockTokenError == nil {
				// Call GetToken again
				token2, err := tokenProvider.GetToken(ctx)
				require.NoError(t, err)
				assert.Equal(t, string(tc.mockTokenResponse), token2)
			}
		})
	}
}

func TestInternalTokenProvider_isTokenExpired(t *testing.T) {
	t.Run("should generate new token when expired", func(t *testing.T) {
		ctx := context.Background()
		runnerName := "test-runner"
		runnerID := "runner-123"

		// Create mock identity provider
		mockIDP := auth.NewMockIdentityProvider(t)

		// First token generation
		mockIDP.On("GenerateToken", ctx, mock.MatchedBy(func(input *auth.TokenInput) bool {
			return input.Subject == runnerName &&
				input.Audience == internalRunnerJWTAudience &&
				input.Typ == internalRunnerJWTType &&
				input.Claims["runner_id"] == gid.ToGlobalID(types.RunnerModelType, runnerID) &&
				input.Expiration != nil
		})).Return([]byte("token-1"), nil).Once()

		// Second token generation after expiration
		mockIDP.On("GenerateToken", ctx, mock.MatchedBy(func(input *auth.TokenInput) bool {
			return input.Subject == runnerName &&
				input.Audience == internalRunnerJWTAudience &&
				input.Typ == internalRunnerJWTType &&
				input.Claims["runner_id"] == gid.ToGlobalID(types.RunnerModelType, runnerID) &&
				input.Expiration != nil
		})).Return([]byte("token-2"), nil).Once()

		// Create the token provider
		tokenProvider := NewInternalTokenProvider(runnerName, runnerID, mockIDP)

		// Get the first token
		token1, err := tokenProvider.GetToken(ctx)
		require.NoError(t, err)
		assert.Equal(t, "token-1", token1)

		// Manually set the expiration to a past time to simulate token expiration
		pastTime := time.Now().Add(-1 * time.Minute)
		tokenProvider.mutex.Lock()
		tokenProvider.expires = &pastTime
		tokenProvider.mutex.Unlock()

		// Get token again, should generate a new one
		token2, err := tokenProvider.GetToken(ctx)
		require.NoError(t, err)
		assert.Equal(t, "token-2", token2)
	})
}
