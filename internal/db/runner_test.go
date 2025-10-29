//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for RunnerSortableField
func (r RunnerSortableField) getValue() string {
	return string(r)
}

func TestRunners_CreateRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runner",
		Description: "test group for runner",
		FullPath:    "test-group-runner",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		runnerName      string
		groupID         string
		runnerType      models.RunnerType
	}

	testCases := []testCase{
		{
			name:       "create runner",
			runnerName: "test-runner",
			groupID:    group.Metadata.ID,
			runnerType: models.GroupRunnerType,
		},
		{
			name:            "create runner with invalid group ID",
			runnerName:      "invalid-runner",
			groupID:         invalidID,
			runnerType:      models.GroupRunnerType,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
				Name:      test.runnerName,
				GroupID:   &test.groupID,
				Type:      test.runnerType,
				CreatedBy: "db-integration-tests",
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, runner)

			assert.Equal(t, test.runnerName, runner.Name)
			assert.Equal(t, test.groupID, *runner.GroupID)
			assert.Equal(t, test.runnerType, runner.Type)
			assert.NotEmpty(t, runner.Metadata.ID)
		})
	}
}

func TestRunners_UpdateRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and runner for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runner-update",
		Description: "test group for runner update",
		FullPath:    "test-group-runner-update",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-update",
		GroupID:     &group.Metadata.ID,
		Type:        models.GroupRunnerType,
		Description: "original description",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		description     string
	}

	testCases := []testCase{
		{
			name:        "update runner",
			version:     createdRunner.Metadata.Version,
			description: "updated description",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			description:     "should not update",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runnerToUpdate := *createdRunner
			runnerToUpdate.Metadata.Version = test.version
			runnerToUpdate.Description = test.description

			updatedRunner, err := testClient.client.Runners.UpdateRunner(ctx, &runnerToUpdate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, updatedRunner)

			assert.Equal(t, test.description, updatedRunner.Description)
			assert.Equal(t, createdRunner.Metadata.Version+1, updatedRunner.Metadata.Version)
		})
	}
}

func TestRunners_DeleteRunner(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group and runner for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runner-delete",
		Description: "test group for runner delete",
		FullPath:    "test-group-runner-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.Nil(t, err)

	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-delete",
		GroupID:     &group.Metadata.ID,
		Type:        models.GroupRunnerType,
		Description: "runner to delete",
		CreatedBy:   "db-integration-tests",
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
			name:    "delete runner",
			id:      createdRunner.Metadata.ID,
			version: createdRunner.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdRunner.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Runners.DeleteRunner(ctx, &models.Runner{
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

			// Verify runner was deleted
			runner, err := testClient.client.Runners.GetRunnerByID(ctx, test.id)
			assert.Nil(t, runner)
			assert.Nil(t, err)
		})
	}
}
func TestRunners_GetRunnerByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runner-get-by-id",
		Description: "test group for runner get by id",
		FullPath:    "test-group-runner-get-by-id",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a runner for testing
	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-get-by-id",
		Description: "test runner for get by id",
		GroupID:     &group.Metadata.ID,
		Type:        models.SharedRunnerType,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectRunner    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by id",
			id:           createdRunner.Metadata.ID,
			expectRunner: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runner, err := testClient.client.Runners.GetRunnerByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRunner {
				require.NotNil(t, runner)
				assert.Equal(t, test.id, runner.Metadata.ID)
			} else {
				assert.Nil(t, runner)
			}
		})
	}
}

func TestRunners_GetRunners(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runners-list",
		Description: "test group for runners list",
		FullPath:    "test-group-runners-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test runners
	runners := []models.Runner{
		{
			Name:        "test-runner-1",
			Description: "test runner 1",
			GroupID:     &group.Metadata.ID,
			Type:        models.SharedRunnerType,
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-runner-2",
			Description: "test runner 2",
			GroupID:     &group.Metadata.ID,
			Type:        models.GroupRunnerType,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdRunners := []models.Runner{}
	for _, runner := range runners {
		created, err := testClient.client.Runners.CreateRunner(ctx, &runner)
		require.NoError(t, err)
		createdRunners = append(createdRunners, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetRunnersInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all runners",
			input:       &GetRunnersInput{},
			expectCount: len(createdRunners),
		},
		{
			name: "filter by group ID",
			input: &GetRunnersInput{
				Filter: &RunnerFilter{
					GroupID: &group.Metadata.ID,
				},
			},
			expectCount: len(createdRunners),
		},
		{
			name: "filter by runner name",
			input: &GetRunnersInput{
				Filter: &RunnerFilter{
					RunnerName: &createdRunners[0].Name,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by runner type",
			input: &GetRunnersInput{
				Filter: &RunnerFilter{
					RunnerType: &createdRunners[0].Type,
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by runner IDs",
			input: &GetRunnersInput{
				Filter: &RunnerFilter{
					RunnerIDs: []string{createdRunners[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by namespace paths",
			input: &GetRunnersInput{
				Filter: &RunnerFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdRunners),
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Runners.GetRunners(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.Runners, test.expectCount)
		})
	}
}

func TestRunners_GetRunnersWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runners-pagination",
		Description: "test group for runners pagination",
		FullPath:    "test-group-runners-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
			Name:        fmt.Sprintf("test-runner-%d", i),
			Description: fmt.Sprintf("test runner %d", i),
			GroupID:     &group.Metadata.ID,
			Type:        models.SharedRunnerType,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	// Only test UpdatedAt fields to avoid GROUP_LEVEL complexity
	sortableFields := []sortableField{
		RunnerSortableFieldUpdatedAtAsc,
		RunnerSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := RunnerSortableField(sortByField.getValue())

		result, err := testClient.client.Runners.GetRunners(ctx, &GetRunnersInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Runners {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestRunners_GetRunnerByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-runner-get-by-trn",
		Description: "test group for runner get by trn",
		FullPath:    "test-group-runner-get-by-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a runner for testing
	createdRunner, err := testClient.client.Runners.CreateRunner(ctx, &models.Runner{
		Name:        "test-runner-get-by-trn",
		Description: "test runner for get by trn",
		GroupID:     &group.Metadata.ID,
		Type:        models.SharedRunnerType,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectRunner    bool
	}

	testCases := []testCase{
		{
			name:         "get resource by TRN",
			trn:          createdRunner.Metadata.TRN,
			expectRunner: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:runner:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			runner, err := testClient.client.Runners.GetRunnerByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectRunner {
				require.NotNil(t, runner)
				assert.Equal(t, createdRunner.Metadata.ID, runner.Metadata.ID)
			} else {
				assert.Nil(t, runner)
			}
		})
	}
}
