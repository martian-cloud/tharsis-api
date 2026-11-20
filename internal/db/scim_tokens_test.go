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
				TokenHash:    uuid.New().String(),
				IDPIssuerURL: "https://idp.example.com",
				CreatedBy:    models.Actor{Subject: "db-integration-tests"},
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
			assert.Equal(t, test.token.TokenHash, token.TokenHash)
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
		TokenHash:    uuid.New().String(),
		IDPIssuerURL: "https://idp.example.com",
		CreatedBy:    models.Actor{Subject: "db-integration-tests"},
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
			deletedToken, err := testClient.client.SCIMTokens.GetTokenByHash(ctx, test.token.TokenHash)
			if err != nil {
				assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
			}
			assert.Nil(t, deletedToken)
		})
	}
}

func TestSCIMTokens_GetTokenByHash(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a SCIM token for testing
	createdToken, err := testClient.client.SCIMTokens.CreateToken(ctx, &models.SCIMToken{
		TokenHash:    "token-hash-1",
		IDPIssuerURL: "https://idp.example.com",
		CreatedBy:    models.Actor{Subject: "db-integration-tests"},
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		tokenHash       string
		expectToken     bool
	}

	testCases := []testCase{
		{
			name:        "get resource by hash",
			tokenHash:   createdToken.TokenHash,
			expectToken: true,
		},
		{
			name:      "resource with tokenHash not found",
			tokenHash: "missing-token-hash",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			token, err := testClient.client.SCIMTokens.GetTokenByHash(ctx, test.tokenHash)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectToken {
				require.NotNil(t, token)
				assert.Equal(t, test.tokenHash, token.TokenHash)
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
			TokenHash:    "token-hash-2",
			IDPIssuerURL: "https://idp.example.com",
			CreatedBy:    models.Actor{Subject: "db-integration-tests"},
		},
		{
			TokenHash:    "token-hash-3",
			IDPIssuerURL: "https://idp.example.com",
			CreatedBy:    models.Actor{Subject: "db-integration-tests"},
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
