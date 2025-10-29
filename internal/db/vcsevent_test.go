//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for VCSEventSortableField
func (v VCSEventSortableField) getValue() string {
	return string(v)
}

func TestVCSEvents_CreateVCSEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-event",
		Description: "test group for vcs event",
		FullPath:    "test-group-vcs-event",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-event",
		Description:    "test workspace for vcs event",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		event           *models.VCSEvent
	}

	commitID := "abc123def456"
	sourceBranch := "main"

	testCases := []testCase{
		{
			name: "successfully create vcs event",
			event: &models.VCSEvent{
				WorkspaceID:         workspace.Metadata.ID,
				CommitID:            &commitID,
				SourceReferenceName: &sourceBranch,
				RepositoryURL:       "https://github.com/test/repo",
				Type:                models.BranchEventType,
				Status:              models.VCSEventPending,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			event, err := testClient.client.VCSEvents.CreateEvent(ctx, test.event)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, event)
			assert.Equal(t, test.event.WorkspaceID, event.WorkspaceID)
			assert.Equal(t, test.event.CommitID, event.CommitID)
			assert.Equal(t, test.event.Type, event.Type)
		})
	}
}

func TestVCSEvents_UpdateVCSEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-event-update",
		Description: "test group for vcs event update",
		FullPath:    "test-group-vcs-event-update",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-event-update",
		Description:    "test workspace for vcs event update",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	// Create a VCS event to update
	commitID := "abc123def456"
	sourceBranch := "main"
	createdEvent, err := testClient.client.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
		WorkspaceID:         workspace.Metadata.ID,
		CommitID:            &commitID,
		SourceReferenceName: &sourceBranch,
		RepositoryURL:       "https://github.com/test/repo",
		Type:                models.BranchEventType,
		Status:              models.VCSEventPending,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		updateEvent     *models.VCSEvent
	}

	errorMessage := "Test error message"

	testCases := []testCase{
		{
			name: "successfully update vcs event",
			updateEvent: &models.VCSEvent{
				Metadata:            createdEvent.Metadata,
				WorkspaceID:         createdEvent.WorkspaceID,
				CommitID:            createdEvent.CommitID,
				SourceReferenceName: createdEvent.SourceReferenceName,
				RepositoryURL:       createdEvent.RepositoryURL,
				Type:                createdEvent.Type,
				Status:              models.VCSEventErrored,
				ErrorMessage:        &errorMessage,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			event, err := testClient.client.VCSEvents.UpdateEvent(ctx, test.updateEvent)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, event)
			assert.Equal(t, test.updateEvent.Status, event.Status)
			assert.Equal(t, test.updateEvent.ErrorMessage, event.ErrorMessage)
		})
	}
}

func TestVCSEvents_GetEventByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-event-get-by-id",
		Description: "test group for vcs event get by id",
		FullPath:    "test-group-vcs-event-get-by-id",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-event-get-by-id",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS event for testing
	createdVCSEvent, err := testClient.client.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.BranchEventType,
		Status:      models.VCSEventPending,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectVCSEvent  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by id",
			id:             createdVCSEvent.Metadata.ID,
			expectVCSEvent: true,
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
			vcsEvent, err := testClient.client.VCSEvents.GetEventByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVCSEvent {
				require.NotNil(t, vcsEvent)
				assert.Equal(t, test.id, vcsEvent.Metadata.ID)
			} else {
				assert.Nil(t, vcsEvent)
			}
		})
	}
}

func TestVCSEvents_GetEvents(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-events-list",
		Description: "test group for vcs events list",
		FullPath:    "test-group-vcs-events-list",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-events-list",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test VCS events
	vcsEvents := []models.VCSEvent{
		{
			WorkspaceID: workspace.Metadata.ID,
			Type:        models.BranchEventType,
			Status:      models.VCSEventPending,
		},
		{
			WorkspaceID: workspace.Metadata.ID,
			Type:        models.MergeRequestEventType,
			Status:      models.VCSEventFinished,
		},
	}

	createdVCSEvents := []models.VCSEvent{}
	for _, vcsEvent := range vcsEvents {
		created, err := testClient.client.VCSEvents.CreateEvent(ctx, &vcsEvent)
		require.NoError(t, err)
		createdVCSEvents = append(createdVCSEvents, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetVCSEventsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all vcs events",
			input:       &GetVCSEventsInput{},
			expectCount: len(createdVCSEvents),
		},
		{
			name: "filter by workspace ID",
			input: &GetVCSEventsInput{
				Filter: &VCSEventFilter{
					WorkspaceID: &workspace.Metadata.ID,
				},
			},
			expectCount: len(createdVCSEvents),
		},
		{
			name: "filter by VCS event IDs",
			input: &GetVCSEventsInput{
				Filter: &VCSEventFilter{
					VCSEventIDs: []string{createdVCSEvents[0].Metadata.ID},
				},
			},
			expectCount: 1,
		}}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.VCSEvents.GetEvents(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.VCSEvents, test.expectCount)
		})
	}
}

func TestVCSEvents_GetEventsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-events-pagination",
		Description: "test group for vcs events pagination",
		FullPath:    "test-group-vcs-events-pagination",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-events-pagination",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
			WorkspaceID: workspace.Metadata.ID,
			Type:        models.BranchEventType,
			Status:      models.VCSEventPending,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		VCSEventSortableFieldCreatedAtAsc,
		VCSEventSortableFieldCreatedAtDesc,
		VCSEventSortableFieldUpdatedAtAsc,
		VCSEventSortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := VCSEventSortableField(sortByField.getValue())

		result, err := testClient.client.VCSEvents.GetEvents(ctx, &GetVCSEventsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.VCSEvents {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestVCSEvents_GetEventByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a workspace for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-event-get-by-trn",
		Description: "test group for vcs event get by trn",
		FullPath:    "test-group-vcs-event-get-by-trn",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-event-get-by-trn",
		GroupID:        group.Metadata.ID,
		MaxJobDuration: ptr.Int32(1),
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS event for testing
	createdVCSEvent, err := testClient.client.VCSEvents.CreateEvent(ctx, &models.VCSEvent{
		WorkspaceID: workspace.Metadata.ID,
		Type:        models.BranchEventType,
		Status:      models.VCSEventPending,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectVCSEvent  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by TRN",
			trn:            createdVCSEvent.Metadata.TRN,
			expectVCSEvent: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:vcs_event:non-existent-id",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "invalid-trn",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			vcsEvent, err := testClient.client.VCSEvents.GetEventByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVCSEvent {
				require.NotNil(t, vcsEvent)
				assert.Equal(t, createdVCSEvent.Metadata.ID, vcsEvent.Metadata.ID)
			} else {
				assert.Nil(t, vcsEvent)
			}
		})
	}
}
