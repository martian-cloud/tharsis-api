//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSCIMTokens_CreateSCIMToken(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		token           *models.SCIMToken
	}

	testCases := []testCase{
		{
			name: "successfully create SCIM token",
			token: &models.SCIMToken{
				Nonce:     uuid.New().String(),
				CreatedBy: "db-integration-tests",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			token, err := testClient.client.SCIMTokens.CreateToken(ctx, test.token)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, token)
			assert.Equal(t, test.token.Nonce, token.Nonce)
			assert.Equal(t, test.token.CreatedBy, token.CreatedBy)
		})
	}
}

func TestSCIMTokens_DeleteSCIMToken(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a SCIM token to delete
	createdToken, err := testClient.client.SCIMTokens.CreateToken(ctx, &models.SCIMToken{
		Nonce:     uuid.New().String(),
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		token           *models.SCIMToken
	}

	testCases := []testCase{
		{
			name:  "successfully delete SCIM token",
			token: createdToken,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.SCIMTokens.DeleteToken(ctx, test.token)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify the token was deleted
			deletedToken, err := testClient.client.SCIMTokens.GetTokenByNonce(ctx, test.token.Nonce)
			if err != nil {
				assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
			}
			assert.Nil(t, deletedToken)
		})
	}
}

func TestSCIMTokens_GetTokenByNonce(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a SCIM token for testing
	createdToken, err := testClient.client.SCIMTokens.CreateToken(ctx, &models.SCIMToken{
		Nonce:     "550e8400-e29b-41d4-a716-446655440000",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		nonce           string
		expectToken     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by nonce",
			nonce:       createdToken.Nonce,
			expectToken: true,
		},
		{
			name:  "resource with nonce not found",
			nonce: "550e8400-e29b-41d4-a716-446655440999",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			token, err := testClient.client.SCIMTokens.GetTokenByNonce(ctx, test.nonce)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectToken {
				require.NotNil(t, token)
				assert.Equal(t, test.nonce, token.Nonce)
			} else {
				assert.Nil(t, token)
			}
		})
	}
}

func TestSCIMTokens_GetTokens(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create test SCIM tokens
	tokens := []models.SCIMToken{
		{
			Nonce:     "550e8400-e29b-41d4-a716-446655440001",
			CreatedBy: "db-integration-tests",
		},
		{
			Nonce:     "550e8400-e29b-41d4-a716-446655440002",
			CreatedBy: "db-integration-tests",
		},
	}

	createdTokens := []models.SCIMToken{}
	for _, token := range tokens {
		created, err := testClient.client.SCIMTokens.CreateToken(ctx, &token)
		require.NoError(t, err)
		createdTokens = append(createdTokens, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		expectMinCount  int
	}

	testCases := []testCase{
		{
			name:           "get all tokens",
			expectMinCount: len(createdTokens),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.SCIMTokens.GetTokens(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(result), test.expectMinCount)
		})
	}
}
