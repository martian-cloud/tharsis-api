package announcement

import (
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestNewService(t *testing.T) {
	logger, _ := logger.NewForTest()
	dbClient := &db.Client{}

	expect := &service{
		logger:   logger,
		dbClient: dbClient,
	}

	assert.Equal(t, expect, NewService(logger, dbClient))
}

func TestGetAnnouncementByID(t *testing.T) {
	testID := "test-announcement-id"
	endTime := time.Now().Add(time.Hour)
	sampleAnnouncement := &models.Announcement{
		Metadata: models.ResourceMetadata{
			ID: testID,
		},
		Message:     "Test announcement",
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     &endTime,
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
		CreatedBy:   "test-user",
	}

	type testCase struct {
		name               string
		expectAnnouncement *models.Announcement
		expectErrorCode    errors.CodeType
		withCaller         bool
		announcementFromDB *models.Announcement
	}

	tests := []testCase{
		{
			name:               "successfully get announcement by ID",
			expectAnnouncement: sampleAnnouncement,
			withCaller:         true,
			announcementFromDB: sampleAnnouncement,
		},
		{
			name:            "no caller returns error",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "announcement not found",
			expectErrorCode: errors.ENotFound,
			withCaller:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockAnnouncements.On("GetAnnouncementByID", mock.Anything, testID).Return(test.announcementFromDB, nil)
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			service := &service{
				dbClient: dbClient,
			}

			announcement, err := service.GetAnnouncementByID(ctx, testID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectAnnouncement, announcement)
		})
	}
}

func TestGetAnnouncementByTRN(t *testing.T) {
	testTRN := "test-announcement-trn"
	endTime := time.Now().Add(time.Hour)
	sampleAnnouncement := &models.Announcement{
		Metadata: models.ResourceMetadata{
			ID: "test-id",
		},
		Message:     "Test announcement",
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     &endTime,
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
		CreatedBy:   "test-user",
	}

	type testCase struct {
		name               string
		expectAnnouncement *models.Announcement
		expectErrorCode    errors.CodeType
		withCaller         bool
		announcementFromDB *models.Announcement
	}

	tests := []testCase{
		{
			name:               "successfully get announcement by TRN",
			expectAnnouncement: sampleAnnouncement,
			withCaller:         true,
			announcementFromDB: sampleAnnouncement,
		},
		{
			name:            "no caller returns error",
			expectErrorCode: errors.EUnauthorized,
		},
		{
			name:            "announcement not found",
			expectErrorCode: errors.ENotFound,
			withCaller:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockAnnouncements.On("GetAnnouncementByTRN", mock.Anything, testTRN).Return(test.announcementFromDB, nil)
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			service := &service{
				dbClient: dbClient,
			}

			announcement, err := service.GetAnnouncementByTRN(ctx, testTRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectAnnouncement, announcement)
		})
	}
}

func TestGetAnnouncements(t *testing.T) {
	endTime := time.Now().Add(time.Hour)
	sampleResult := &db.AnnouncementsResult{
		Announcements: []models.Announcement{
			{
				Metadata:    models.ResourceMetadata{ID: "test-1"},
				Message:     "Test announcement 1",
				StartTime:   time.Now().Add(-time.Hour),
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
				CreatedBy:   "test-user",
			},
		},
		PageInfo: &pagination.PageInfo{},
	}

	type testCase struct {
		name            string
		input           *GetAnnouncementsInput
		expectResult    *db.AnnouncementsResult
		expectErrorCode errors.CodeType
		withCaller      bool
	}

	tests := []testCase{
		{
			name: "successfully get announcements",
			input: &GetAnnouncementsInput{
				Active: ptr.Bool(true),
			},
			expectResult: sampleResult,
			withCaller:   true,
		},
		{
			name:            "no caller returns error",
			expectErrorCode: errors.EUnauthorized,
			input:           &GetAnnouncementsInput{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			if test.withCaller {
				ctx = auth.WithCaller(ctx, mockCaller)

				expectedDBInput := &db.GetAnnouncementsInput{
					Sort:              test.input.Sort,
					PaginationOptions: test.input.PaginationOptions,
				}
				if test.input.Active != nil {
					expectedDBInput.Filter = &db.AnnouncementFilter{
						Active: test.input.Active,
					}
				}

				mockAnnouncements.On("GetAnnouncements", mock.Anything, expectedDBInput).Return(test.expectResult, nil)
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			service := &service{
				dbClient: dbClient,
			}

			result, err := service.GetAnnouncements(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectResult, result)
		})
	}
}

func TestCreateAnnouncement(t *testing.T) {
	testSubject := "testSubject"
	startTime := time.Now().Add(time.Hour)
	endTime := time.Now().Add(2 * time.Hour)

	type testCase struct {
		name               string
		input              *CreateAnnouncementInput
		expectAnnouncement *models.Announcement
		expectErrorCode    errors.CodeType
		isAdmin            bool
		authError          error
	}

	tests := []testCase{
		{
			name: "admin can create announcement",
			input: &CreateAnnouncementInput{
				Message:     "Test announcement",
				StartTime:   &startTime,
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
			},
			expectAnnouncement: &models.Announcement{
				Metadata: models.ResourceMetadata{
					ID: "created-id",
				},
				Message:     "Test announcement",
				StartTime:   startTime,
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
				CreatedBy:   testSubject,
			},
			isAdmin: true,
		},
		{
			name: "admin can create announcement with nil start time",
			input: &CreateAnnouncementInput{
				Message:     "Test announcement",
				StartTime:   nil,
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
			},
			expectAnnouncement: &models.Announcement{
				Metadata: models.ResourceMetadata{
					ID: "created-id",
				},
				Message:     "Test announcement",
				StartTime:   time.Now().UTC(),
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
				CreatedBy:   testSubject,
			},
			isAdmin: true,
		},
		{
			name: "non-admin caller cannot create announcement",
			input: &CreateAnnouncementInput{
				Message:     "Test announcement",
				StartTime:   &startTime,
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
			},
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "auth error",
			input: &CreateAnnouncementInput{
				Message:     "Test announcement",
				StartTime:   &startTime,
				EndTime:     &endTime,
				Type:        models.AnnouncementTypeInfo,
				Dismissible: true,
			},
			expectErrorCode: errors.EUnauthorized,
			authError:       errors.New("auth error", errors.WithErrorCode(errors.EUnauthorized)),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			if test.authError == nil {
				ctx = auth.WithCaller(ctx, mockCaller)
				mockCaller.On("IsAdmin").Return(test.isAdmin)

				if test.isAdmin {
					mockCaller.On("GetSubject").Return(testSubject)
					mockAnnouncements.On("CreateAnnouncement", mock.Anything, mock.AnythingOfType("*models.Announcement")).Return(test.expectAnnouncement, nil)
				}
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:   logger,
				dbClient: dbClient,
			}

			announcement, err := service.CreateAnnouncement(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectAnnouncement, announcement)
		})
	}
}

func TestUpdateAnnouncement(t *testing.T) {
	testSubject := "testSubject"
	testID := "test-announcement-id"
	endTime := time.Now().Add(2 * time.Hour)

	existingAnnouncement := &models.Announcement{
		Metadata: models.ResourceMetadata{
			ID: testID,
		},
		Message:     "Original message",
		StartTime:   time.Now().Add(time.Hour),
		EndTime:     &endTime,
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
		CreatedBy:   testSubject,
	}

	updatedMessage := "Updated message"
	updatedType := models.AnnouncementTypeWarning

	type testCase struct {
		name                 string
		expectAnnouncement   *models.Announcement
		expectErrorCode      errors.CodeType
		isAdmin              bool
		existingAnnouncement *models.Announcement
	}

	tests := []testCase{
		{
			name: "admin can update announcement",
			expectAnnouncement: &models.Announcement{
				Metadata: models.ResourceMetadata{
					ID: testID,
				},
				Message:     updatedMessage,
				StartTime:   existingAnnouncement.StartTime,
				Type:        updatedType,
				Dismissible: existingAnnouncement.Dismissible,
				CreatedBy:   testSubject,
			},
			isAdmin:              true,
			existingAnnouncement: existingAnnouncement,
		},
		{
			name:            "non-admin caller cannot update announcement",
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "announcement not found",
			expectErrorCode: errors.ENotFound,
			isAdmin:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("IsAdmin").Return(test.isAdmin)

			if test.isAdmin {
				mockCaller.On("GetSubject").Return(testSubject).Maybe()
				mockAnnouncements.On("GetAnnouncementByID", mock.Anything, testID).Return(test.existingAnnouncement, nil)

				if test.existingAnnouncement != nil {
					mockAnnouncements.On("UpdateAnnouncement", mock.Anything, test.expectAnnouncement).Return(test.expectAnnouncement, nil)
				}
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:   logger,
				dbClient: dbClient,
			}

			announcement, err := service.UpdateAnnouncement(auth.WithCaller(ctx, mockCaller), &UpdateAnnouncementInput{
				ID:      testID,
				Message: &updatedMessage,
				Type:    &updatedType,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expectAnnouncement, announcement)
		})
	}
}

func TestDeleteAnnouncement(t *testing.T) {
	testSubject := "testSubject"
	testID := "test-announcement-id"
	endTime := time.Now().Add(2 * time.Hour)

	existingAnnouncement := &models.Announcement{
		Metadata: models.ResourceMetadata{
			ID:      testID,
			Version: 1,
		},
		Message:     "Test message",
		StartTime:   time.Now().Add(time.Hour),
		EndTime:     &endTime,
		Type:        models.AnnouncementTypeInfo,
		Dismissible: true,
		CreatedBy:   testSubject,
	}

	type testCase struct {
		name                 string
		expectErrorCode      errors.CodeType
		isAdmin              bool
		existingAnnouncement *models.Announcement
	}

	tests := []testCase{
		{
			name:                 "admin can delete announcement",
			isAdmin:              true,
			existingAnnouncement: existingAnnouncement,
		},
		{
			name:            "non-admin caller cannot delete announcement",
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "announcement not found",
			expectErrorCode: errors.ENotFound,
			isAdmin:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockAnnouncements := db.NewMockAnnouncements(t)
			mockCaller := auth.NewMockCaller(t)

			mockCaller.On("IsAdmin").Return(test.isAdmin)

			if test.isAdmin {
				mockCaller.On("GetSubject").Return(testSubject).Maybe()
				mockAnnouncements.On("GetAnnouncementByID", mock.Anything, testID).Return(test.existingAnnouncement, nil)

				if test.existingAnnouncement != nil {
					mockAnnouncements.On("DeleteAnnouncement", mock.Anything, existingAnnouncement).Return(nil)
				}
			}

			dbClient := &db.Client{
				Announcements: mockAnnouncements,
			}

			logger, _ := logger.NewForTest()

			service := &service{
				logger:   logger,
				dbClient: dbClient,
			}

			err := service.DeleteAnnouncement(auth.WithCaller(ctx, mockCaller), &DeleteAnnouncementInput{
				ID: testID,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
		})
	}
}
