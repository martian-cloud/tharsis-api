package user

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gotest.tools/v3/assert"
)

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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewWorkspacePermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockGroups.On("GetGroupByFullPath", mock.Anything, *test.input.NamespacePath).Return(response, nil).Maybe()
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewWorkspacePermission}, mock.Anything).Return(nil)

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockAuthorizer.On("RequireAccess", mock.Anything, []permissions.Permission{permissions.ViewGroupPermission}, mock.Anything).
					Return(errors.New("forbidden", errors.WithErrorCode(errors.EForbidden)))

				return auth.NewUserCaller(&models.User{
					Metadata: models.ResourceMetadata{
						ID: userID,
					},
				}, mockAuthorizer, nil, mockMaintenanceMonitor)
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
				mockGroups.On("GetGroupByFullPath", mock.Anything, *test.input.NamespacePath).Return(response, nil).Maybe()
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
