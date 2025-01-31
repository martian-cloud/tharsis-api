package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gotest.tools/v3/assert"
)

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
