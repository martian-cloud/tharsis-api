package managedidentity

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
)

func TestGetManagedIdentities(t *testing.T) {
	sampleResult := &db.ManagedIdentitiesResult{
		PageInfo: &db.PageInfo{
			Cursor: func(item interface{}) (*string, error) {
				return nil, nil
			},
			TotalCount:      1,
			HasNextPage:     false,
			HasPreviousPage: false,
		},
		ManagedIdentities: []models.ManagedIdentity{
			{
				Name: "a-sample-managed-identity",
			},
		},
	}

	type testCase struct {
		authError       error
		input           *GetManagedIdentitiesInput
		dbInput         *db.GetManagedIdentitiesInput
		expectResult    *db.ManagedIdentitiesResult
		expectErrorCode string
		name            string
	}

	testCases := []testCase{
		{
			name: "positive: mostly empty input",
			input: &GetManagedIdentitiesInput{
				NamespacePath: "a-namespace",
			},
			dbInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					NamespacePaths: []string{"a-namespace"},
				},
			},
			expectResult: sampleResult,
		},
		{
			name: "positive: mostly empty input with include inherited",
			input: &GetManagedIdentitiesInput{
				NamespacePath:    "a-namespace/a-workspace",
				IncludeInherited: true,
			},
			dbInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					NamespacePaths: []string{"a-namespace/a-workspace", "a-namespace"},
				},
			},
			expectResult: sampleResult,
		},
		{
			name: "positive: input with search field populated",
			input: &GetManagedIdentitiesInput{
				NamespacePath:    "a-namespace/a-workspace",
				IncludeInherited: true,
				Search:           ptr.String("a-sample"),
			},
			dbInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					NamespacePaths: []string{"a-namespace/a-workspace", "a-namespace"},
					Search:         ptr.String("a-sample"),
				},
			},
			expectResult: sampleResult,
		},
		{
			name: "negative: subject does not have viewer access to namespace",
			input: &GetManagedIdentitiesInput{
				NamespacePath: "a-namespace/a-workspace",
			},
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockCaller := auth.NewMockCaller(t)
			mockManagedIdentities := db.NewMockManagedIdentities(t)

			mockCaller.On("RequireAccessToNamespace", mock.Anything, test.input.NamespacePath, models.ViewerRole).Return(test.authError)

			mockManagedIdentities.On("GetManagedIdentities", mock.Anything, test.dbInput).Return(test.expectResult, nil).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			result, err := service.GetManagedIdentities(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, result)
		})
	}
}

func TestDeleteManagedIdentity(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-id",
		},
		Name:         "a-managed-identity-to-delete",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String("some/resource"),
		Action:        models.ActionDeleteChildResource,
		TargetType:    models.TargetGroup,
		TargetID:      sampleManagedIdentity.GroupID,
		Payload: &models.ActivityEventDeleteChildResourcePayload{
			Name: sampleManagedIdentity.Name,
			ID:   sampleManagedIdentity.Metadata.ID,
			Type: string(models.TargetManagedIdentity),
		},
	}

	type testCase struct {
		input                     *DeleteManagedIdentityInput
		authError                 error
		expectErrorCode           string
		name                      string
		managedIdentityWorkspaces []models.Workspace
	}

	testCases := []testCase{
		{
			name: "positive: successfully delete a managed identity",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentity,
			},
		},
		{
			name: "positive: successfully delete a managed identity with force option",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentity,
				Force:           true,
			},
		},
		{
			name: "negative: no force option and managed identity is assigned to a workspace",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentity,
			},
			managedIdentityWorkspaces: []models.Workspace{
				{
					FullPath: "some/associated/workspace",
				},
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "negative: attempting to delete a managed identity alias",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: &models.ManagedIdentity{
					AliasSourceID: &sampleManagedIdentity.Metadata.ID,
				},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:      "negative: subject does not have deployer role in group",
			authError: errors.NewError(errors.EForbidden, "Forbidden"),
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentity,
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("DeleteManagedIdentity", mock.Anything, test.input.ManagedIdentity).Return(nil).Maybe()

			mockWorkspaces.On("GetWorkspacesForManagedIdentity", mock.Anything, sampleManagedIdentity.Metadata.ID).Return(test.managedIdentityWorkspaces, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockCaller.On("RequireAccessToGroup", mock.Anything, test.input.ManagedIdentity.GroupID, models.DeployerRole).Return(test.authError).Maybe()
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Workspaces:        mockWorkspaces,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			err := service.DeleteManagedIdentity(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetManagedIdentitiesForWorkspace(t *testing.T) {
	sampleManagedIdentity := models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
	}

	type testCase struct {
		name            string
		workspaceID     string
		expectErrorCode string
		authError       error
		expectResult    []models.ManagedIdentity
	}

	testCases := []testCase{
		{
			name:        "positive: successfully returns managed identities for a workspace",
			workspaceID: "some-workspace-id",
			expectResult: []models.ManagedIdentity{
				sampleManagedIdentity,
			},
		},
		{
			name:            "negative: subject does not have viewer access to workspace",
			workspaceID:     "some-workspace-id",
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, test.workspaceID).Return(test.expectResult, nil).Maybe()

			mockCaller.On("RequireAccessToWorkspace", mock.Anything, test.workspaceID, models.ViewerRole).Return(test.authError)

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			result, err := service.GetManagedIdentitiesForWorkspace(auth.WithCaller(ctx, mockCaller), test.workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult, result)
		})
	}
}

func TestAddManagedIdentityToWorkspace(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleWorkspace := &models.Workspace{
		FullPath: "some/resource/path",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: &sampleWorkspace.FullPath,
		Action:        models.ActionAdd,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      sampleManagedIdentity.Metadata.ID,
	}

	type testCase struct {
		authError               error
		existingManagedIdentity *models.ManagedIdentity
		existingWorkspace       *models.Workspace
		name                    string
		managedIdentityID       string
		workspaceID             string
		expectErrorCode         string
		identitiesInWorkspace   []models.ManagedIdentity
	}

	testCases := []testCase{
		{
			name:                    "positive: successfully add managed identity to workspace",
			existingManagedIdentity: sampleManagedIdentity,
			existingWorkspace:       sampleWorkspace,
			identitiesInWorkspace:   []models.ManagedIdentity{},
			managedIdentityID:       "some-managed-identity-id",
			workspaceID:             "some-workspace-id",
		},
		{
			name:              "negative: managed identity being added doesn't exist",
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			expectErrorCode:   errors.ENotFound,
		},
		{
			name:                    "negative: managed identity is not under the same group hierarchy",
			existingManagedIdentity: sampleManagedIdentity,
			existingWorkspace: &models.Workspace{
				FullPath: "another/path",
			},
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:                    "negative: managed identity with type is already assigned to workspace",
			existingManagedIdentity: sampleManagedIdentity,
			existingWorkspace:       sampleWorkspace,
			identitiesInWorkspace: []models.ManagedIdentity{
				*sampleManagedIdentity,
			},
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "negative: subject does not have deployer access to workspace",
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			authError:         errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:   errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockWorkspaces := workspace.NewMockService(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.managedIdentityID).Return(test.existingManagedIdentity, nil).Maybe()
			mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, test.workspaceID).Return(test.identitiesInWorkspace, nil).Maybe()
			mockManagedIdentities.On("AddManagedIdentityToWorkspace", mock.Anything, test.managedIdentityID, test.workspaceID).Return(nil).Maybe()

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.workspaceID).Return(test.existingWorkspace, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockCaller.On("RequireAccessToWorkspace", mock.Anything, test.workspaceID, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, mockWorkspaces, nil, mockActivityEvents)

			err := service.AddManagedIdentityToWorkspace(auth.WithCaller(ctx, mockCaller), test.managedIdentityID, test.workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRemoveManagedIdentityFromWorkspace(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleWorkspace := &models.Workspace{
		FullPath: "some/resource/path",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: &sampleWorkspace.FullPath,
		Action:        models.ActionRemove,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      sampleManagedIdentity.Metadata.ID,
	}

	type testCase struct {
		authError               error
		existingManagedIdentity *models.ManagedIdentity
		name                    string
		managedIdentityID       string
		workspaceID             string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name:                    "positive: successfully remove managed identity from workspace",
			existingManagedIdentity: sampleManagedIdentity,
			managedIdentityID:       "some-managed-identity-id",
			workspaceID:             "some-workspace-id",
		},
		{
			name:              "negative: managed identity being removed doesn't exist",
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			expectErrorCode:   errors.ENotFound,
		},
		{
			name:              "negative: subject does not have deployer access to workspace",
			managedIdentityID: "some-managed-identity-id",
			workspaceID:       "some-workspace-id",
			authError:         errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:   errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockWorkspaces := workspace.NewMockService(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.managedIdentityID).Return(test.existingManagedIdentity, nil).Maybe()
			mockManagedIdentities.On("RemoveManagedIdentityFromWorkspace", mock.Anything, test.managedIdentityID, test.workspaceID).Return(nil).Maybe()

			mockWorkspaces.On("GetWorkspaceByID", mock.Anything, test.workspaceID).Return(sampleWorkspace, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockCaller.On("RequireAccessToWorkspace", mock.Anything, test.workspaceID, mock.Anything).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, mockWorkspaces, nil, mockActivityEvents)

			err := service.RemoveManagedIdentityFromWorkspace(auth.WithCaller(ctx, mockCaller), test.managedIdentityID, test.workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetManagedIdentityByID(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	type testCase struct {
		authError             error
		expectManagedIdentity *models.ManagedIdentity
		name                  string
		searchID              string
		expectErrorCode       string
	}

	testCases := []testCase{
		{
			name:                  "positive: successfully return a managed identity",
			expectManagedIdentity: sampleManagedIdentity,
			searchID:              "some-managed-identity-id",
		},
		{
			name:            "negative: managed identity doesn't exist",
			searchID:        "some-managed-identity-id",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:                  "negative: subject does not have access to resource",
			searchID:              "some-managed-identity-id",
			expectManagedIdentity: sampleManagedIdentity,
			authError:             errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:       errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.searchID).Return(test.expectManagedIdentity, nil)

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, sampleManagedIdentity.GroupID).Return(test.authError).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			identity, err := service.GetManagedIdentityByID(auth.WithCaller(ctx, mockCaller), test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectManagedIdentity, identity)
		})
	}
}

func TestGetManagedIdentityByPath(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	type testCase struct {
		authError             error
		expectManagedIdentity *models.ManagedIdentity
		name                  string
		searchPath            string
		expectErrorCode       string
	}

	testCases := []testCase{
		{
			name:                  "positive: successfully returns a managed identity",
			expectManagedIdentity: sampleManagedIdentity,
			searchPath:            "some/resource/path",
		},
		{
			name:            "negative: path is invalid",
			searchPath:      "/invalid/path/",
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                  "negative: subject does not have access to group resource",
			searchPath:            "some/resource/path",
			expectManagedIdentity: sampleManagedIdentity,
			authError:             errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:       errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByPath", mock.Anything, test.searchPath).Return(test.expectManagedIdentity, nil).Maybe()

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, sampleManagedIdentity.GroupID).Return(test.authError).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			identity, err := service.GetManagedIdentityByPath(auth.WithCaller(ctx, mockCaller), test.searchPath)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectManagedIdentity, identity)
		})
	}
}

func TestCreateManagedIdentityAlias(t *testing.T) {
	mockSubject := "mockSubject"

	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		Description:  "this is a managed identity being created",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Data:         []byte("some-data"),
		CreatedBy:    mockSubject,
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "some-group-id",
		},
		FullPath: "some/resource",
	}

	sampleAliasGroup := &models.Group{
		Metadata: models.ResourceMetadata{
			ID: "some-other-group-id",
		},
		FullPath: "some/sibling",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: &sampleAliasGroup.FullPath,
		Action:        models.ActionCreate,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      "some-new-alias-id",
	}

	sampleAliasName := "some-managed-identity-alias"

	type testCase struct {
		authError               error
		existingManagedIdentity *models.ManagedIdentity
		existingGroup           *models.Group
		expectCreatedAlias      *models.ManagedIdentity
		createInput             *models.ManagedIdentity
		input                   *CreateManagedIdentityAliasInput
		name                    string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name: "positive: successfully create a managed identity alias in a sibling group",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleAliasGroup,
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			createInput: &models.ManagedIdentity{
				GroupID:       sampleAliasGroup.Metadata.ID,
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
				CreatedBy:     mockSubject,
			},
			existingManagedIdentity: sampleManagedIdentity,
			existingGroup:           sampleGroup,
			expectCreatedAlias: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID: "some-new-alias-id",
				},
				Type:          sampleManagedIdentity.Type,
				ResourcePath:  sampleAliasGroup.FullPath + "/some-managed-identity-alias",
				Name:          sampleAliasName,
				Description:   sampleManagedIdentity.Description,
				GroupID:       sampleAliasGroup.Metadata.ID,
				CreatedBy:     mockSubject,
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
				Data:          sampleManagedIdentity.Data,
			},
		},
		{
			name: "negative: source managed identity doesn't exist",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleAliasGroup,
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			// Shouldn't happen.
			name: "negative: group associated with source managed identity doesn't exist",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleAliasGroup,
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			existingManagedIdentity: sampleManagedIdentity,
			expectErrorCode:         errors.EInternal,
		},
		{
			name: "negative: source managed identity is already available for namespace",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleGroup, // Using the same group here.
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			existingManagedIdentity: sampleManagedIdentity,
			existingGroup:           sampleGroup,
			expectErrorCode:         errors.EInvalid,
		},
		{
			name: "negative: attempting to alias another alias",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleAliasGroup,
				AliasSourceID: "some-alias-id",
				Name:          sampleAliasName,
			},
			existingManagedIdentity: &models.ManagedIdentity{
				AliasSourceID: &sampleManagedIdentity.Metadata.ID, // Only populated for aliases.
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: invalid name",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleGroup, // Using the same group here.
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          "some/invalid/name",
			},
			createInput: &models.ManagedIdentity{
				GroupID:       sampleAliasGroup.Metadata.ID,
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
				Name:          "some/invalid/name",
				CreatedBy:     mockSubject,
			},
			existingManagedIdentity: sampleManagedIdentity,
			existingGroup:           sampleGroup,
			expectErrorCode:         errors.EInvalid,
		},
		{
			name: "negative: subject does not have owner role in target group",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleGroup, // Using the same group here.
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
		{
			name: "negative: subject does not have owner role in source group",
			input: &CreateManagedIdentityAliasInput{
				Group:         sampleGroup, // Using the same group here.
				AliasSourceID: sampleManagedIdentity.Metadata.ID,
				Name:          sampleAliasName,
			},
			existingManagedIdentity: sampleManagedIdentity,
			existingGroup:           sampleGroup,
			authError:               errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:         errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockGroups := db.NewMockGroups(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("CreateManagedIdentity", mock.Anything, test.createInput).Return(test.expectCreatedAlias, nil).Maybe()
			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, mock.Anything).Return(test.existingManagedIdentity, nil).Maybe()

			mockGroups.On("GetGroupByID", mock.Anything, mock.Anything).Return(test.existingGroup, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			if test.existingGroup != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.existingGroup.Metadata.ID, models.OwnerRole).Return(test.authError)
			}

			mockCaller.On("RequireAccessToGroup", mock.Anything, test.input.Group.Metadata.ID, models.OwnerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Transactions:      mockTransactions,
				Groups:            mockGroups,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			alias, err := service.CreateManagedIdentityAlias(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCreatedAlias, alias)
		})
	}
}

func TestDeleteManagedIdentityAlias(t *testing.T) {
	sampleManagedIdentityAlias := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-alias-id",
		},
		Name:          "a-managed-identity-alias-to-delete",
		ResourcePath:  "some/resource/path",
		GroupID:       "some-group-id",
		AliasSourceID: ptr.String("some-source-managed-identity-id"),
	}

	sampleSourceIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-source-managed-identity-id",
		},
		GroupID: "some-group-id",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String("some/resource"),
		Action:        models.ActionDeleteChildResource,
		TargetType:    models.TargetGroup,
		TargetID:      sampleManagedIdentityAlias.GroupID,
		Payload: &models.ActivityEventDeleteChildResourcePayload{
			Name: sampleManagedIdentityAlias.Name,
			ID:   sampleManagedIdentityAlias.Metadata.ID,
			Type: string(models.TargetManagedIdentity),
		},
	}

	type testCase struct {
		authError                 error
		input                     *DeleteManagedIdentityInput
		sourceIdentity            *models.ManagedIdentity
		expectErrorCode           string
		name                      string
		managedIdentityWorkspaces []models.Workspace
	}

	testCases := []testCase{
		{
			name: "positive: successfully delete a managed identity alias",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentityAlias,
			},
		},
		{
			name: "positive: successfully delete a managed identity alias with force option",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentityAlias,
				Force:           true,
			},
		},
		{
			name: "negative: no force option and managed identity alias is assigned to a workspace",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentityAlias,
			},
			managedIdentityWorkspaces: []models.Workspace{{}},
			expectErrorCode:           errors.EConflict,
		},
		{
			name: "negative: attempting to delete a source managed identity",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleSourceIdentity,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: subject does not have owner role in alias' group",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentityAlias,
			},
			sourceIdentity: sampleSourceIdentity,
		},
		{
			name: "negative: subject does not have owner role in source group",
			input: &DeleteManagedIdentityInput{
				ManagedIdentity: sampleManagedIdentityAlias,
			},
			sourceIdentity:  sampleSourceIdentity,
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockWorkspaces := db.NewMockWorkspaces(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, sampleSourceIdentity.Metadata.ID).Return(test.sourceIdentity, nil).Maybe()
			mockManagedIdentities.On("DeleteManagedIdentity", mock.Anything, test.input.ManagedIdentity).Return(nil).Maybe()

			mockWorkspaces.On("GetWorkspacesForManagedIdentity", mock.Anything, sampleManagedIdentityAlias.Metadata.ID).Return(test.managedIdentityWorkspaces, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			if test.sourceIdentity != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.sourceIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			}

			mockCaller.On("RequireAccessToGroup", mock.Anything, test.input.ManagedIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Workspaces:        mockWorkspaces,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			err := service.DeleteManagedIdentityAlias(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateManagedIdentity(t *testing.T) {
	mockSubject := "mockSubject"

	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		Description:  "this is a managed identity being created",
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Data:         []byte("some-data"),
		CreatedBy:    mockSubject,
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleServiceAccount := &models.ServiceAccount{
		ResourcePath: "some/resource/service-account",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String(sampleManagedIdentity.GetGroupPath()),
		Action:        models.ActionCreate,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      sampleManagedIdentity.Metadata.ID,
	}

	createIdentityInput := &models.ManagedIdentity{
		Name:        "a-managed-identity",
		Description: "this is a managed identity being created",
		GroupID:     "some-group-id",
		CreatedBy:   mockSubject,
		Type:        models.ManagedIdentityAWSFederated,
	}

	createAccessRuleInput := &models.ManagedIdentityAccessRule{
		ManagedIdentityID:        sampleManagedIdentity.Metadata.ID,
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		RunStage:                 models.JobPlanType,
		AllowedUserIDs:           []string{"user-1-id", "user-2-id"},
		AllowedServiceAccountIDs: []string{"service-account-1-id"},
		AllowedTeamIDs:           []string{"team-1-id"},
	}

	type testCase struct {
		authError              error
		input                  *CreateManagedIdentityInput
		existingServiceAccount *models.ServiceAccount
		name                   string
		expectErrorCode        string
	}

	testCases := []testCase{
		{
			name: "positive: successfully create a managed identity",
			input: &CreateManagedIdentityInput{
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "a-managed-identity",
				Description: "this is a managed identity being created",
				GroupID:     "some-group-id",
				Data:        []byte("some-data"),
				AccessRules: []struct {
					Type                      models.ManagedIdentityAccessRuleType
					RunStage                  models.JobType
					ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
					AllowedUserIDs            []string
					AllowedServiceAccountIDs  []string
					AllowedTeamIDs            []string
				}{
					{
						Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
						RunStage:                 models.JobPlanType,
						AllowedUserIDs:           []string{"user-1-id", "user-2-id"},
						AllowedServiceAccountIDs: []string{"service-account-1-id"},
						AllowedTeamIDs:           []string{"team-1-id"},
					},
				},
			},
			existingServiceAccount: sampleServiceAccount,
		},
		{
			name: "negative: service account in access policy does not exist",
			input: &CreateManagedIdentityInput{
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "a-managed-identity",
				Description: "this is a managed identity being created",
				GroupID:     "some-group-id",
				Data:        []byte("some-data"),
				AccessRules: []struct {
					Type                      models.ManagedIdentityAccessRuleType
					RunStage                  models.JobType
					ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
					AllowedUserIDs            []string
					AllowedServiceAccountIDs  []string
					AllowedTeamIDs            []string
				}{
					{
						Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
						RunStage:                 models.JobPlanType,
						AllowedUserIDs:           []string{"user-1-id", "user-2-id"},
						AllowedServiceAccountIDs: []string{"non-existent-service-account"},
						AllowedTeamIDs:           []string{"team-1-id"},
					},
				},
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "negative: service account in access policy is outside group scope",
			input: &CreateManagedIdentityInput{
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "a-managed-identity",
				Description: "this is a managed identity being created",
				GroupID:     "some-group-id",
				Data:        []byte("some-data"),
				AccessRules: []struct {
					Type                      models.ManagedIdentityAccessRuleType
					RunStage                  models.JobType
					ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
					AllowedUserIDs            []string
					AllowedServiceAccountIDs  []string
					AllowedTeamIDs            []string
				}{
					{
						Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
						RunStage:                 models.JobPlanType,
						AllowedUserIDs:           []string{"user-1-id", "user-2-id"},
						AllowedServiceAccountIDs: []string{"outside-scope-1"},
						AllowedTeamIDs:           []string{"team-1-id"},
					},
				},
			},
			existingServiceAccount: &models.ServiceAccount{
				ResourcePath: "outside/scope/service-account",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: unsupported managed identity type",
			input: &CreateManagedIdentityInput{
				Type:    "unknown-type",
				GroupID: "some-group-id",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: managed identity has an invalid name",
			input: &CreateManagedIdentityInput{
				Type:    models.ManagedIdentityAWSFederated,
				Name:    "-invalid-name-",
				GroupID: "some-group-id",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: subject does not have owner role for group",
			input: &CreateManagedIdentityInput{
				Type:    models.ManagedIdentityAWSFederated,
				Name:    "a-managed-identity",
				GroupID: "some-group-id",
			},
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDelegate := NewMockDelegate(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("CreateManagedIdentity", mock.Anything, createIdentityInput).Return(sampleManagedIdentity, nil).Maybe()
			mockManagedIdentities.On("UpdateManagedIdentity", mock.Anything, sampleManagedIdentity).Return(sampleManagedIdentity, nil).Maybe()
			mockManagedIdentities.On("CreateManagedIdentityAccessRule", mock.Anything, createAccessRuleInput).Return(&models.ManagedIdentityAccessRule{}, nil).Maybe()

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, mock.Anything).Return(test.existingServiceAccount, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockDelegate.On("SetManagedIdentityData", mock.Anything, sampleManagedIdentity, sampleManagedIdentity.Data).Return(nil).Maybe()

			mockCaller.On("RequireAccessToGroup", mock.Anything, test.input.GroupID, models.OwnerRole).Return(test.authError)
			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				ServiceAccounts:   mockServiceAccounts,
				Transactions:      mockTransactions,
			}

			delegateMap := map[models.ManagedIdentityType]Delegate{
				models.ManagedIdentityAWSFederated:     mockDelegate,
				models.ManagedIdentityAzureFederated:   mockDelegate,
				models.ManagedIdentityTharsisFederated: mockDelegate,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, delegateMap, nil, nil, mockActivityEvents)

			identity, err := service.CreateManagedIdentity(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, sampleManagedIdentity, identity)
		})
	}
}

func TestGetManagedIdentitiesByIDs(t *testing.T) {
	sampleManagedIdentity := models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		GroupID: "some-group-id",
	}

	inputList := []string{
		"some-managed-identity-id",
	}

	type testCase struct {
		authError       error
		dbInput         *db.GetManagedIdentitiesInput
		expectResult    *db.ManagedIdentitiesResult
		name            string
		expectErrorCode string
		inputIDList     []string
	}

	testCases := []testCase{
		{
			name:        "positive: successfully return a list of managed identities",
			inputIDList: inputList,
			dbInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					ManagedIdentityIDs: inputList,
				},
			},
			expectResult: &db.ManagedIdentitiesResult{
				ManagedIdentities: []models.ManagedIdentity{sampleManagedIdentity},
			},
		},
		{
			name:        "negative: subject does not have access to group resource",
			inputIDList: inputList,
			dbInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					ManagedIdentityIDs: inputList,
				},
			},
			expectResult: &db.ManagedIdentitiesResult{
				ManagedIdentities: []models.ManagedIdentity{sampleManagedIdentity},
			},
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentities", mock.Anything, test.dbInput).Return(test.expectResult, nil)

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, sampleManagedIdentity.GroupID).Return(test.authError)

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			result, err := service.GetManagedIdentitiesByIDs(auth.WithCaller(ctx, mockCaller), test.inputIDList)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectResult.ManagedIdentities, result)
		})
	}
}

func TestUpdateManagedIdentity(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		Description:  "old-description",
		GroupID:      "some-group-id",
		Data:         []byte("this is old data"),
		Type:         models.ManagedIdentityAWSFederated,
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String(sampleManagedIdentity.GetGroupPath()),
		Action:        models.ActionUpdate,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      sampleManagedIdentity.Metadata.ID,
	}

	type testCase struct {
		authError               error
		existingManagedIdentity *models.ManagedIdentity
		expectManagedIdentity   *models.ManagedIdentity
		input                   *UpdateManagedIdentityInput
		name                    string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name: "positive: successfully update a managed identity",
			input: &UpdateManagedIdentityInput{
				ID:          "some-managed-identity-id",
				Description: "This is an updated description",
				Data:        []byte("this is new data"),
			},
			existingManagedIdentity: sampleManagedIdentity,
			expectManagedIdentity: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID: "some-managed-identity-id",
				},
				Name:         "a-managed-identity",
				ResourcePath: "some/resource/path",
				Description:  "This is an updated description",
				GroupID:      "some-group-id",
				Data:         []byte("this is new data"),
				Type:         models.ManagedIdentityAWSFederated,
			},
		},
		{
			name: "negative: updated description is too long",
			input: &UpdateManagedIdentityInput{
				ID:          "some-managed-identity-id",
				Description: strings.Repeat("really long description", 20),
				Data:        []byte("this is new data"),
			},
			expectErrorCode:         errors.EInvalid,
			existingManagedIdentity: sampleManagedIdentity,
		},
		{
			name: "negative: managed identity being updated doesn't exist",
			input: &UpdateManagedIdentityInput{
				ID:          "non-existent-id",
				Description: "This is an updated description",
				Data:        []byte("this is new data"),
			},
			expectErrorCode: errors.ENotFound,
		},
		{
			name: "negative: attempting to update a managed identity alias",
			input: &UpdateManagedIdentityInput{
				ID:          "some-managed-identity-id",
				Description: "This is an updated description",
				Data:        []byte("this is new data"),
			},
			existingManagedIdentity: &models.ManagedIdentity{
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "negative: subject does not have access to group",
			input: &UpdateManagedIdentityInput{
				ID:          "some-managed-identity-id",
				Description: "This is an updated description",
				Data:        []byte("this is new data"),
			},
			existingManagedIdentity: sampleManagedIdentity,
			authError:               errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:         errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDelegate := NewMockDelegate(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.input.ID).Return(test.existingManagedIdentity, nil)
			mockManagedIdentities.On("UpdateManagedIdentity", mock.Anything, test.existingManagedIdentity).Return(test.expectManagedIdentity, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			mockDelegate.On("SetManagedIdentityData", mock.Anything, test.existingManagedIdentity, test.input.Data).Return(nil).Maybe()

			if test.existingManagedIdentity != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.existingManagedIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			}

			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Transactions:      mockTransactions,
			}

			delegateMap := map[models.ManagedIdentityType]Delegate{
				models.ManagedIdentityAWSFederated:     mockDelegate,
				models.ManagedIdentityAzureFederated:   mockDelegate,
				models.ManagedIdentityTharsisFederated: mockDelegate,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, delegateMap, nil, nil, mockActivityEvents)

			identity, err := service.UpdateManagedIdentity(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectManagedIdentity, identity)
		})
	}
}

func TestGetManagedIdentityAccessRules(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		Name:         "a-managed-identity",
		ResourcePath: "some/resource/path",
		Description:  "old-description",
		GroupID:      "some-group-id",
		Data:         []byte("this is old data"),
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleAccessRules := []models.ManagedIdentityAccessRule{
		{
			Metadata: models.ResourceMetadata{
				ID: "some-access-rule",
			},
			Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
			RunStage:                 models.JobPlanType,
			ManagedIdentityID:        sampleManagedIdentity.Metadata.ID,
			AllowedUserIDs:           []string{"user-id-1"},
			AllowedServiceAccountIDs: []string{"service-account-id-1"},
			AllowedTeamIDs:           []string{"team-id-1"},
		},
	}

	type testCase struct {
		authError         error
		input             *models.ManagedIdentity
		dbInput           *db.GetManagedIdentityAccessRulesInput
		dbResult          *db.ManagedIdentityAccessRulesResult
		name              string
		expectErrorCode   string
		expectAccessRules []models.ManagedIdentityAccessRule
	}

	testCases := []testCase{
		{
			name:  "positive: successfully return managed identity access rules",
			input: sampleManagedIdentity,
			dbInput: &db.GetManagedIdentityAccessRulesInput{
				Filter: &db.ManagedIdentityAccessRuleFilter{
					ManagedIdentityID: &sampleManagedIdentity.Metadata.ID,
				},
			},
			expectAccessRules: sampleAccessRules,
			dbResult: &db.ManagedIdentityAccessRulesResult{
				ManagedIdentityAccessRules: sampleAccessRules,
			},
		},
		{
			name:  "negative: subject does have access to group resource",
			input: sampleManagedIdentity,
			dbInput: &db.GetManagedIdentityAccessRulesInput{
				Filter: &db.ManagedIdentityAccessRuleFilter{
					ManagedIdentityID: &sampleManagedIdentity.Metadata.ID,
				},
			},
			authError: errors.NewError(errors.EForbidden, "Forbidden"),
			dbResult: &db.ManagedIdentityAccessRulesResult{
				ManagedIdentityAccessRules: sampleAccessRules,
			},
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityAccessRules", mock.Anything, test.dbInput).Return(test.dbResult, nil).Maybe()

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, test.input.GroupID).Return(test.authError)

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			rules, err := service.GetManagedIdentityAccessRules(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAccessRules, rules)
		})
	}
}

func TestGetManagedIdentityAccessRulesByIDs(t *testing.T) {
	sampleAccessRules := []models.ManagedIdentityAccessRule{
		{
			Metadata: models.ResourceMetadata{
				ID: "an-access-rule-1",
			},
			Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
			RunStage:          models.JobApplyType,
			ManagedIdentityID: "some-managed-identity-id",
			AllowedUserIDs:    []string{"some-user-1"},
		},
	}

	sampleIdentitiesResult := &db.ManagedIdentitiesResult{
		ManagedIdentities: []models.ManagedIdentity{
			{
				Metadata: models.ResourceMetadata{
					ID: "some-managed-identity-id",
				},
				GroupID: "some-group-id",
			},
		},
	}

	idList := []string{"access-rule-id-1", "access-rule-id-2"}

	type testCase struct {
		authError         error
		name              string
		expectErrorCode   string
		inputIDList       []string
		ruleDBInput       *db.GetManagedIdentityAccessRulesInput
		identityDBInput   *db.GetManagedIdentitiesInput
		dbResult          *db.ManagedIdentityAccessRulesResult
		expectAccessRules []models.ManagedIdentityAccessRule
	}

	testCases := []testCase{
		{
			name:        "positive: successfully return managed identity access rules",
			inputIDList: idList,
			ruleDBInput: &db.GetManagedIdentityAccessRulesInput{
				Filter: &db.ManagedIdentityAccessRuleFilter{
					ManagedIdentityAccessRuleIDs: idList,
				},
			},
			identityDBInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					ManagedIdentityIDs: []string{sampleAccessRules[0].ManagedIdentityID},
				},
			},
			dbResult: &db.ManagedIdentityAccessRulesResult{
				ManagedIdentityAccessRules: sampleAccessRules,
			},
			expectAccessRules: sampleAccessRules,
		},
		{
			name:        "negative: subject does not have access to group resource",
			inputIDList: idList,
			ruleDBInput: &db.GetManagedIdentityAccessRulesInput{
				Filter: &db.ManagedIdentityAccessRuleFilter{
					ManagedIdentityAccessRuleIDs: idList,
				},
			},
			identityDBInput: &db.GetManagedIdentitiesInput{
				Filter: &db.ManagedIdentityFilter{
					ManagedIdentityIDs: []string{sampleAccessRules[0].ManagedIdentityID},
				},
			},
			dbResult: &db.ManagedIdentityAccessRulesResult{
				ManagedIdentityAccessRules: sampleAccessRules,
			},
			authError:       errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode: errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityAccessRules", mock.Anything, test.ruleDBInput).Return(test.dbResult, nil)
			mockManagedIdentities.On("GetManagedIdentities", mock.Anything, test.identityDBInput).Return(sampleIdentitiesResult, nil)

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, mock.Anything).Return(test.authError)

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			rules, err := service.GetManagedIdentityAccessRulesByIDs(auth.WithCaller(ctx, mockCaller), test.inputIDList)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAccessRules, rules)
		})
	}
}

func TestGetManagedIdentityAccessRule(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		GroupID: "some-group-id",
		Type:    models.ManagedIdentityAWSFederated,
	}

	sampleAccessRule := &models.ManagedIdentityAccessRule{
		Metadata: models.ResourceMetadata{
			ID: "some-access-rule",
		},
		Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
		RunStage:          models.JobPlanType,
		ManagedIdentityID: sampleManagedIdentity.Metadata.ID,
		AllowedUserIDs:    []string{"user-id-1"},
	}

	type testCase struct {
		authError        error
		expectAccessRule *models.ManagedIdentityAccessRule
		searchID         string
		name             string
		expectErrorCode  string
	}

	testCases := []testCase{
		{
			name:             "positive: successfully return a managed identity access rule",
			expectAccessRule: sampleAccessRule,
			searchID:         sampleAccessRule.Metadata.ID,
		},
		{
			name:            "negative: access rule doesn't exist",
			expectErrorCode: errors.ENotFound,
			searchID:        "unknown-access-rule-id",
		},
		{
			name:             "negative: subject does not have access to group resource",
			searchID:         sampleAccessRule.Metadata.ID,
			expectAccessRule: sampleAccessRule,
			authError:        errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:  errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityAccessRule", mock.Anything, test.searchID).Return(test.expectAccessRule, nil)
			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, sampleManagedIdentity.Metadata.ID).Return(sampleManagedIdentity, nil).Maybe()

			mockCaller.On("RequireAccessToInheritedGroupResource", mock.Anything, mock.Anything).Return(test.authError).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			service := NewService(nil, dbClient, nil, nil, nil, nil)

			rule, err := service.GetManagedIdentityAccessRule(auth.WithCaller(ctx, mockCaller), test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAccessRule, rule)
		})
	}
}

func TestCreateManagedIdentityAccessRule(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleAccessRule := &models.ManagedIdentityAccessRule{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-access-rule-id",
		},
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		RunStage:                 models.JobApplyType,
		ManagedIdentityID:        sampleManagedIdentity.Metadata.ID,
		AllowedUserIDs:           []string{"user-id-1"},
		AllowedServiceAccountIDs: []string{"service-account-id-1"},
		AllowedTeamIDs:           []string{"team-id-1"},
	}

	sampleServiceAccount := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID: "service-account-id-1",
		},
		ResourcePath: "some/resource/service-account",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String(sampleManagedIdentity.GetGroupPath()),
		Action:        models.ActionCreate,
		TargetType:    models.TargetManagedIdentityAccessRule,
		TargetID:      sampleAccessRule.Metadata.ID,
	}

	type testCase struct {
		authError               error
		expectAccessRule        *models.ManagedIdentityAccessRule
		existingServiceAccount  *models.ServiceAccount
		existingManagedIdentity *models.ManagedIdentity
		input                   *models.ManagedIdentityAccessRule
		name                    string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name:                    "positive: successfully create a managed identity access rule",
			existingManagedIdentity: sampleManagedIdentity,
			existingServiceAccount:  sampleServiceAccount,
			expectAccessRule:        sampleAccessRule,
			input:                   sampleAccessRule,
		},
		{
			name:                    "negative: allowed service account doesn't exist",
			existingManagedIdentity: sampleManagedIdentity,
			input:                   sampleAccessRule,
			expectErrorCode:         errors.ENotFound,
		},
		{
			name:            "negative: managed identity associated with rules doesn't exist",
			input:           sampleAccessRule,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:                    "negative: service account is out of group scope",
			input:                   sampleAccessRule,
			existingManagedIdentity: sampleManagedIdentity,
			existingServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID: "service-account-id-1",
				},
				ResourcePath: "out/of/scope/service-account",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "negative: attempting to create access rule for a managed identity alias",
			input: sampleAccessRule,
			existingManagedIdentity: &models.ManagedIdentity{
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                    "negative: subject does not have owner role for group",
			input:                   sampleAccessRule,
			existingManagedIdentity: sampleManagedIdentity,
			authError:               errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:         errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.input.ManagedIdentityID).Return(test.existingManagedIdentity, nil)
			mockManagedIdentities.On("CreateManagedIdentityAccessRule", mock.Anything, test.input).Return(test.expectAccessRule, nil).Maybe()

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(test.existingServiceAccount, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			if test.existingManagedIdentity != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.existingManagedIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			}

			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				ServiceAccounts:   mockServiceAccounts,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			accessRule, err := service.CreateManagedIdentityAccessRule(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAccessRule, accessRule)
		})
	}
}

func TestUpdateManagedIdentityAccessRule(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleAccessRule := &models.ManagedIdentityAccessRule{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-access-rule-id",
		},
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		RunStage:                 models.JobApplyType,
		ManagedIdentityID:        sampleManagedIdentity.Metadata.ID,
		AllowedUserIDs:           []string{"user-id-1"},
		AllowedServiceAccountIDs: []string{"service-account-id-1"},
		AllowedTeamIDs:           []string{"team-id-1"},
	}

	sampleServiceAccount := &models.ServiceAccount{
		Metadata: models.ResourceMetadata{
			ID: "service-account-id-1",
		},
		ResourcePath: "some/resource/service-account",
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String(sampleManagedIdentity.GetGroupPath()),
		Action:        models.ActionUpdate,
		TargetType:    models.TargetManagedIdentityAccessRule,
		TargetID:      sampleAccessRule.Metadata.ID,
	}

	type testCase struct {
		authError               error
		expectAccessRule        *models.ManagedIdentityAccessRule
		existingServiceAccount  *models.ServiceAccount
		existingManagedIdentity *models.ManagedIdentity
		input                   *models.ManagedIdentityAccessRule
		name                    string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name:                    "positive: successfully update a managed identity access rule",
			existingManagedIdentity: sampleManagedIdentity,
			existingServiceAccount:  sampleServiceAccount,
			expectAccessRule:        sampleAccessRule,
			input:                   sampleAccessRule,
		},
		{
			name:                    "negative: allowed service account doesn't exist",
			existingManagedIdentity: sampleManagedIdentity,
			input:                   sampleAccessRule,
			expectErrorCode:         errors.ENotFound,
		},
		{
			name:                    "negative: service account is out of group scope",
			input:                   sampleAccessRule,
			existingManagedIdentity: sampleManagedIdentity,
			existingServiceAccount: &models.ServiceAccount{
				Metadata: models.ResourceMetadata{
					ID: "service-account-id-1",
				},
				ResourcePath: "out/of/scope/service-account",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "negative: attempting to update access rules for a managed identity alias",
			input: sampleAccessRule,
			existingManagedIdentity: &models.ManagedIdentity{
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                    "negative: subject does not have owner role for group",
			input:                   sampleAccessRule,
			existingManagedIdentity: sampleManagedIdentity,
			authError:               errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:         errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockServiceAccounts := db.NewMockServiceAccounts(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.input.ManagedIdentityID).Return(test.existingManagedIdentity, nil)
			mockManagedIdentities.On("UpdateManagedIdentityAccessRule", mock.Anything, test.input).Return(test.expectAccessRule, nil).Maybe()

			mockServiceAccounts.On("GetServiceAccountByID", mock.Anything, sampleServiceAccount.Metadata.ID).Return(test.existingServiceAccount, nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			if test.existingManagedIdentity != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.existingManagedIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			}

			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				ServiceAccounts:   mockServiceAccounts,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			accessRule, err := service.UpdateManagedIdentityAccessRule(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectAccessRule, accessRule)
		})
	}
}

func TestDeleteManagedIdentityAccessRule(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleAccessRule := &models.ManagedIdentityAccessRule{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-access-rule-id",
		},
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		RunStage:                 models.JobApplyType,
		ManagedIdentityID:        sampleManagedIdentity.Metadata.ID,
		AllowedUserIDs:           []string{"user-id-1"},
		AllowedServiceAccountIDs: []string{"service-account-id-1"},
		AllowedTeamIDs:           []string{"team-id-1"},
	}

	activityEventInput := &activityevent.CreateActivityEventInput{
		NamespacePath: ptr.String(sampleManagedIdentity.GetGroupPath()),
		Action:        models.ActionDeleteChildResource,
		TargetType:    models.TargetManagedIdentity,
		TargetID:      sampleManagedIdentity.Metadata.ID,
		Payload: &models.ActivityEventDeleteChildResourcePayload{
			ID:   sampleAccessRule.Metadata.ID,
			Name: string(sampleAccessRule.RunStage),
			Type: string(models.TargetManagedIdentityAccessRule),
		},
	}

	type testCase struct {
		authError               error
		existingManagedIdentity *models.ManagedIdentity
		input                   *models.ManagedIdentityAccessRule
		name                    string
		expectErrorCode         string
	}

	testCases := []testCase{
		{
			name:                    "positive: successfully delete a managed identity access rule",
			existingManagedIdentity: sampleManagedIdentity,
			input:                   sampleAccessRule,
		},
		{
			name:            "negative: managed identity associated with access rule doesn't exist",
			input:           sampleAccessRule,
			expectErrorCode: errors.ENotFound,
		},
		{
			name:  "negative: attempting to delete access rule for a managed identity alias",
			input: sampleAccessRule,
			existingManagedIdentity: &models.ManagedIdentity{
				AliasSourceID: &sampleManagedIdentity.Metadata.ID,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:                    "negative: subject does not have owner role for group",
			input:                   sampleAccessRule,
			existingManagedIdentity: sampleManagedIdentity,
			authError:               errors.NewError(errors.EForbidden, "Forbidden"),
			expectErrorCode:         errors.EForbidden,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockActivityEvents := activityevent.NewMockService(t)
			mockTransactions := db.NewMockTransactions(t)
			mockCaller := auth.NewMockCaller(t)

			mockManagedIdentities.On("GetManagedIdentityByID", mock.Anything, test.input.ManagedIdentityID).Return(test.existingManagedIdentity, nil)
			mockManagedIdentities.On("DeleteManagedIdentityAccessRule", mock.Anything, test.input).Return(nil).Maybe()

			mockActivityEvents.On("CreateActivityEvent", mock.Anything, activityEventInput).Return(&models.ActivityEvent{}, nil).Maybe()

			mockTransactions.On("BeginTx", mock.Anything).Return(ctx, nil).Maybe()
			mockTransactions.On("RollbackTx", mock.Anything).Return(nil).Maybe()
			mockTransactions.On("CommitTx", mock.Anything).Return(nil).Maybe()

			if test.existingManagedIdentity != nil {
				mockCaller.On("RequireAccessToGroup", mock.Anything, test.existingManagedIdentity.GroupID, models.OwnerRole).Return(test.authError).Maybe()
			}

			mockCaller.On("GetSubject").Return("mockSubject").Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
				Transactions:      mockTransactions,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, nil, nil, nil, mockActivityEvents)

			err := service.DeleteManagedIdentityAccessRule(auth.WithCaller(ctx, mockCaller), test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	sampleManagedIdentity := &models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "some-managed-identity-id",
		},
		ResourcePath: "some/resource/path",
		GroupID:      "some-group-id",
		Type:         models.ManagedIdentityAWSFederated,
	}

	sampleJob := &models.Job{
		Metadata: models.ResourceMetadata{
			ID: "some-job-id",
		},
		WorkspaceID: "some-workspace-id",
	}

	type testCase struct {
		caller                    auth.Caller
		input                     *models.ManagedIdentity
		existingManagedIdentities []models.ManagedIdentity
		name                      string
		expectErrorCode           string
		expectCredentials         []byte
	}

	testCases := []testCase{
		{
			name: "positive: successfully create managed identity credentials",
			caller: &auth.JobCaller{
				JobID:       sampleJob.Metadata.ID,
				WorkspaceID: sampleJob.WorkspaceID,
			},
			input:                     sampleManagedIdentity,
			existingManagedIdentities: []models.ManagedIdentity{*sampleManagedIdentity},
			expectCredentials:         []byte("some-credentials"),
		},
		{
			name: "negative: managed identities don't belong to respective workspace",
			caller: &auth.JobCaller{
				JobID:       sampleJob.Metadata.ID,
				WorkspaceID: sampleJob.WorkspaceID,
			},
			input:                     sampleManagedIdentity,
			existingManagedIdentities: []models.ManagedIdentity{},
			expectErrorCode:           errors.EUnauthorized,
		},
		{
			name:            "negative: not a job caller",
			caller:          &auth.UserCaller{},
			input:           sampleManagedIdentity,
			expectErrorCode: errors.EForbidden,
		},
		{
			name:            "negative: no caller",
			input:           sampleManagedIdentity,
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ctx := auth.WithCaller(context.Background(), test.caller)

			mockManagedIdentities := db.NewMockManagedIdentities(t)
			mockJobService := job.NewMockService(t)
			mockDelegate := NewMockDelegate(t)

			mockManagedIdentities.On("GetManagedIdentitiesForWorkspace", mock.Anything, sampleJob.WorkspaceID).Return(test.existingManagedIdentities, nil).Maybe()

			mockJobService.On("GetJob", mock.Anything, mock.Anything).Return(sampleJob, nil).Maybe()

			mockDelegate.On("CreateCredentials", mock.Anything, test.input, sampleJob).Return([]byte("some-credentials"), nil).Maybe()

			dbClient := &db.Client{
				ManagedIdentities: mockManagedIdentities,
			}

			delegateMap := map[models.ManagedIdentityType]Delegate{
				models.ManagedIdentityAWSFederated:     mockDelegate,
				models.ManagedIdentityAzureFederated:   mockDelegate,
				models.ManagedIdentityTharsisFederated: mockDelegate,
			}

			logger, _ := logger.NewForTest()
			service := NewService(logger, dbClient, delegateMap, nil, mockJobService, nil)

			credentials, err := service.CreateCredentials(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, test.expectCredentials, credentials)
		})
	}
}