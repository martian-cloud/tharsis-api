//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestGetMaintenanceMode(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	mode, err := testClient.client.MaintenanceModes.CreateMaintenanceMode(ctx, &models.MaintenanceMode{
		CreatedBy: "test-user",
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		expectModel     bool
	}

	testCases := []testCase{
		{
			name:        "get resource",
			expectModel: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualMode, err := testClient.client.MaintenanceModes.GetMaintenanceMode(ctx)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectModel {
				require.NotNil(t, actualMode)
				assert.Equal(t, mode.Metadata.ID, actualMode.Metadata.ID)
			} else {
				assert.Nil(t, actualMode)
			}
		})
	}
}

func TestCreateMaintenanceMode(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		createdBy       string
	}

	testCases := []testCase{
		{
			name:      "successfully create resource",
			createdBy: "test-user",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mode, err := testClient.client.MaintenanceModes.CreateMaintenanceMode(ctx, &models.MaintenanceMode{
				CreatedBy: test.createdBy,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, mode)
		})
	}
}

func TestDeleteMaintenanceMode(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	mode, err := testClient.client.MaintenanceModes.CreateMaintenanceMode(ctx, &models.MaintenanceMode{
		CreatedBy: "test-user",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              mode.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      mode.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.MaintenanceModes.DeleteMaintenanceMode(ctx, &models.MaintenanceMode{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
		})
	}
}
