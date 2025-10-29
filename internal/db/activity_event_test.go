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
		}, {
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
