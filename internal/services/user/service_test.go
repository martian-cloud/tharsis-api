package user

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestGetUserByID(t *testing.T) {
	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID: "user-id-1",
		},
		Username: "user-1",
		Email:    "user@test.com",
		Active:   true,
		Admin:    false,
	}

	type testCase struct {
		caller          auth.Caller
		name            string
		user            *models.User
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "successfully get user by ID",
			caller: &auth.SystemCaller{},
			user:   sampleUser,
		},
		{
			name:            "user not found",
			caller:          &auth.SystemCaller{},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockUsers := db.NewMockUsers(t)

			if test.caller != nil {
				ctx = auth.WithCaller(ctx, test.caller)
				mockUsers.On("GetUserByID", mock.Anything, sampleUser.Metadata.ID).Return(test.user, nil)
			}

			dbClient := &db.Client{
				Users: mockUsers,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualUser, err := service.GetUserByID(ctx, sampleUser.Metadata.ID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.user, actualUser)
		})
	}
}

func TestGetUserByTRN(t *testing.T) {
	sampleUser := &models.User{
		Metadata: models.ResourceMetadata{
			ID:  "user-id-1",
			TRN: types.UserModelType.BuildTRN("user-1"),
		},
		Username: "user-1",
		Email:    "user@test.com",
		Active:   true,
		Admin:    false,
	}

	type testCase struct {
		caller          auth.Caller
		name            string
		user            *models.User
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:   "successfully get user by trn",
			caller: &auth.SystemCaller{},
			user:   sampleUser,
		},
		{
			name:            "user not found",
			caller:          &auth.SystemCaller{},
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "without caller",
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockUsers := db.NewMockUsers(t)

			if test.caller != nil {
				ctx = auth.WithCaller(ctx, test.caller)
				mockUsers.On("GetUserByTRN", mock.Anything, sampleUser.Metadata.TRN).Return(test.user, nil)
			}

			dbClient := &db.Client{
				Users: mockUsers,
			}

			service := &service{
				dbClient: dbClient,
			}

			actualUser, err := service.GetUserByTRN(ctx, sampleUser.Metadata.TRN)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.user, actualUser)
		})
	}
}

func TestGetNotificationPreference(t *testing.T) {
	userID := "user123"
	scope := models.NotificationPreferenceScopeAll

	type testCase struct {
		name               string
		input              *GetNotificationPreferenceInput
		caller             func() auth.Caller
		existingPreference *models.NotificationPreference
		expectErrCode      errors.CodeType
		expectPreference   *namespace.NotificationPreferenceSetting
	}

	testCases := []testCase{
		{
			name:  "successfully get global notification preference",
			input: &GetNotificationPreferenceInput{},
			caller: func() auth.Caller {
				return &auth.UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{
							ID: userID,
						},
					},
				}
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited: false,
				Scope:     scope,
			},
		},
		{
			name: "successfully get group notification preference",
			input: &GetNotificationPreferenceInput{
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited:     false,
				Scope:         scope,
				NamespacePath: ptr.String("test-namespace"),
			},
		},
		{
			name: "successfully get workspace notification preference",
			input: &GetNotificationPreferenceInput{
				NamespacePath: ptr.String("test-namespace/workspace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewWorkspacePermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited:     false,
				Scope:         scope,
				NamespacePath: ptr.String("test-namespace/workspace"),
			},
		},
		{
			name: "user does not have permission to access group",
			input: &GetNotificationPreferenceInput{
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name: "caller cannot get preference because the caller is not a user",
			input: &GetNotificationPreferenceInput{
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				return &auth.ServiceAccountCaller{}
			},
			expectErrCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockGroups := db.NewMockGroups(t)
			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)

			if test.input.NamespacePath != nil {
				var response *models.Group
				if !strings.Contains(*test.input.NamespacePath, "/") {
					response = &models.Group{}
				}
				mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(*test.input.NamespacePath)).Return(response, nil).Maybe()
			}

			if test.expectPreference != nil {
				mockInheritedSettingsResolver.On("GetNotificationPreference", mock.Anything, userID, test.input.NamespacePath).Return(test.expectPreference, nil)
			}

			dbClient := &db.Client{
				Groups: mockGroups,
			}

			logger, _ := logger.NewForTest()

			service := NewService(logger, dbClient, mockInheritedSettingsResolver)

			result, err := service.GetNotificationPreference(auth.WithCaller(ctx, test.caller()), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if test.expectPreference != nil {
				assert.Equal(t, test.expectPreference, result)
			}
		})
	}
}

func TestSetNotificationPreference(t *testing.T) {
	userID := "user123"
	scope := models.NotificationPreferenceScopeAll

	type testCase struct {
		name               string
		input              *SetNotificationPreferenceInput
		caller             func() auth.Caller
		existingPreference *models.NotificationPreference
		expectErrCode      errors.CodeType
		expectPreference   *namespace.NotificationPreferenceSetting
	}

	testCases := []testCase{
		{
			name: "successfully set global notification preference when no existing preference exists",
			input: &SetNotificationPreferenceInput{
				Scope: &scope,
			},
			caller: func() auth.Caller {
				return &auth.UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{
							ID: userID,
						},
					},
				}
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited: false,
				Scope:     scope,
			},
		},
		{
			name: "successfully set global notification preference when existing preference exists",
			input: &SetNotificationPreferenceInput{
				Scope: &scope,
			},
			caller: func() auth.Caller {
				return &auth.UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{
							ID: userID,
						},
					},
				}
			},
			existingPreference: &models.NotificationPreference{
				UserID: userID,
				Scope:  scope,
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited: false,
				Scope:     scope,
			},
		},
		{
			name: "successfully set group notification preference when no existing preference exists",
			input: &SetNotificationPreferenceInput{
				Scope:         &scope,
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited:     false,
				Scope:         scope,
				NamespacePath: ptr.String("test-namespace"),
			},
		},
		{
			name: "successfully clear group notification preference when no existing preference exists",
			input: &SetNotificationPreferenceInput{
				Inherit:       true,
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited: true,
				Scope:     scope,
			},
		},
		{
			name: "successfully clear group notification preference when existing preference exists",
			input: &SetNotificationPreferenceInput{
				Inherit:       true,
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			existingPreference: &models.NotificationPreference{
				UserID:        userID,
				Scope:         scope,
				NamespacePath: ptr.String("test-namespace"),
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited: true,
				Scope:     scope,
			},
		},
		{
			name: "successfully set workspace notification preference when no existing preference exists",
			input: &SetNotificationPreferenceInput{
				Scope:         &scope,
				NamespacePath: ptr.String("test-namespace/workspace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewWorkspacePermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectPreference: &namespace.NotificationPreferenceSetting{
				Inherited:     false,
				Scope:         scope,
				NamespacePath: ptr.String("test-namespace/workspace"),
			},
		},
		{
			name: "user does not have permission to access group",
			input: &SetNotificationPreferenceInput{
				Scope:         &scope,
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
				mockAuthorizer := auth.NewMockAuthorizer(t)

				mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)
				mockAuthorizer.On("RequireAccess", mock.Anything, []models.Permission{models.ViewGroupPermission}, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor, nil)
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name: "caller cannot set preference because the caller is not a user",
			input: &SetNotificationPreferenceInput{
				Scope:         &scope,
				NamespacePath: ptr.String("test-namespace"),
			},
			caller: func() auth.Caller {
				return &auth.ServiceAccountCaller{}
			},
			expectErrCode: errors.EForbidden,
		},
		{
			name: "cannot set global notification preference to inherit",
			input: &SetNotificationPreferenceInput{
				Inherit: true,
				Scope:   &scope,
			},
			caller: func() auth.Caller {
				return &auth.UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{
							ID: userID,
						},
					},
				}
			},
			expectErrCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNotificationPreferences := db.NewMockNotificationPreferences(t)
			mockGroups := db.NewMockGroups(t)

			mockInheritedSettingsResolver := namespace.NewMockInheritedSettingResolver(t)

			if test.input.NamespacePath != nil {
				var response *models.Group
				if !strings.Contains(*test.input.NamespacePath, "/") {
					response = &models.Group{}
				}
				mockGroups.On("GetGroupByTRN", mock.Anything, types.GroupModelType.BuildTRN(*test.input.NamespacePath)).Return(response, nil).Maybe()
			}

			existingPreferences := []models.NotificationPreference{}
			if test.existingPreference != nil {
				existingPreferences = append(existingPreferences, *test.existingPreference)

				if !test.input.Inherit {
					mockNotificationPreferences.On("UpdateNotificationPreference", mock.Anything, &models.NotificationPreference{
						UserID:        userID,
						Scope:         scope,
						NamespacePath: test.input.NamespacePath,
						CustomEvents:  test.input.CustomEvents,
					}).Return(nil, nil).Maybe()
				} else {
					mockNotificationPreferences.On("DeleteNotificationPreference", mock.Anything, test.existingPreference).Return(nil, nil).Maybe()
				}
			} else {
				mockNotificationPreferences.On("CreateNotificationPreference", mock.Anything, &models.NotificationPreference{
					UserID:        userID,
					Scope:         scope,
					NamespacePath: test.input.NamespacePath,
					CustomEvents:  test.input.CustomEvents,
				}).Return(nil, nil).Maybe()
			}

			var globalFilter *bool
			if test.input.NamespacePath == nil {
				globalFilter = ptr.Bool(true)
			}
			mockNotificationPreferences.On("GetNotificationPreferences", mock.Anything, &db.GetNotificationPreferencesInput{
				Filter: &db.NotificationPreferenceFilter{
					UserIDs:       []string{userID},
					NamespacePath: test.input.NamespacePath,
					Global:        globalFilter,
				},
			}).Return(&db.NotificationPreferencesResult{
				NotificationPreferences: existingPreferences,
			}, nil).Maybe()

			if test.expectPreference != nil {
				mockInheritedSettingsResolver.On("GetNotificationPreference", mock.Anything, userID, test.input.NamespacePath).Return(test.expectPreference, nil)
			}

			dbClient := &db.Client{
				NotificationPreferences: mockNotificationPreferences,
				Groups:                  mockGroups,
			}

			logger, _ := logger.NewForTest()

			service := NewService(logger, dbClient, mockInheritedSettingsResolver)

			result, err := service.SetNotificationPreference(auth.WithCaller(ctx, test.caller()), test.input)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			if test.expectPreference != nil {
				assert.Equal(t, test.expectPreference, result)
			}
		})
	}
}

func TestUpdateAdminStatusForUser(t *testing.T) {
	userID := "user-1"

	type testCase struct {
		name            string
		callerType      string
		callerUserID    string
		newAdminStatus  bool
		userToUpdate    *models.User
		callerIsAdmin   bool
		expectAdmin     bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:            "non user callers cannot modify admin status",
			callerType:      "not-user",
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "non admin users cannot modify admin status",
			callerType:      "user",
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "target user not found",
			callerType:      "user",
			callerIsAdmin:   true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "caller cannot modify their own admin status",
			callerType:      "user",
			callerUserID:    userID,
			newAdminStatus:  true,
			callerIsAdmin:   true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name:           "cannot make an inactive user an admin",
			callerType:     "user",
			callerUserID:   userID,
			newAdminStatus: true,
			callerIsAdmin:  true,
			userToUpdate: &models.User{
				Metadata: models.ResourceMetadata{
					ID: userID,
				},
				Username: "user.name",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:       "successfully set non admin active user as admin",
			callerType: "user",
			userToUpdate: &models.User{
				Metadata: models.ResourceMetadata{
					ID: userID,
				},
				Username: "user.name",
				Active:   true,
			},
			callerIsAdmin:  true,
			newAdminStatus: true,
			expectAdmin:    true,
		},
		{
			name:          "successfully set admin active user as non admin",
			callerType:    "user",
			callerIsAdmin: true,
			userToUpdate: &models.User{
				Metadata: models.ResourceMetadata{
					ID: userID,
				},
				Username: "user.name",
				Admin:    true,
				Active:   true,
			},
		},
		{
			name:          "successfully set admin inactive user as non admin",
			callerType:    "user",
			callerIsAdmin: true,
			userToUpdate: &models.User{
				Metadata: models.ResourceMetadata{
					ID: userID,
				},
				Username: "user.name",
				Admin:    true,
				Active:   false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			var mockCaller auth.Caller
			if tc.callerType == "user" {
				mockCaller = &auth.UserCaller{
					User: &models.User{
						Metadata: models.ResourceMetadata{
							ID: tc.callerUserID,
						},
						Username: "calling user",
						Admin:    tc.callerIsAdmin,
					},
				}
			} else {
				mockCaller = auth.NewMockCaller(t)
			}

			mockUsers := db.NewMockUsers(t)

			mockUsers.On("GetUserByID", mock.Anything, userID).Return(tc.userToUpdate, nil).Maybe()

			mockUsers.On("UpdateUser", mock.Anything, tc.userToUpdate).Return(tc.userToUpdate, nil).Maybe()

			logger, _ := logger.NewForTest()

			dbClient := &db.Client{
				Users: mockUsers,
			}

			service := &service{
				dbClient: dbClient,
				logger:   logger,
			}

			actualUser, err := service.UpdateAdminStatusForUser(auth.WithCaller(ctx, mockCaller), &UpdateAdminStatusForUserInput{
				UserID: userID,
				Admin:  tc.newAdminStatus,
			})

			if tc.expectErrorCode != "" {
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, actualUser)
			assert.Equal(t, tc.expectAdmin, actualUser.Admin)
		})
	}
}

func TestGetUserSessions(t *testing.T) {
	userID := "user-id"
	adminUserID := "admin-user-id"
	otherUserID := "other-user-id"
	userSessionID := "user-session-id"

	testCases := []struct {
		name          string
		input         *GetUserSessionsInput
		caller        auth.Caller
		expectError   bool
		expectErrorCode errors.CodeType
	}{
		{
			name: "admin can query any user's sessions",
			input: &GetUserSessionsInput{
				UserID: userID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
		},
		{
			name: "user can query their own sessions",
			input: &GetUserSessionsInput{
				UserID: userID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    false,
				},
			},
		},
		{
			name: "user cannot query other user's sessions",
			input: &GetUserSessionsInput{
				UserID: otherUserID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    false,
				},
			},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "service account cannot query user sessions",
			input: &GetUserSessionsInput{
				UserID: userID,
			},
			caller:          &auth.ServiceAccountCaller{},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUsers := db.NewMockUsers(t)
			mockUserSessions := db.NewMockUserSessions(t)

			dbClient := &db.Client{
				Users:        mockUsers,
				UserSessions: mockUserSessions,
			}

			if !tc.expectError {
				mockUserSessions.On("GetUserSessions", mock.Anything, mock.MatchedBy(func(input *db.GetUserSessionsInput) bool {
					return input.Filter != nil && input.Filter.UserID != nil && *input.Filter.UserID == tc.input.UserID
				})).Return(&db.UserSessionsResult{
					UserSessions: []models.UserSession{
						{
							Metadata: models.ResourceMetadata{ID: userSessionID},
							UserID:   tc.input.UserID,
						},
					},
				}, nil)
			}

			logger, _ := logger.NewForTest()
			testService := NewService(logger, dbClient, nil)

			ctx = auth.WithCaller(ctx, tc.caller)

			result, err := testService.GetUserSessions(ctx, tc.input)

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Len(t, result.UserSessions, 1)
			assert.Equal(t, userSessionID, result.UserSessions[0].Metadata.ID)
		})
	}
}

func TestGetUserSessionByID(t *testing.T) {
	userID := "user-id"
	adminUserID := "admin-user-id"
	otherUserID := "other-user-id"
	userSessionID := "user-session-id"

	testCases := []struct {
		name          string
		userSessionID string
		caller        auth.Caller
		expectError   bool
		expectErrorCode errors.CodeType
	}{
		{
			name:          "admin can access any user session",
			userSessionID: userSessionID,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
		},
		{
			name:          "user can access their own session",
			userSessionID: userSessionID,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    false,
				},
			},
		},
		{
			name:          "user cannot access other user's session",
			userSessionID: userSessionID,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: otherUserID},
					Admin:    false,
				},
			},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:          "service account cannot access user sessions",
			userSessionID: userSessionID,
			caller:        &auth.ServiceAccountCaller{},
			expectError:   true,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUserSessions := db.NewMockUserSessions(t)

			dbClient := &db.Client{
				UserSessions: mockUserSessions,
			}

			if !tc.expectError {
				mockUserSessions.On("GetUserSessionByID", mock.Anything, tc.userSessionID).Return(&models.UserSession{
					Metadata: models.ResourceMetadata{ID: tc.userSessionID},
					UserID:   userID,
				}, nil)
			} else if tc.expectErrorCode == errors.EForbidden {
				// For forbidden cases where we need to check ownership, we still need to return the session
				userCaller, isUserCaller := tc.caller.(*auth.UserCaller)
				if isUserCaller && !userCaller.IsAdmin() {
					mockUserSessions.On("GetUserSessionByID", mock.Anything, tc.userSessionID).Return(&models.UserSession{
						Metadata: models.ResourceMetadata{ID: tc.userSessionID},
						UserID:   userID, // Session belongs to userID, but caller is otherUserID
					}, nil)
				}
				// For service accounts, we don't mock the database call since it should fail before that
			}

			logger, _ := logger.NewForTest()
			testService := NewService(logger, dbClient, nil)

			ctx = auth.WithCaller(ctx, tc.caller)

			result, err := testService.GetUserSessionByID(ctx, tc.userSessionID)

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tc.userSessionID, result.Metadata.ID)
		})
	}
}

func TestGetUserSessionByTRN(t *testing.T) {
	userID := "user-id"
	adminUserID := "admin-user-id"
	otherUserID := "other-user-id"
	userSessionID := "user-session-id"
	username := "test-user"
	sessionTRN := "trn:user_session:" + username + "/US_" + userSessionID

	testCases := []struct {
		name          string
		trn           string
		caller        auth.Caller
		expectError   bool
		expectErrorCode errors.CodeType
	}{
		{
			name: "admin can access any user session by TRN",
			trn:  sessionTRN,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
		},
		{
			name: "user can access their own session by TRN",
			trn:  sessionTRN,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    false,
				},
			},
		},
		{
			name: "user cannot access other user's session by TRN",
			trn:  sessionTRN,
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: otherUserID},
					Admin:    false,
				},
			},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "service account cannot access user sessions by TRN",
			trn:  sessionTRN,
			caller:        &auth.ServiceAccountCaller{},
			expectError:   true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:          "invalid TRN format",
			trn:           "invalid-trn",
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
			expectError:     true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUserSessions := db.NewMockUserSessions(t)

			dbClient := &db.Client{
				UserSessions: mockUserSessions,
			}

			if !tc.expectError {
				mockUserSessions.On("GetUserSessionByTRN", mock.Anything, tc.trn).Return(&models.UserSession{
					Metadata: models.ResourceMetadata{ID: userSessionID, TRN: tc.trn},
					UserID:   userID,
				}, nil)
			} else if tc.expectErrorCode == errors.EForbidden {
				// For forbidden cases where we need to check ownership, we still need to return the session
				userCaller, isUserCaller := tc.caller.(*auth.UserCaller)
				if isUserCaller && !userCaller.IsAdmin() {
					mockUserSessions.On("GetUserSessionByTRN", mock.Anything, tc.trn).Return(&models.UserSession{
						Metadata: models.ResourceMetadata{ID: userSessionID, TRN: tc.trn},
						UserID:   userID, // Session belongs to userID, but caller is otherUserID
					}, nil)
				}
				// For service accounts, we don't mock the database call since it should fail before that
			} else if tc.expectErrorCode == errors.EInvalid {
				// For invalid TRN, the database call should return an error
				mockUserSessions.On("GetUserSessionByTRN", mock.Anything, tc.trn).Return(nil, errors.New("invalid TRN format", errors.WithErrorCode(errors.EInvalid)))
			}

			logger, _ := logger.NewForTest()
			testService := NewService(logger, dbClient, nil)

			ctx = auth.WithCaller(ctx, tc.caller)

			result, err := testService.GetUserSessionByTRN(ctx, tc.trn)

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, userSessionID, result.Metadata.ID)
			assert.Equal(t, tc.trn, result.Metadata.TRN)
		})
	}
}

func TestRevokeUserSession(t *testing.T) {
	userID := "user-id"
	adminUserID := "admin-user-id"
	otherUserID := "other-user-id"
	userSessionID := "user-session-id"

	testCases := []struct {
		name            string
		input           *RevokeUserSessionInput
		caller          auth.Caller
		expectError     bool
		expectErrorCode errors.CodeType
	}{
		{
			name: "admin can revoke any user session",
			input: &RevokeUserSessionInput{
				UserSessionID: userSessionID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
		},
		{
			name: "user can revoke their own session",
			input: &RevokeUserSessionInput{
				UserSessionID: userSessionID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: userID},
					Admin:    false,
				},
			},
		},
		{
			name: "user cannot revoke other user's session",
			input: &RevokeUserSessionInput{
				UserSessionID: userSessionID,
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: otherUserID},
					Admin:    false,
				},
			},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "service account cannot revoke user sessions",
			input: &RevokeUserSessionInput{
				UserSessionID: userSessionID,
			},
			caller:          &auth.ServiceAccountCaller{},
			expectError:     true,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "session not found",
			input: &RevokeUserSessionInput{
				UserSessionID: "non-existent-session",
			},
			caller: &auth.UserCaller{
				User: &models.User{
					Metadata: models.ResourceMetadata{ID: adminUserID},
					Admin:    true,
				},
			},
			expectError:     true,
			expectErrorCode: errors.ENotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockUserSessions := db.NewMockUserSessions(t)

			dbClient := &db.Client{
				UserSessions: mockUserSessions,
			}

			if tc.name == "session not found" {
				mockUserSessions.On("GetUserSessionByID", mock.Anything, tc.input.UserSessionID).Return(nil, nil)
			} else if !tc.expectError {
				// Mock successful case
				mockUserSessions.On("GetUserSessionByID", mock.Anything, tc.input.UserSessionID).Return(&models.UserSession{
					Metadata: models.ResourceMetadata{ID: tc.input.UserSessionID},
					UserID:   userID,
				}, nil)
				mockUserSessions.On("DeleteUserSession", mock.Anything, mock.MatchedBy(func(session *models.UserSession) bool {
					return session.Metadata.ID == tc.input.UserSessionID
				})).Return(nil)
			} else if tc.expectErrorCode == errors.EForbidden {
				// For forbidden cases where we need to check ownership, we still need to return the session
				userCaller, isUserCaller := tc.caller.(*auth.UserCaller)
				if isUserCaller && !userCaller.IsAdmin() {
					mockUserSessions.On("GetUserSessionByID", mock.Anything, tc.input.UserSessionID).Return(&models.UserSession{
						Metadata: models.ResourceMetadata{ID: tc.input.UserSessionID},
						UserID:   userID, // Session belongs to userID, but caller is otherUserID
					}, nil)
				}
			}

			logger, _ := logger.NewForTest()
			testService := NewService(logger, dbClient, nil)

			ctx = auth.WithCaller(ctx, tc.caller)

			err := testService.RevokeUserSession(ctx, tc.input)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectErrorCode != "" {
					assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				}
				return
			}

			require.NoError(t, err)
		})
	}
}
