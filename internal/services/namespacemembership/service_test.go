package namespacemembership

import (
	"context"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email/builder"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

func TestGetNamespaceMembershipByTRN(t *testing.T) {
	sampleMembership := &models.NamespaceMembership{
		Metadata: models.ResourceMetadata{
			ID:  "membership-1",
			TRN: trn.TypeNamespaceMembership.Build("group-1/membership-1"),
		},
		Namespace: models.MembershipNamespace{
			Path:    "group-1",
			GroupID: ptr.String("group-1"),
		},
		RoleID: models.OwnerRoleID.String(),
		UserID: ptr.String("user-1"),
	}

	type testCase struct {
		name          string
		membership    *models.NamespaceMembership
		authError     error
		expectErrCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:       "get namespace membership by TRN",
			membership: sampleMembership,
		},
		{
			name:          "subject does not have access to namespace membership",
			membership:    sampleMembership,
			authError:     errors.New("Forbidden", errors.WithErrorCode(errors.EForbidden)),
			expectErrCode: errors.EForbidden,
		},
		{
			name:          "namespace membership not found",
			expectErrCode: errors.ENotFound,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockCaller := auth.NewMockCaller(t)
			mockNamespaceMemberships := db.NewMockNamespaceMemberships(t)

			mockNamespaceMemberships.On("GetNamespaceMembershipByTRN", mock.Anything, sampleMembership.Metadata.TRN).Return(test.membership, nil)

			if test.membership != nil {
				mockCaller.On("RequirePermission", mock.Anything, models.ViewNamespaceMembershipPermission, mock.Anything).Return(test.authError)
			}

			dbClient := &db.Client{
				NamespaceMemberships: mockNamespaceMemberships,
			}

			service := &service{
				dbClient: dbClient,
			}

			membership, err := service.GetNamespaceMembershipByTRN(auth.WithCaller(ctx, mockCaller), sampleMembership.Metadata.TRN)

			if test.expectErrCode != "" {
				assert.Equal(t, test.expectErrCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.membership, membership)
		})
	}
}

func TestCreateNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		expectNamespaceMembership *models.NamespaceMembership
		input                     CreateNamespaceMembershipInput
		name                      string
		expectErrorCode           errors.CodeType
		hasOwnerRole              bool
	}{
		{
			name: "create user namespace membership with owner role in top level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in nested namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1/ns11/ns111",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata: models.ResourceMetadata{
						ID:  "serviceAccount1",
						TRN: trn.TypeServiceAccount.Build("ns1/ns11/serviceAccount"),
					},
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1/ns11/ns111",
					GroupID: ptr.String("group1"),
				},
				RoleID:           models.OwnerRoleID.String(),
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "create service account namespace membership with owner role in top-level namespace",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata: models.ResourceMetadata{
						ID:  "serviceAccount1",
						TRN: trn.TypeServiceAccount.Build("ns1/serviceAccount"),
					},
				},
			},
			expectNamespaceMembership: &models.NamespaceMembership{
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID:           models.OwnerRoleID.String(),
				ServiceAccountID: ptr.String("serviceAccount1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "no owner role",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				User:          &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "missing user and service account",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "user and service account can't both be defined",
			input: CreateNamespaceMembershipInput{
				NamespacePath:  "ns1",
				RoleID:         models.OwnerRoleID.String(),
				User:           &models.User{Metadata: models.ResourceMetadata{ID: "user1"}},
				ServiceAccount: &models.ServiceAccount{Metadata: models.ResourceMetadata{ID: "serviceAccount1"}},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to create service account namespace membership in a namespace it doesn't exist in",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata: models.ResourceMetadata{
						ID:  "serviceAccount1",
						TRN: trn.TypeServiceAccount.Build("ns2/serviceAccount"),
					},
				},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to create service account namespace membership in a nested namespace it doesn't exist in",
			input: CreateNamespaceMembershipInput{
				NamespacePath: "ns1",
				RoleID:        models.OwnerRoleID.String(),
				ServiceAccount: &models.ServiceAccount{
					Metadata: models.ResourceMetadata{
						ID:  "serviceAccount1",
						TRN: trn.TypeServiceAccount.Build("ns1/ns11/serviceAccount"),
					},
				},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}
			mockCaller.On("RequirePermission", mock.Anything, models.CreateNamespaceMembershipPermission, mock.Anything).Return(authError)

			var userID, serviceAccountID *string
			if test.input.User != nil {
				userID = &test.input.User.Metadata.ID
			} else if test.input.ServiceAccount != nil {
				serviceAccountID = &test.input.ServiceAccount.Metadata.ID
			}

			mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, &db.CreateNamespaceMembershipInput{
				NamespacePath:    test.input.NamespacePath,
				RoleID:           test.input.RoleID,
				UserID:           userID,
				ServiceAccountID: serviceAccountID,
			}).Return(test.expectNamespaceMembership, nil)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			mockUsers := db.MockUsers{}
			mockTransactions.Test(t)

			mockServiceAccounts := db.MockServiceAccounts{}
			mockServiceAccounts.Test(t)

			mockRoles := db.MockRoles{}
			mockRoles.Test(t)

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
				Users:                &mockUsers,
				ServiceAccounts:      &mockServiceAccounts,
				Roles:                &mockRoles,
			}

			mockRoles.On("GetRoleByID", mock.Anything, test.input.RoleID).Return(&models.Role{Name: "role-1"}, nil)

			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(&models.User{
				Username: "mock-user",
				Email:    "mock-user@example.invalid",
			}, nil)

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).Return(&models.ServiceAccount{
				Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("ns1/mock-service-account-name")},
			}, nil)

			// If a new test case is added that uses a team principal, will need to mock GetTeamByID here.

			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything, mock.Anything).
				Return(&db.NamespaceMembershipResult{}, nil).Maybe()

			mockEmailClient := email.MockClient{}
			mockEmailClient.Test(t)

			mockNotifMgr := namespace.MockNotificationManager{}
			mockNotifMgr.Test(t)

			mockTaskManager := asynctask.MockManager{}
			mockTaskManager.Test(t)
			mockTaskManager.On("StartTask", mock.Anything).Maybe()

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockEmailClient, &mockNotifMgr, &mockTaskManager)

			namespaceMembership, err := service.CreateNamespaceMembership(auth.WithCaller(ctx, &mockCaller), &test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectNamespaceMembership, namespaceMembership)
		})
	}
}

func TestUpdateNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		input                *models.NamespaceMembership
		current              *models.NamespaceMembership
		name                 string
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
		hasOwnerRole         bool
	}{
		{
			name: "update namespace membership by reducing role from owner to deployer",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
				{Metadata: models.ResourceMetadata{ID: "2"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole: true,
		},
		{
			name: "update namespace membership by reducing role from owner to deployer in nested group",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1/ns11",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "should not be able to update namespace membership because only one owner exists",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			current: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "current namespace membership not found",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "should not be able to update namespace membership because caller doesn't have owner role",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path: "ns1",
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			mockRoles := db.MockRoles{}
			mockRoles.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}

			mockCaller.On("RequirePermission", mock.Anything, models.UpdateNamespaceMembershipPermission, mock.Anything).Return(authError)

			mockNamespaceMemberships.On("GetNamespaceMembershipByID", mock.Anything, test.input.Metadata.ID).Return(test.current, nil)

			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: []string{test.input.Namespace.Path},
				},
			}
			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			mockNamespaceMemberships.On("UpdateNamespaceMembership", mock.Anything, test.input).Return(test.input, nil)

			mockRoles.On("GetRoleByID", mock.Anything, test.input.RoleID).Return(&models.Role{
				Metadata: models.ResourceMetadata{ID: test.input.RoleID},
				Name:     "role-1",
			}, nil)

			if test.current != nil {
				mockRoles.On("GetRoleByID", mock.Anything, test.current.RoleID).Return(&models.Role{
					Metadata: models.ResourceMetadata{ID: test.current.RoleID},
					Name:     "role-2",
				}, nil)
			}

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
				Roles:                &mockRoles,
			}

			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			mockEmailClient := email.MockClient{}
			mockEmailClient.Test(t)

			mockNotifMgr := namespace.MockNotificationManager{}
			mockNotifMgr.Test(t)

			mockTaskManager := asynctask.MockManager{}
			mockTaskManager.Test(t)
			mockTaskManager.On("StartTask", mock.Anything).Maybe()

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockEmailClient, &mockNotifMgr, &mockTaskManager)

			namespaceMembership, err := service.UpdateNamespaceMembership(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			} else if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.input, namespaceMembership)
		})
	}
}

func TestSendMembershipChangeEmail(t *testing.T) {
	const (
		namespacePath = "parent/child"
		userID        = "user-1"
		teamID        = "team-1"
		saID          = "sa-1"
		ownerUserID   = "owner-1"
		performedBy   = "caller@example.com"
	)

	baseMembership := func() *models.NamespaceMembership {
		return &models.NamespaceMembership{
			Namespace: models.MembershipNamespace{
				Path:    namespacePath,
				GroupID: ptr.String("group-1"),
			},
			RoleID: models.DeployerRoleID.String(),
		}
	}

	type testCase struct {
		name            string
		membership      func() *models.NamespaceMembership
		setupMocks      func(_ *testing.T, mockSAs *db.MockServiceAccounts, mockTeams *db.MockTeams, mockTeamMembers *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, mockEmail *email.MockClient)
		expectError     bool
		expectEmailSent bool
	}

	tests := []testCase{
		{
			name: "user membership - email sent",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.UserID = ptr.String(userID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, mockEmail *email.MockClient) {
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.MatchedBy(func(in *namespace.GetUsersToNotifyInput) bool {
					return in.NamespacePath == namespacePath && len(in.ParticipantUserIDs) == 1 && in.ParticipantUserIDs[0] == userID
				})).Return([]string{userID}, nil)
				mockEmail.On("SendMail", mock.Anything, mock.Anything).Return()
			},
			expectEmailSent: true,
		},
		{
			name: "user membership - GetUsersToNotify returns empty - no email",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.UserID = ptr.String(userID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, _ *email.MockClient) {
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.Anything).Return([]string{}, nil)
			},
		},
		{
			name: "user membership - GetUsersToNotify error",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.UserID = ptr.String(userID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, _ *email.MockClient) {
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
			},
			expectError: true,
		},
		{
			name: "team membership - team members notified",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.TeamID = ptr.String(teamID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, mockTeams *db.MockTeams, mockTeamMembers *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, mockEmail *email.MockClient) {
				mockTeams.On("GetTeamByID", mock.Anything, teamID).Return(&models.Team{Name: "my-team"}, nil)
				mockTeamMembers.On("GetTeamMembers", mock.Anything, &db.GetTeamMembersInput{
					Filter: &db.TeamMemberFilter{TeamIDs: []string{teamID}},
				}).Return(&db.TeamMembersResult{TeamMembers: []models.TeamMember{{UserID: userID}}}, nil)
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.MatchedBy(func(in *namespace.GetUsersToNotifyInput) bool {
					return in.NamespacePath == namespacePath && len(in.ParticipantUserIDs) == 1 && in.ParticipantUserIDs[0] == userID
				})).Return([]string{userID}, nil)
				mockEmail.On("SendMail", mock.Anything, mock.Anything).Return()
			},
			expectEmailSent: true,
		},
		{
			name: "team membership - team not found",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.TeamID = ptr.String(teamID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, mockTeams *db.MockTeams, _ *db.MockTeamMembers, _ *namespace.MockNotificationManager, _ *email.MockClient) {
				mockTeams.On("GetTeamByID", mock.Anything, teamID).Return(nil, nil)
			},
			expectError: true,
		},
		{
			name: "team membership - GetTeamMembers error",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.TeamID = ptr.String(teamID)
				return m
			},
			setupMocks: func(_ *testing.T, _ *db.MockServiceAccounts, mockTeams *db.MockTeams, mockTeamMembers *db.MockTeamMembers, _ *namespace.MockNotificationManager, _ *email.MockClient) {
				mockTeams.On("GetTeamByID", mock.Anything, teamID).Return(&models.Team{Name: "my-team"}, nil)
				mockTeamMembers.On("GetTeamMembers", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))
			},
			expectError: true,
		},
		{
			name: "SA membership - owners notified",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.ServiceAccountID = ptr.String(saID)
				return m
			},
			setupMocks: func(_ *testing.T, mockSAs *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, mockEmail *email.MockClient) {
				mockSAs.On("GetServiceAccountByID", mock.Anything, saID).Return(&models.ServiceAccount{
					Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("parent/child/my-sa")},
				}, nil)
				mockNotifMgr.On("GetNamespaceMembersWithRole", mock.Anything, namespacePath, models.OwnerRoleID.String()).Return([]string{ownerUserID}, nil)
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.MatchedBy(func(in *namespace.GetUsersToNotifyInput) bool {
					return in.NamespacePath == namespacePath && len(in.ParticipantUserIDs) == 1 && in.ParticipantUserIDs[0] == ownerUserID
				})).Return([]string{ownerUserID}, nil)
				mockEmail.On("SendMail", mock.Anything, mock.Anything).Return()
			},
			expectEmailSent: true,
		},
		{
			name: "SA membership - no owners found - no email",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.ServiceAccountID = ptr.String(saID)
				return m
			},
			setupMocks: func(_ *testing.T, mockSAs *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, _ *email.MockClient) {
				mockSAs.On("GetServiceAccountByID", mock.Anything, saID).Return(&models.ServiceAccount{
					Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("parent/child/my-sa")},
				}, nil)
				mockNotifMgr.On("GetNamespaceMembersWithRole", mock.Anything, namespacePath, models.OwnerRoleID.String()).Return([]string{}, nil)
				mockNotifMgr.On("GetUsersToNotify", mock.Anything, mock.Anything).Return([]string{}, nil)
			},
		},
		{
			name: "SA membership - SA not found",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.ServiceAccountID = ptr.String(saID)
				return m
			},
			setupMocks: func(_ *testing.T, mockSAs *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, _ *namespace.MockNotificationManager, _ *email.MockClient) {
				mockSAs.On("GetServiceAccountByID", mock.Anything, saID).Return(nil, nil)
			},
			expectError: true,
		},
		{
			name: "SA membership - GetServiceAccountByID error",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.ServiceAccountID = ptr.String(saID)
				return m
			},
			setupMocks: func(_ *testing.T, mockSAs *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, _ *namespace.MockNotificationManager, _ *email.MockClient) {
				mockSAs.On("GetServiceAccountByID", mock.Anything, saID).Return(nil, errors.New("db error"))
			},
			expectError: true,
		},
		{
			name: "SA membership - GetNamespaceMembersWithRole error",
			membership: func() *models.NamespaceMembership {
				m := baseMembership()
				m.ServiceAccountID = ptr.String(saID)
				return m
			},
			setupMocks: func(_ *testing.T, mockSAs *db.MockServiceAccounts, _ *db.MockTeams, _ *db.MockTeamMembers, mockNotifMgr *namespace.MockNotificationManager, _ *email.MockClient) {
				mockSAs.On("GetServiceAccountByID", mock.Anything, saID).Return(&models.ServiceAccount{
					Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("parent/child/my-sa")},
				}, nil)
				mockNotifMgr.On("GetNamespaceMembersWithRole", mock.Anything, namespacePath, models.OwnerRoleID.String()).Return(nil, errors.New("db error"))
			},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := t.Context()

			mockSAs := db.NewMockServiceAccounts(t)
			mockTeams := db.NewMockTeams(t)
			mockTeamMembers := db.NewMockTeamMembers(t)
			mockNotifMgr := namespace.NewMockNotificationManager(t)
			mockEmail := email.NewMockClient(t)

			test.setupMocks(t, mockSAs, mockTeams, mockTeamMembers, mockNotifMgr, mockEmail)

			mockCaller := auth.NewMockCaller(t)
			mockCaller.On("GetSubject").Return(performedBy).Maybe()

			testLogger, _ := logger.NewForTest()
			svc := &service{
				dbClient: &db.Client{
					ServiceAccounts: mockSAs,
					Teams:           mockTeams,
					TeamMembers:     mockTeamMembers,
				},
				notificationManager: mockNotifMgr,
				emailClient:         mockEmail,
				logger:              testLogger,
			}

			err := svc.sendMembershipChangeEmail(ctx, &sendMembershipChangeEmailInput{
				membership: test.membership(),
				action:     builder.MembershipChangeActionCreated,
				roleName:   "deployer",
				caller:     mockCaller,
			})

			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			if test.expectEmailSent {
				mockEmail.AssertCalled(t, "SendMail", mock.Anything, mock.Anything)
			} else {
				mockEmail.AssertNotCalled(t, "SendMail", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestCreateNamespaceMembership_SkipNotification(t *testing.T) {
	// When SkipNotification=true, CreateNamespaceMembership must not kick off an email task.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const userID = "user1"

	mockNamespaceMemberships := db.MockNamespaceMemberships{}
	mockNamespaceMemberships.Test(t)
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	mockRoles := db.MockRoles{}
	mockRoles.Test(t)
	mockUsers := db.MockUsers{}
	mockUsers.Test(t)
	mockServiceAccounts := db.MockServiceAccounts{}
	mockServiceAccounts.Test(t)

	mockCaller := auth.MockCaller{}
	mockCaller.Test(t)
	mockCaller.On("RequirePermission", mock.Anything, models.CreateNamespaceMembershipPermission, mock.Anything).Return(nil)

	expectedMembership := &models.NamespaceMembership{
		Namespace: models.MembershipNamespace{Path: "ns1", GroupID: ptr.String("group1")},
		RoleID:    models.DeployerRoleID.String(),
		UserID:    ptr.String(userID),
	}

	mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, mock.Anything).Return(expectedMembership, nil)
	mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything, mock.Anything).
		Return(&db.NamespaceMembershipResult{}, nil).Maybe()
	mockRoles.On("GetRoleByID", mock.Anything, models.DeployerRoleID.String()).Return(&models.Role{Name: "deployer"}, nil)
	mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)
	mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(&models.User{Username: "u", Email: "u@example.invalid"}, nil)
	mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).
		Return(&models.ServiceAccount{Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("ns1/sa")}}, nil)

	mockEmailClient := email.NewMockClient(t)
	mockNotifMgr := namespace.MockNotificationManager{}
	mockNotifMgr.Test(t)
	// No StartTask expectation -- it must NOT be called.
	mockTaskManager := asynctask.MockManager{}
	mockTaskManager.Test(t)

	testLogger, _ := logger.NewForTest()
	svc := NewService(testLogger, &db.Client{
		NamespaceMemberships: &mockNamespaceMemberships,
		Transactions:         &mockTransactions,
		Roles:                &mockRoles,
		Users:                &mockUsers,
		ServiceAccounts:      &mockServiceAccounts,
	}, mockEmailClient, &mockNotifMgr, &mockTaskManager)

	_, err := svc.CreateNamespaceMembership(auth.WithCaller(ctx, &mockCaller), &CreateNamespaceMembershipInput{
		NamespacePath:    "ns1",
		RoleID:           models.DeployerRoleID.String(),
		User:             &models.User{Metadata: models.ResourceMetadata{ID: userID}},
		SkipNotification: true,
	})
	require.NoError(t, err)
	mockTaskManager.AssertNotCalled(t, "StartTask")
}

func TestCreateNamespaceMembership_CallerIsSubject(t *testing.T) {
	// When the caller's user ID matches the membership subject, no email task should be started.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const userID = "user1"

	mockNamespaceMemberships := db.MockNamespaceMemberships{}
	mockNamespaceMemberships.Test(t)
	mockTransactions := db.MockTransactions{}
	mockTransactions.Test(t)
	mockRoles := db.MockRoles{}
	mockRoles.Test(t)
	mockUsers := db.MockUsers{}
	mockUsers.Test(t)
	mockServiceAccounts := db.MockServiceAccounts{}
	mockServiceAccounts.Test(t)

	expectedMembership := &models.NamespaceMembership{
		Namespace: models.MembershipNamespace{Path: "ns1", GroupID: ptr.String("group1")},
		RoleID:    models.DeployerRoleID.String(),
		UserID:    ptr.String(userID),
	}

	mockNamespaceMemberships.On("CreateNamespaceMembership", mock.Anything, mock.Anything).Return(expectedMembership, nil)
	mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything, mock.Anything).
		Return(&db.NamespaceMembershipResult{}, nil).Maybe()
	mockRoles.On("GetRoleByID", mock.Anything, models.DeployerRoleID.String()).Return(&models.Role{Name: "deployer"}, nil)
	mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
	mockTransactions.On("CommitTx", mock.Anything).Return(nil)
	mockUsers.On("GetUserByID", mock.Anything, mock.Anything).Return(&models.User{Username: "u", Email: "u@example.invalid"}, nil)
	mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).
		Return(&models.ServiceAccount{Metadata: models.ResourceMetadata{TRN: trn.TypeServiceAccount.Build("ns1/sa")}}, nil)

	mockActivityEvents := db.NewMockActivityEvents(t)
	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.Anything).
		Return(&models.ActivityEvent{}, nil)

	mockAuthorizer := auth.NewMockAuthorizer(t)
	mockAuthorizer.On("RequireAccess", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMaintenanceMonitor := maintenance.NewMockMonitor(t)
	mockMaintenanceMonitor.On("InMaintenanceMode", mock.Anything).Return(false, nil)

	// Caller's user ID matches the membership subject.
	userCaller := auth.NewUserCaller(
		&models.User{Metadata: models.ResourceMetadata{ID: userID}},
		mockAuthorizer,
		&db.Client{},
		mockMaintenanceMonitor,
		nil,
	)
	callerCtx := auth.WithCaller(ctx, userCaller)

	mockTransactions.On("BeginTx", mock.Anything).Return(callerCtx, nil)

	mockEmailClient := email.NewMockClient(t)
	mockNotifMgr := namespace.MockNotificationManager{}
	mockNotifMgr.Test(t)
	// No StartTask expectation -- it must NOT be called.
	mockTaskManager := asynctask.MockManager{}
	mockTaskManager.Test(t)

	testLogger, _ := logger.NewForTest()
	svc := NewService(testLogger, &db.Client{
		NamespaceMemberships: &mockNamespaceMemberships,
		Transactions:         &mockTransactions,
		Roles:                &mockRoles,
		Users:                &mockUsers,
		ServiceAccounts:      &mockServiceAccounts,
		ActivityEvents:       mockActivityEvents,
	}, mockEmailClient, &mockNotifMgr, &mockTaskManager)

	_, err := svc.CreateNamespaceMembership(callerCtx, &CreateNamespaceMembershipInput{
		NamespacePath: "ns1",
		RoleID:        models.DeployerRoleID.String(),
		User:          &models.User{Metadata: models.ResourceMetadata{ID: userID}},
	})
	require.NoError(t, err)
	mockTaskManager.AssertNotCalled(t, "StartTask")
}

func TestDeleteNamespaceMembership(t *testing.T) {
	// Test cases
	tests := []struct {
		input                *models.NamespaceMembership
		name                 string
		expectErrorCode      errors.CodeType
		namespaceMemberships []models.NamespaceMembership
		hasOwnerRole         bool
	}{
		{
			name: "delete namespace membership",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:    "ns1",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
				{Metadata: models.ResourceMetadata{ID: "2"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole: true,
		},
		{
			name: "delete namespace membership in nested group",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:    "ns1/ns11",
					GroupID: ptr.String("group1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole: true,
		},
		{
			name: "should not be able to delete namespace membership because only one owner exists",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:        "ns1",
					WorkspaceID: ptr.String("ws1"),
				},
				RoleID: models.OwnerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			namespaceMemberships: []models.NamespaceMembership{
				{Metadata: models.ResourceMetadata{ID: "1"}, RoleID: models.OwnerRoleID.String()},
			},
			hasOwnerRole:    true,
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "should not be able to delete namespace membership because caller doesn't have owner role",
			input: &models.NamespaceMembership{
				Metadata: models.ResourceMetadata{ID: "1"},
				Namespace: models.MembershipNamespace{
					Path:        "ns1",
					WorkspaceID: ptr.String("ws1"),
				},
				RoleID: models.DeployerRoleID.String(),
				UserID: ptr.String("user1"),
			},
			hasOwnerRole:    false,
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockNamespaceMemberships := db.MockNamespaceMemberships{}
			mockNamespaceMemberships.Test(t)

			mockCaller := auth.MockCaller{}
			mockCaller.Test(t)

			var authError error
			if !test.hasOwnerRole {
				authError = errors.New("not authorized", errors.WithErrorCode(errors.EForbidden))
			}

			mockCaller.On("RequirePermission", mock.Anything, models.DeleteNamespaceMembershipPermission, mock.Anything).Return(authError)

			getNamespaceMembershipsInput := &db.GetNamespaceMembershipsInput{
				Filter: &db.NamespaceMembershipFilter{
					NamespacePaths: []string{test.input.Namespace.Path},
				},
			}
			mockNamespaceMemberships.On("GetNamespaceMemberships", mock.Anything,
				getNamespaceMembershipsInput).Return(&db.NamespaceMembershipResult{
				NamespaceMemberships: test.namespaceMemberships,
			}, nil)

			mockNamespaceMemberships.On("DeleteNamespaceMembership", mock.Anything, test.input).Return(nil)

			mockTransactions := db.MockTransactions{}
			mockTransactions.Test(t)
			// The mocks are enabled by the above function.

			dbClient := db.Client{
				NamespaceMemberships: &mockNamespaceMemberships,
				Transactions:         &mockTransactions,
			}

			mockEmailClient := email.MockClient{}
			mockEmailClient.Test(t)

			mockNotifMgr := namespace.MockNotificationManager{}
			mockNotifMgr.Test(t)

			mockTaskManager := asynctask.MockManager{}
			mockTaskManager.Test(t)
			mockTaskManager.On("StartTask", mock.Anything).Maybe()

			logger, _ := logger.NewForTest()
			service := NewService(logger, &dbClient, &mockEmailClient, &mockNotifMgr, &mockTaskManager)

			mockTransactions.On("BeginTx", mock.Anything).Return(auth.WithCaller(ctx, &mockCaller), nil)
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
			mockTransactions.On("CommitTx", mock.Anything).Return(nil)

			err := service.DeleteNamespaceMembership(auth.WithCaller(ctx, &mockCaller), test.input)
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
