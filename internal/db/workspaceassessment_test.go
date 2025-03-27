//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func (sf WorkspaceAssessmentSortableField) getValue() string {
	return string(sf)
}

func TestGetWorkspaceAssessmentByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode           errors.CodeType
		name                      string
		id                        string
		expectWorkspaceAssessment bool
	}

	testCases := []testCase{
		{
			name:                      "get resource by id",
			id:                        workspaceAssessment.Metadata.ID,
			expectWorkspaceAssessment: true,
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
			workspaceAssessment, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessmentByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectWorkspaceAssessment {
				require.NotNil(t, workspaceAssessment)
				assert.Equal(t, test.id, workspaceAssessment.Metadata.ID)
			} else {
				assert.Nil(t, workspaceAssessment)
			}
		})
	}
}

func TestGetWorkspaceAssessmentByWorkspaceID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.Nil(t, err)

	type testCase struct {
		expectErrorCode           errors.CodeType
		name                      string
		id                        string
		expectWorkspaceAssessment bool
	}

	testCases := []testCase{
		{
			name:                      "get resource by id",
			id:                        workspaceAssessment.WorkspaceID,
			expectWorkspaceAssessment: true,
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
			workspaceAssessment, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectWorkspaceAssessment {
				require.NotNil(t, workspaceAssessment)
				assert.Equal(t, test.id, workspaceAssessment.WorkspaceID)
			} else {
				assert.Nil(t, workspaceAssessment)
			}
		})
	}
}

func TestCreateWorkspaceAssessment(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
	}

	testCases := []testCase{
		{
			name:        "successfully create resource",
			workspaceID: workspace.Metadata.ID,
		},
		{
			name:            "create will fail because assessment does not exist",
			workspaceID:     nonExistentID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
				WorkspaceID: test.workspaceID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, workspaceAssessment)
		})
	}
}

func TestUpdateWorkspaceAssessment(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.Nil(t, err)

	currentTime := time.Now().UTC()

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
	}

	testCases := []testCase{
		{
			name:    "successfully update resource",
			version: 1,
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualWorkspaceAssessment, err := testClient.client.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
				Metadata: models.ResourceMetadata{
					ID:      workspaceAssessment.Metadata.ID,
					Version: test.version,
				},
				CompletedAtTimestamp: &currentTime,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)
			require.NotNil(t, actualWorkspaceAssessment)
			assert.Equal(t, currentTime.Format(time.RFC3339), actualWorkspaceAssessment.CompletedAtTimestamp.Format(time.RFC3339))
		})
	}
}

func TestDeleteWorkspaceAssessment(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
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
			id:              workspaceAssessment.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      workspaceAssessment.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.WorkspaceAssessments.DeleteWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
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

func TestGetWorkspaceAssessments(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-1",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	workspace2, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-2",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	currentTime := time.Now().UTC()

	assessments := []*models.WorkspaceAssessment{}

	assessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:          workspace1.Metadata.ID,
		StartedAtTimestamp:   currentTime,
		CompletedAtTimestamp: &currentTime,
	})
	require.Nil(t, err)

	assessments = append(assessments, assessment)

	assessment, err = testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        workspace2.Metadata.ID,
		StartedAtTimestamp: currentTime,
	})
	require.Nil(t, err)

	assessments = append(assessments, assessment)

	type testCase struct {
		filter            *WorkspaceAssessmentFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name: "return all assessments for workspace 1",
			filter: &WorkspaceAssessmentFilter{
				WorkspaceIDs: []string{workspace1.Metadata.ID},
			},
			expectResultCount: len(assessments) - 1,
		},
		{
			name: "return all assessments that are in progress",
			filter: &WorkspaceAssessmentFilter{
				InProgress: ptr.Bool(true),
			},
			expectResultCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &GetWorkspaceAssessmentsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.Nil(t, err)

			assert.Equal(t, test.expectResultCount, len(result.WorkspaceAssessments))
		})
	}
}

func TestGetWorkspaceAssessmentsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
			Name:           fmt.Sprintf("ws-%d", i),
			GroupID:        group.Metadata.ID,
			MaxJobDuration: ptr.Int32(1),
		})
		require.Nil(t, err)

		_, err = testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
			WorkspaceID: workspace.Metadata.ID,
		})
		require.Nil(t, err)
	}

	sortableFields := []sortableField{
		WorkspaceAssessmentSortableFieldUpdatedAtAsc,
		WorkspaceAssessmentSortableFieldUpdatedAtDesc,
		WorkspaceAssessmentSortableFieldStartedAtAsc,
		WorkspaceAssessmentSortableFieldStartedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := WorkspaceAssessmentSortableField(sortByField.getValue())

		result, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &GetWorkspaceAssessmentsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.WorkspaceAssessments {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
