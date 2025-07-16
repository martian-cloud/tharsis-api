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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for AnnouncementSortableField
func (as AnnouncementSortableField) getValue() string {
	return string(as)
}

func TestGetAnnouncementByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	startTime := time.Now().UTC().Add(-1 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)

	announcement, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Test announcement message",
		StartTime:   startTime,
		EndTime:     &endTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode    errors.CodeType
		name               string
		id                 string
		expectAnnouncement bool
	}

	testCases := []testCase{
		{
			name:               "get resource by id",
			id:                 announcement.Metadata.ID,
			expectAnnouncement: true,
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
			announcement, err := testClient.client.Announcements.GetAnnouncementByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectAnnouncement {
				require.NotNil(t, announcement)
				assert.Equal(t, test.id, announcement.Metadata.ID)
			} else {
				assert.Nil(t, announcement)
			}
		})
	}
}

func TestGetAnnouncementByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	startTime := time.Now().UTC().Add(-1 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)

	announcement, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Test announcement message",
		StartTime:   startTime,
		EndTime:     &endTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode    errors.CodeType
		name               string
		trn                string
		expectAnnouncement bool
	}

	testCases := []testCase{
		{
			name:               "get resource by TRN",
			trn:                announcement.Metadata.TRN,
			expectAnnouncement: true,
		},
		{
			name: "resource with TRN not found",
			trn:  types.AnnouncementModelType.BuildTRN(nonExistentID),
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			announcement, err := testClient.client.Announcements.GetAnnouncementByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectAnnouncement {
				require.NotNil(t, announcement)
				assert.Equal(t, test.trn, announcement.Metadata.TRN)
			} else {
				assert.Nil(t, announcement)
			}
		})
	}
}

func TestCreateAnnouncement(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	startTime := time.Now().UTC().Add(-1 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		announcement    *models.Announcement
	}

	testCases := []testCase{
		{
			name: "successfully create resource",
			announcement: &models.Announcement{
				Message:     "Test announcement message",
				StartTime:   startTime,
				EndTime:     &endTime,
				CreatedBy:   "test-user",
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
			},
		},
		{
			name: "successfully create resource with warning type",
			announcement: &models.Announcement{
				Message:     "Warning announcement",
				StartTime:   startTime,
				EndTime:     &endTime,
				CreatedBy:   "test-user",
				Type:        models.AnnouncementTypeWarning,
				Dismissible: false,
			},
		},
		{
			name: "successfully create resource with error type",
			announcement: &models.Announcement{
				Message:     "Error announcement",
				StartTime:   startTime,
				EndTime:     &endTime,
				CreatedBy:   "test-user",
				Type:        models.AnnouncementTypeError,
				Dismissible: true,
			},
		},
		{
			name: "successfully create resource with success type",
			announcement: &models.Announcement{
				Message:     "Success announcement",
				StartTime:   startTime,
				EndTime:     &endTime,
				CreatedBy:   "test-user",
				Type:        models.AnnouncementTypeSuccess,
				Dismissible: false,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			announcement, err := testClient.client.Announcements.CreateAnnouncement(ctx, test.announcement)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, announcement)
			assert.Equal(t, test.announcement.Message, announcement.Message)
			assert.Equal(t, test.announcement.Type, announcement.Type)
			assert.Equal(t, test.announcement.Dismissible, announcement.Dismissible)
			assert.Equal(t, test.announcement.CreatedBy, announcement.CreatedBy)
		})
	}
}

func TestUpdateAnnouncement(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	startTime := time.Now().UTC().Add(-1 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)

	announcement, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Original message",
		StartTime:   startTime,
		EndTime:     &endTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	newStartTime := time.Now().UTC().Add(-2 * time.Hour)
	newEndTime := time.Now().UTC().Add(2 * time.Hour)

	type testCase struct {
		name             string
		expectErrorCode  errors.CodeType
		version          int
		message          string
		announcementType models.AnnouncementType
		dismissible      bool
		startTime        time.Time
		endTime          *time.Time
	}

	testCases := []testCase{
		{
			name:             "successfully update resource",
			version:          1,
			message:          "Updated message",
			announcementType: models.AnnouncementTypeWarning,
			dismissible:      false,
			startTime:        newStartTime,
			endTime:          &newEndTime,
		},
		{
			name:             "update will fail because resource version doesn't match",
			version:          -1,
			message:          "Updated message",
			announcementType: models.AnnouncementTypeWarning,
			dismissible:      false,
			startTime:        newStartTime,
			endTime:          &newEndTime,
			expectErrorCode:  errors.EOptimisticLock,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAnnouncement, err := testClient.client.Announcements.UpdateAnnouncement(ctx, &models.Announcement{
				Metadata: models.ResourceMetadata{
					ID:      announcement.Metadata.ID,
					Version: test.version,
				},
				Message:     test.message,
				StartTime:   test.startTime,
				EndTime:     test.endTime,
				Type:        test.announcementType,
				Dismissible: test.dismissible,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualAnnouncement)
			assert.Equal(t, test.message, actualAnnouncement.Message)
			assert.Equal(t, test.announcementType, actualAnnouncement.Type)
			assert.Equal(t, test.dismissible, actualAnnouncement.Dismissible)
			assert.Equal(t, test.startTime.Format(time.RFC3339), actualAnnouncement.StartTime.Format(time.RFC3339))
			if test.endTime != nil && actualAnnouncement.EndTime != nil {
				assert.Equal(t, test.endTime.Format(time.RFC3339), actualAnnouncement.EndTime.Format(time.RFC3339))
			} else {
				assert.Equal(t, test.endTime, actualAnnouncement.EndTime)
			}
		})
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	startTime := time.Now().UTC().Add(-1 * time.Hour)
	endTime := time.Now().UTC().Add(1 * time.Hour)

	announcement, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Test announcement message",
		StartTime:   startTime,
		EndTime:     &endTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              announcement.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      announcement.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.Announcements.DeleteAnnouncement(ctx, &models.Announcement{
				Metadata: models.ResourceMetadata{
					ID:      test.id,
					Version: test.version,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestGetAnnouncements(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	currentTime := time.Now().UTC()
	pastTime := currentTime.Add(-2 * time.Hour)
	futureTime := currentTime.Add(2 * time.Hour)

	// Create active announcement
	activeAnnouncement, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Active announcement",
		StartTime:   pastTime,
		EndTime:     &futureTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	// Create inactive announcement (future)
	futureEndTime := futureTime.Add(1 * time.Hour)
	_, err = testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Future announcement",
		StartTime:   futureTime,
		EndTime:     &futureEndTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	// Create inactive announcement (expired)
	_, err = testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Expired announcement",
		StartTime:   pastTime.Add(-1 * time.Hour),
		EndTime:     &pastTime,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	// Create announcement without end time (should be active)
	_, err = testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
		Message:     "Announcement without end time",
		StartTime:   pastTime,
		EndTime:     nil,
		CreatedBy:   "test-user",
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
	})
	require.NoError(t, err)

	type testCase struct {
		filter            *AnnouncementFilter
		name              string
		expectErrorCode   errors.CodeType
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return all announcements",
			expectResultCount: 4, // Updated to include announcement without end time
		},
		{
			name: "return only active announcements",
			filter: &AnnouncementFilter{
				Active: ptr.Bool(true),
			},
			expectResultCount: 2, // Both the regular active announcement and the one without end time
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.Announcements.GetAnnouncements(ctx, &GetAnnouncementsInput{
				Filter: test.filter,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResultCount, len(result.Announcements))

			if test.filter != nil && test.filter.Active != nil && *test.filter.Active {
				var hasActiveAnnouncement, hasAnnouncementWithoutEndTime bool
				for _, ann := range result.Announcements {
					if ann.Metadata.ID == activeAnnouncement.Metadata.ID {
						hasActiveAnnouncement = true
					}
					if ann.EndTime == nil {
						hasAnnouncementWithoutEndTime = true
					}
				}
				assert.True(t, hasActiveAnnouncement)
				assert.True(t, hasAnnouncementWithoutEndTime)
			}
		})
	}
}

func TestGetAnnouncementsWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	resourceCount := 10
	currentTime := time.Now().UTC()

	for i := range resourceCount {
		startTime := currentTime.Add(time.Duration(i) * time.Minute)
		endTime := startTime.Add(1 * time.Hour)

		_, err := testClient.client.Announcements.CreateAnnouncement(ctx, &models.Announcement{
			Message:     fmt.Sprintf("Test announcement %d", i),
			StartTime:   startTime,
			EndTime:     &endTime,
			CreatedBy:   "test-user",
			Type:        models.AnnouncementTypeInfo,
			Dismissible: true,
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		AnnouncementSortableFieldCreatedAtAsc,
		AnnouncementSortableFieldCreatedAtDesc,
		AnnouncementSortableFieldStartTimeAsc,
		AnnouncementSortableFieldStartTimeDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := AnnouncementSortableField(sortByField.getValue())

		result, err := testClient.client.Announcements.GetAnnouncements(ctx, &GetAnnouncementsInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.Announcements {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}
