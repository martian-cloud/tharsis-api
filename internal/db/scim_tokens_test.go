//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestGetTokenByNonce(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupToken, err := createInitialSCIMToken(ctx, testClient, standardWarmupSCIMToken)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup token was not created.
		return
	}

	type testCase struct {
		expectMsg       *string
		name            string
		searchNonce     string
		expectSCIMToken bool
	}

	testCases := []testCase{
		{
			name:            "positive-" + createdWarmupToken.Metadata.ID,
			searchNonce:     createdWarmupToken.Nonce,
			expectSCIMToken: true,
		},
		{
			name:        "negative, non-existent Nonce",
			searchNonce: nonExistentID,
		},
		{
			name:        "defective-nonce",
			searchNonce: invalidID,
			expectMsg:   invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			token, err := testClient.client.SCIMTokens.GetTokenByNonce(ctx, test.searchNonce)

			checkError(t, test.expectMsg, err)

			if test.expectSCIMToken {
				// the positive case
				require.NotNil(t, token)
				assert.Equal(t, test.searchNonce, token.Nonce)
			} else {
				// the negative and defective cases
				assert.Nil(t, token)
			}
		})
	}
}

func TestGetTokens(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupToken, err := createInitialSCIMToken(ctx, testClient, standardWarmupSCIMToken)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup token was not created.
		return
	}

	type testCase struct {
		name          string
		expectedNonce string
	}

	testCases := []testCase{
		{
			name:          "positive-" + createdWarmupToken.Metadata.ID,
			expectedNonce: createdWarmupToken.Nonce,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			tokens, err := testClient.client.SCIMTokens.GetTokens(ctx)

			assert.Nil(t, err)
			assert.Equal(t, 1, len(tokens))
			assert.Equal(t, test.expectedNonce, tokens[0].Nonce)
		})
	}
}

func TestCreateToken(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupNonce := uuid.New().String()

	type testCase struct {
		toCreate      *models.SCIMToken
		expectCreated *models.SCIMToken
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive full-token",
			toCreate: &models.SCIMToken{
				Nonce:     warmupNonce,
				CreatedBy: "some-admin@example.com",
			},
			expectCreated: &models.SCIMToken{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Nonce:     warmupNonce,
				CreatedBy: "some-admin@example.com",
			},
		},
		{
			name: "duplicate will fail",
			toCreate: &models.SCIMToken{
				Nonce:     warmupNonce,
				CreatedBy: "some-admin@example.com",
			},
			expectMsg: ptr.String("SCIM token already exists"),
		},
		{
			name: "defective nonce ID",
			toCreate: &models.SCIMToken{
				Nonce:     invalidID,
				CreatedBy: "some-admin@example.com",
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.SCIMTokens.CreateToken(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareSCIMTokens(t, test.expectCreated, actualCreated, false, timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestDeleteToken(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdWarmupToken, err := createInitialSCIMToken(ctx, testClient, standardWarmupSCIMToken)
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup token was not created.
		return
	}

	type testCase struct {
		toDelete  *models.SCIMToken
		expectMsg *string
		name      string
	}

	testCases := []testCase{
		{
			name: "positive-" + createdWarmupToken.Metadata.ID,
			toDelete: &models.SCIMToken{
				Metadata: models.ResourceMetadata{
					ID:      createdWarmupToken.Metadata.ID,
					Version: createdWarmupToken.Metadata.Version,
				},
			},
		},
		{
			name: "negative, non-existent ID",
			toDelete: &models.SCIMToken{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toDelete: &models.SCIMToken{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.SCIMTokens.DeleteToken(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

// Standard warmup SCIM tokens for tests in this module:
// The DB should AT MOST have only one token hence
// a slice is not needed here.
var standardWarmupSCIMToken = &models.SCIMToken{
	Nonce: uuid.New().String(),
}

// createInitialSCIMToken creates a warmup SCIM token for a test.
func createInitialSCIMToken(ctx context.Context, testClient *testClient,
	toCreate *models.SCIMToken) (*models.SCIMToken, error) {

	// At most one token will exist.
	created, err := testClient.client.SCIMTokens.CreateToken(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return created, nil
}

// Compare two SCIM token objects, including bounds for creation and updated times.
func compareSCIMTokens(t *testing.T, expected, actual *models.SCIMToken, checkID bool, times timeBounds) {
	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
	compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)

	assert.Equal(t, expected.Nonce, actual.Nonce)
}
