//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestUpdateResourceLimits(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	initialResourceLimits, err := testClient.client.ResourceLimits.GetResourceLimits(ctx)
	require.Nil(t, err)

	type testCase struct {
		input         *models.ResourceLimit
		expectMsg     *string
		expectUpdated *models.ResourceLimit
		name          string
	}

	/*
		template test case:

		{
		name        string
		input       *models.ResourceLimit
		expectMsg   *string
		expectResourceLimits []string
		}
	*/

	testCases := []testCase{}

	for _, preUpdate := range initialResourceLimits {
		now := currentTime()
		testCases = append(testCases, testCase{
			name: "positive-" + preUpdate.Name,
			input: &models.ResourceLimit{
				Metadata: models.ResourceMetadata{
					ID:      preUpdate.Metadata.ID,
					Version: preUpdate.Metadata.Version,
				},
				Name:  preUpdate.Name,
				Value: 43,
			},
			expectUpdated: &models.ResourceLimit{
				Metadata: models.ResourceMetadata{
					ID:                   preUpdate.Metadata.ID,
					Version:              preUpdate.Metadata.Version + 1,
					CreationTimestamp:    preUpdate.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				Name:  preUpdate.Name,
				Value: 43,
			},
		})
	}

	testCases = append(testCases, testCase{
		name: "negative, non-exist",
		input: &models.ResourceLimit{
			Metadata: models.ResourceMetadata{
				ID:      nonExistentID,
				Version: 1,
			},
		},
		expectMsg: resourceVersionMismatch,
	},

	// No invalid test case is applicable.

	)

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualUpdated, err := testClient.client.ResourceLimits.UpdateResourceLimit(ctx, test.input)

			checkError(t, test.expectMsg, err)

			if test.expectUpdated != nil {
				// the positive case
				require.NotNil(t, actualUpdated)
				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectUpdated.Metadata.CreationTimestamp
				now := currentTime()

				compareResourceLimits(t, test.expectUpdated, actualUpdated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualUpdated)
			}
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup resource limits for tests in this module:
var standardWarmupResourceLimits = []models.ResourceLimit{
	{
		Name:  "resource-limit-a",
		Value: 104,
	},
	{
		Name:  "resource-limit-b",
		Value: 204,
	},
	{
		Name:  "resource-limit-c",
		Value: 304,
	},
	{
		Name:  "resource-limit-99",
		Value: 994,
	},
}

// compareResourceLimits compares two resource limit objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareResourceLimits(t *testing.T, expected, actual *models.ResourceLimit,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Value, actual.Value)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
