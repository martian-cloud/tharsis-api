//go:build integration

package db

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
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

func TestGetWorkspaceAssessmentByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	workspaceAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name                      string
		trn                       string
		expectWorkspaceAssessment bool
		expectErrorCode           errors.CodeType
	}

	testCases := []testCase{
		{
			name:                      "get resource by TRN",
			trn:                       workspaceAssessment.Metadata.TRN,
			expectWorkspaceAssessment: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeWorkspaceAssessment.Build(workspace.FullPath, nonExistentGlobalID),
		},
		{
			name:            "assessment TRN cannot have less than two parts",
			trn:             trn.TypeWorkspaceAssessment.Build(nonExistentGlobalID),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualWorkspaceAssessment, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessmentByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectWorkspaceAssessment {
				require.NotNil(t, actualWorkspaceAssessment)
				assert.Equal(t, trn.TypeWorkspaceAssessment.Build(workspace.FullPath, workspaceAssessment.GetGlobalID()), actualWorkspaceAssessment.Metadata.TRN)
			} else {
				assert.Nil(t, actualWorkspaceAssessment)
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

func TestGetWorkspaceAssessmentsInProgressExcludesStale(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.Nil(t, err)

	freshWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "fresh-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	staleWorkspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "stale-workspace",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
	})
	require.Nil(t, err)

	currentTime := time.Now().UTC()

	// Fresh in-progress assessment: no completed_at, updated_at defaults to now.
	_, err = testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        freshWorkspace.Metadata.ID,
		StartedAtTimestamp: currentTime,
	})
	require.Nil(t, err)

	staleAssessment, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        staleWorkspace.Metadata.ID,
		StartedAtTimestamp: currentTime,
	})
	require.Nil(t, err)

	// Backdate the stale assessment's updated_at beyond the stale timeout, simulating a
	// run that was abandoned before completing.
	staleTimestamp := currentTime.Add(-2 * models.AssessmentStaleTimeout)
	_, err = testClient.client.getConnection(ctx).Exec(ctx,
		"UPDATE workspace_assessments SET updated_at = $1 WHERE id = $2",
		staleTimestamp, staleAssessment.Metadata.ID)
	require.Nil(t, err)

	t.Run("in progress excludes the stale assessment", func(t *testing.T) {
		result, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &GetWorkspaceAssessmentsInput{
			Filter: &WorkspaceAssessmentFilter{InProgress: ptr.Bool(true)},
		})
		require.Nil(t, err)
		require.Len(t, result.WorkspaceAssessments, 1)
		assert.Equal(t, freshWorkspace.Metadata.ID, result.WorkspaceAssessments[0].WorkspaceID)
	})

	t.Run("not in progress includes the stale assessment", func(t *testing.T) {
		result, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessments(ctx, &GetWorkspaceAssessmentsInput{
			Filter: &WorkspaceAssessmentFilter{InProgress: ptr.Bool(false)},
		})
		require.Nil(t, err)
		require.Len(t, result.WorkspaceAssessments, 1)
		assert.Equal(t, staleWorkspace.Metadata.ID, result.WorkspaceAssessments[0].WorkspaceID)
	})
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

// TestWorkspaceAssessmentErrorMessageRoundTrip covers the error_message column through create,
// read and update. The column is appended to workspaceAssessmentsFieldList and scanned
// positionally, so a mismatch between the two silently mis-scans every read; asserting the
// neighbouring fields survive intact is what catches that.
func TestWorkspaceAssessmentErrorMessageRoundTrip(t *testing.T) {
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

	startedAt := time.Now().UTC()

	// Created without a message: the last attempt did not fail.
	created, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID:        workspace.Metadata.ID,
		StartedAtTimestamp: startedAt,
	})
	require.Nil(t, err)
	require.NotNil(t, created)
	assert.Nil(t, created.ErrorMessage, "a new assessment has no recorded failure")

	// Record a failure, as the scheduler does when it cannot create the assessment run. The
	// message is truncated by the model helper first, which is what the scheduler does.
	completedAt := time.Now().UTC()
	longMessage := models.TruncateAssessmentErrorMessage(
		"token in environment variable TF_TOKEN_example_com is not authorized to access this module: " +
			strings.Repeat("x", models.MaxAssessmentErrorMessageLength),
	)

	created.ErrorMessage = &longMessage
	created.CompletedAtTimestamp = &completedAt
	created.HasDrift = true

	updated, err := testClient.client.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, created)
	require.Nil(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, updated.ErrorMessage)
	assert.Equal(t, longMessage, *updated.ErrorMessage)
	assert.LessOrEqual(t, len(*updated.ErrorMessage), models.MaxAssessmentErrorMessageLength)
	// Neighbouring columns must survive, proving the positional scan stayed aligned.
	assert.True(t, updated.HasDrift)
	assert.Equal(t, workspace.Metadata.ID, updated.WorkspaceID)
	assert.Equal(t, startedAt.Format(time.RFC3339), updated.StartedAtTimestamp.Format(time.RFC3339))
	require.NotNil(t, updated.CompletedAtTimestamp)
	assert.Equal(t, completedAt.Format(time.RFC3339), updated.CompletedAtTimestamp.Format(time.RFC3339))

	// A separate read goes through getSelectFields/scanWorkspaceAssessment too.
	fetched, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, fetched)
	require.NotNil(t, fetched.ErrorMessage)
	assert.Equal(t, longMessage, *fetched.ErrorMessage)
	assert.True(t, fetched.HasDrift)

	// Cleared when an assessment run is created again.
	fetched.ErrorMessage = nil
	cleared, err := testClient.client.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, fetched)
	require.Nil(t, err)
	require.NotNil(t, cleared)
	assert.Nil(t, cleared.ErrorMessage, "clearing the message must write a NULL, not an empty string")
	assert.True(t, cleared.HasDrift, "clearing the failure must not disturb the drift verdict")
}

// TestWorkspaceAssessmentBackoffRoundTrip covers the available_at and consecutive_failures
// columns the scheduler uses to back off a workspace that keeps failing. Like error_message,
// both are appended to workspaceAssessmentsFieldList and scanned positionally, so this also
// guards against the scan silently drifting out of alignment.
func TestWorkspaceAssessmentBackoffRoundTrip(t *testing.T) {
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

	// Created without backoff state: the last attempt did not fail.
	created, err := testClient.client.WorkspaceAssessments.CreateWorkspaceAssessment(ctx, &models.WorkspaceAssessment{
		WorkspaceID: workspace.Metadata.ID,
	})
	require.Nil(t, err)
	require.NotNil(t, created)
	assert.Nil(t, created.AvailableAtTimestamp)
	assert.Equal(t, 0, created.ConsecutiveFailures)

	// Record a failure, as the scheduler does: a future available_at suppresses eligibility,
	// and the failure count drives how far out it was pushed.
	availableAt := time.Now().UTC().Add(4 * time.Hour)
	created.AvailableAtTimestamp = &availableAt
	created.ConsecutiveFailures = 3

	updated, err := testClient.client.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, created)
	require.Nil(t, err)
	require.NotNil(t, updated)
	require.NotNil(t, updated.AvailableAtTimestamp)
	assert.Equal(t, availableAt.Format(time.RFC3339), updated.AvailableAtTimestamp.Format(time.RFC3339))
	assert.Equal(t, 3, updated.ConsecutiveFailures)

	// A separate read goes through getSelectFields/scanWorkspaceAssessment too.
	fetched, err := testClient.client.WorkspaceAssessments.GetWorkspaceAssessmentByWorkspaceID(ctx, workspace.Metadata.ID)
	require.Nil(t, err)
	require.NotNil(t, fetched)
	require.NotNil(t, fetched.AvailableAtTimestamp)
	assert.Equal(t, availableAt.Format(time.RFC3339), fetched.AvailableAtTimestamp.Format(time.RFC3339))
	assert.Equal(t, 3, fetched.ConsecutiveFailures)

	// Cleared when an assessment run is created again.
	fetched.AvailableAtTimestamp = nil
	fetched.ConsecutiveFailures = 0
	cleared, err := testClient.client.WorkspaceAssessments.UpdateWorkspaceAssessment(ctx, fetched)
	require.Nil(t, err)
	require.NotNil(t, cleared)
	assert.Nil(t, cleared.AvailableAtTimestamp, "clearing must write a NULL, not a zero timestamp")
	assert.Equal(t, 0, cleared.ConsecutiveFailures)
}
