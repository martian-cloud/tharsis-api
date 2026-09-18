//go:build integration

package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for ActivityEventSortableField
func (ae ActivityEventSortableField) getValue() string {
	return string(ae)
}

func TestActivityEvents_CreateActivityEvent(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for the activity event
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-create-activity-event",
		Email:    "test-create-activity-event@example.com",
	})
	require.NoError(t, err)

	// Create a group for the namespace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-create-activity-event",
		Description: "test group for create activity event",
		FullPath:    "test-group-create-activity-event",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		activityEvent   *models.ActivityEvent
	}

	testCases := []testCase{
		{
			name: "create activity event",
			activityEvent: &models.ActivityEvent{
				UserID:        &user.Metadata.ID,
				NamespacePath: &group.FullPath,
				Action:        models.ActionCreate,
				TargetType:    models.TargetGroup,
				TargetID:      group.Metadata.ID,
			},
		},
		{
			name: "create activity event with invalid target ID",
			activityEvent: &models.ActivityEvent{
				UserID:        &user.Metadata.ID,
				NamespacePath: &group.FullPath,
				Action:        models.ActionCreate,
				TargetType:    models.TargetGroup,
				TargetID:      invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			activityEvent, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, test.activityEvent)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, activityEvent)
			assert.Equal(t, test.activityEvent.Action, activityEvent.Action)
			assert.Equal(t, test.activityEvent.TargetType, activityEvent.TargetType)
			assert.Equal(t, test.activityEvent.TargetID, activityEvent.TargetID)
			assert.NotEmpty(t, activityEvent.Metadata.ID)
		})
	}
}

// TestActivityEvents_WorkspaceRoleBindingTarget covers workspace_role_binding_target_id's ON DELETE
// CASCADE behavior: a CREATE or UPDATE event that targets a binding is removed along with the
// binding's own row once it is deleted, the same as every other activity event target type. Removing
// a binding is recorded as a DeleteChildResource event against the WORKSPACE instead (see
// removeWorkspaceRoleBinding), never against the binding's own row, so there is no event left
// referencing this column by the time the binding is gone for CASCADE to strand.
func TestActivityEvents_WorkspaceRoleBindingTarget(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-wrb-activity-event",
		Email:    "test-wrb-activity-event@example.com",
	})
	require.NoError(t, err)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-wrb-activity-event",
		Description: "test group for workspace role binding activity event",
		FullPath:    "test-group-wrb-activity-event",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	duration := int32(forTestMaxJobDuration.Minutes())
	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:             "test-workspace-wrb-activity-event",
		GroupID:          group.Metadata.ID,
		CreatedBy:        "db-integration-tests",
		MaxJobDuration:   &duration,
		TerraformVersion: "1.5.0",
	})
	require.NoError(t, err)

	role := createTestRoleForBinding(ctx, t, testClient, "test-role-wrb-activity-event",
		[]models.Permission{{Action: "read", ResourceType: "workspace"}})

	binding, err := testClient.client.WorkspaceRoleBindings.CreateWorkspaceRoleBinding(ctx, &models.WorkspaceRoleBinding{
		WorkspaceID: workspace.Metadata.ID,
		RoleID:      role.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	createEvent, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &models.ActivityEvent{
		UserID:        &user.Metadata.ID,
		NamespacePath: &group.FullPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetWorkspaceRoleBinding,
		TargetID:      binding.Metadata.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, binding.Metadata.ID, createEvent.TargetID)

	updateEvent, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &models.ActivityEvent{
		UserID:        &user.Metadata.ID,
		NamespacePath: &group.FullPath,
		Action:        models.ActionUpdate,
		TargetType:    models.TargetWorkspaceRoleBinding,
		TargetID:      binding.Metadata.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, binding.Metadata.ID, updateEvent.TargetID)

	// Deleting the binding must cascade to both events referencing it -- they no longer exist at
	// all, rather than surviving with a nulled-out target.
	require.NoError(t, testClient.client.WorkspaceRoleBindings.DeleteWorkspaceRoleBinding(ctx, binding))

	result, err := testClient.client.ActivityEvents.GetActivityEvents(ctx, &GetActivityEventsInput{
		Filter: &ActivityEventFilter{
			ActivityEventIDs: []string{createEvent.Metadata.ID, updateEvent.Metadata.ID},
		},
	})
	require.NoError(t, err)
	assert.Empty(t, result.ActivityEvents, "both events referencing the deleted binding must be cascaded away, not survive with a null target")
}

// TestActivityEvents_CreateActivityEvent_ForeignKeyViolationMessages locks down that a foreign key
// violation on each target-id column is translated into a specific, ENotFound "X does not exist"
// error, not just the generic wrapped DB error every other failure gets. This matters because the
// switch in CreateActivityEvent matches on the FK constraint's name as a Go string literal, and that
// name depends on how the column's REFERENCES clause was declared in its migration -- a column added
// via "ADD COLUMN ... REFERENCES x(id)" gets Postgres's auto-generated name
// ("<table>_<column>_fkey"), while a column declared with an explicit "CONSTRAINT fk_... FOREIGN KEY"
// gets that literal name instead. Getting this wrong doesn't fail to compile or fail any test that
// only exercises the golden path -- the switch's case silently never matches, and the violation falls
// through to the generic error path instead of the intended message. That is exactly what had
// happened to the cleanup_policy_target_id case before this test was added.
func TestActivityEvents_CreateActivityEvent_ForeignKeyViolationMessages(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-fk-violation-messages",
		Email:    "test-fk-violation-messages@example.com",
	})
	require.NoError(t, err)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-fk-violation-messages",
		Description: "test group for FK violation message coverage",
		FullPath:    "test-group-fk-violation-messages",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name          string
		targetType    models.ActivityEventTargetType
		expectMessage string
	}

	testCases := []testCase{
		{
			// A control case: this target type's FK was declared with an explicit CONSTRAINT name
			// in the init migration, and was already correctly matched before this test existed.
			// Included so a future refactor that breaks the matching for every target type at once
			// (e.g. an unrelated change to how FK violations are detected) shows up here too, not
			// only in the two cases that were previously broken or newly added.
			name:          "group (explicit CONSTRAINT name, already correct)",
			targetType:    models.TargetGroup,
			expectMessage: "group does not exist",
		},
		{
			name:          "cleanup policy (auto-generated FK name; previously unmatched, silently fell through to the generic error)",
			targetType:    models.TargetCleanupPolicy,
			expectMessage: "cleanup policy does not exist",
		},
		{
			name:          "workspace role binding (auto-generated FK name)",
			targetType:    models.TargetWorkspaceRoleBinding,
			expectMessage: "workspace role binding does not exist",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &models.ActivityEvent{
				UserID:        &user.Metadata.ID,
				NamespacePath: &group.FullPath,
				Action:        models.ActionCreate,
				TargetType:    test.targetType,
				// A well-formed but nonexistent UUID: this must reach the FK constraint check (not
				// a syntax error, which invalidID exercises and which fails before the FK check).
				TargetID: nonExistentID,
			})

			require.Error(t, err)
			assert.Equal(t, errors.ENotFound, errors.ErrorCode(err))
			assert.Contains(t, err.Error(), test.expectMessage)
		})
	}
}

func TestActivityEvents_GetActivityEvents(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for the activity events
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-activity-events",
		Email:    "test-activity-events@example.com",
	})
	require.NoError(t, err)

	// Create a group for the namespace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-activity-events-list",
		Description: "test group for activity events list",
		FullPath:    "test-group-activity-events-list",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test activity events
	activityEvents := []models.ActivityEvent{
		{
			UserID:        &user.Metadata.ID,
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		},
		{
			UserID:        &user.Metadata.ID,
			NamespacePath: &group.FullPath,
			Action:        models.ActionUpdate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		},
	}

	createdActivityEvents := []models.ActivityEvent{}
	for _, activityEvent := range activityEvents {
		created, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &activityEvent)
		require.NoError(t, err)
		createdActivityEvents = append(createdActivityEvents, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetActivityEventsInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all activity events",
			input:       &GetActivityEventsInput{},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by user ID",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					UserID: &user.Metadata.ID,
				},
			},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by namespace path",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					NamespacePath: &group.FullPath,
				},
			},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by activity event IDs",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					ActivityEventIDs: []string{createdActivityEvents[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by actions",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					Actions: []models.ActivityEventAction{models.ActionCreate},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by target types",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					TargetTypes: []models.ActivityEventTargetType{models.TargetGroup},
				},
			},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by user ID",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					UserID: &user.Metadata.ID,
				},
			},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by namespace path",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					NamespacePath: &group.FullPath,
				},
			},
			expectCount: len(createdActivityEvents),
		},
		{
			name: "filter by action",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					Actions: []models.ActivityEventAction{models.ActionCreate},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by target type",
			input: &GetActivityEventsInput{
				Filter: &ActivityEventFilter{
					TargetTypes: []models.ActivityEventTargetType{models.TargetGroup},
				},
			},
			expectCount: len(createdActivityEvents),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.ActivityEvents.GetActivityEvents(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.ActivityEvents, test.expectCount)
		})
	}
}

func TestActivityEvents_GetActivityEventsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a user for the activity events
	user, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "test-user-activity-events-pagination",
		Email:    "test-activity-events-pagination@example.com",
	})
	require.NoError(t, err)

	// Create a group for the namespace
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-activity-events-pagination",
		Description: "test group for activity events pagination",
		FullPath:    "test-group-activity-events-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.ActivityEvents.CreateActivityEvent(ctx, &models.ActivityEvent{
			UserID:        &user.Metadata.ID,
			NamespacePath: &group.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetGroup,
			TargetID:      group.Metadata.ID,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		ActivityEventSortableFieldCreatedAtAsc,
		ActivityEventSortableFieldCreatedAtDesc,
		ActivityEventSortableFieldNamespacePathAsc,
		ActivityEventSortableFieldNamespacePathDesc,
		ActivityEventSortableFieldActionAsc,
		ActivityEventSortableFieldActionDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := ActivityEventSortableField(sortByField.getValue())

		result, err := testClient.client.ActivityEvents.GetActivityEvents(ctx, &GetActivityEventsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.ActivityEvents {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
