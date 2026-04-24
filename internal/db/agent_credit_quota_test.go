//go:build integration

package db

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestAgentCreditQuotas_CreateAgentCreditQuota(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-credit-create",
		Email:    "credit-create@example.com",
	})
	require.Nil(t, err)

	monthDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		userID          string
	}

	testCases := []testCase{
		{
			name:   "create credit quota",
			userID: user.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			quota, err := testClient.client.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
				UserID:       test.userID,
				MonthDate:    monthDate,
				TotalCredits: 0,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, quota)
			assert.Equal(t, test.userID, quota.UserID)
			assert.Equal(t, float64(0), quota.TotalCredits)
			assert.NotEmpty(t, quota.Metadata.ID)
		})
	}
}

func TestAgentCreditQuotas_CreateDuplicate(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-credit-dup",
		Email:    "credit-dup@example.com",
	})
	require.Nil(t, err)

	monthDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	_, err = testClient.client.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
		UserID:    user.Metadata.ID,
		MonthDate: monthDate,
	})
	require.Nil(t, err)

	// Creating a duplicate should fail with conflict
	_, err = testClient.client.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
		UserID:    user.Metadata.ID,
		MonthDate: monthDate,
	})
	assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
}

func TestAgentCreditQuotas_GetAgentCreditQuota(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-credit-get",
		Email:    "credit-get@example.com",
	})
	require.Nil(t, err)

	monthDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	created, err := testClient.client.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
		UserID:       user.Metadata.ID,
		MonthDate:    monthDate,
		TotalCredits: 10.5,
	})
	require.Nil(t, err)

	type testCase struct {
		name        string
		userID      string
		monthDate   time.Time
		expectQuota bool
	}

	testCases := []testCase{
		{
			name:        "get existing quota",
			userID:      user.Metadata.ID,
			monthDate:   monthDate,
			expectQuota: true,
		},
		{
			name:      "quota not found for different month",
			userID:    user.Metadata.ID,
			monthDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "quota not found for different user",
			userID:    nonExistentID,
			monthDate: monthDate,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			quota, err := testClient.client.AgentCreditQuotas.GetAgentCreditQuota(ctx, test.userID, test.monthDate)
			require.Nil(t, err)

			if test.expectQuota {
				require.NotNil(t, quota)
				assert.Equal(t, created.Metadata.ID, quota.Metadata.ID)
				assert.Equal(t, 10.5, quota.TotalCredits)
			} else {
				assert.Nil(t, quota)
			}
		})
	}
}

func TestAgentCreditQuotas_AddCredits(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-credit-add",
		Email:    "credit-add@example.com",
	})
	require.Nil(t, err)

	monthDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	created, err := testClient.client.AgentCreditQuotas.CreateAgentCreditQuota(ctx, &models.AgentCreditQuota{
		UserID:       user.Metadata.ID,
		MonthDate:    monthDate,
		TotalCredits: 5.0,
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		credits         float64
	}

	testCases := []testCase{
		{
			name:    "add credits",
			id:      created.Metadata.ID,
			credits: 3.5,
		},
		{
			name:            "add credits to non-existent quota",
			id:              nonExistentID,
			credits:         1.0,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.AgentCreditQuotas.AddCredits(ctx, test.id, test.credits)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			// Verify credits were added
			quota, err := testClient.client.AgentCreditQuotas.GetAgentCreditQuota(ctx, user.Metadata.ID, monthDate)
			require.Nil(t, err)
			require.NotNil(t, quota)
			assert.Equal(t, 8.5, quota.TotalCredits)
		})
	}
}
