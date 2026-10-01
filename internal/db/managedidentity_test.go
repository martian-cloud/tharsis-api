//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// getValue implements the sortableField interface for ManagedIdentitySortableField
func (sf ManagedIdentitySortableField) getValue() string {
	return string(sf)
}

func TestGetManagedIdentityByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name                  string
		searchID              string
		expectManagedIdentity *models.ManagedIdentity
		expectErrorCode       errors.CodeType
	}

	testCases := []testCase{
		{
			name:                  "get resource by id",
			searchID:              managedIdentity1.Metadata.ID,
			expectManagedIdentity: managedIdentity1,
		},
		{
			name:                  "get a managed identity alias by id",
			searchID:              createdAlias.Metadata.ID,
			expectManagedIdentity: createdAlias,
		},
		{
			name:     "resource with id not found",
			searchID: nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			searchID:        invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			managedIdentity, err := testClient.client.ManagedIdentities.GetManagedIdentityByID(ctx, test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectManagedIdentity != nil {
				require.NotNil(t, managedIdentity)
				assert.Equal(t, test.expectManagedIdentity.Metadata.ID, managedIdentity.Metadata.ID)
				assert.Equal(t, test.expectManagedIdentity.Name, managedIdentity.Name)
				assert.Equal(t, test.expectManagedIdentity.GroupID, managedIdentity.GroupID)
			} else {
				assert.Nil(t, managedIdentity)
			}
		})
	}
}

func TestGetManagedIdentityByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	managedIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:    "test-managed-identity",
		GroupID: group.Metadata.ID,
		Type:    models.ManagedIdentityAWSFederated,
		Data:    []byte("test-data"),
	})
	require.NoError(t, err)

	type testCase struct {
		name                  string
		trn                   string
		expectManagedIdentity bool
		expectErrorCode       errors.CodeType
	}

	testCases := []testCase{
		{
			name:                  "get resource by TRN",
			trn:                   managedIdentity.Metadata.TRN,
			expectManagedIdentity: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeManagedIdentity.Build(group.FullPath, "unknown"),
		},
		{
			name:            "managed identity trn must have two parts",
			trn:             trn.TypeManagedIdentity.Build("unknown"),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualManagedIdentity, err := testClient.client.ManagedIdentities.GetManagedIdentityByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectManagedIdentity {
				require.NotNil(t, actualManagedIdentity)
				assert.Equal(t, trn.TypeManagedIdentity.Build(group.FullPath, managedIdentity.Name), actualManagedIdentity.Metadata.TRN)
			} else {
				assert.Nil(t, actualManagedIdentity)
			}
		})
	}
}

func TestGetManagedIdentitiesForWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: ptr.Int32(int32(forTestMaxJobDuration.Minutes())),
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.NoError(t, err)

	t.Run("not added to workspace returns empty", func(t *testing.T) {
		managedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspace1.Metadata.ID)
		require.NoError(t, err)
		assert.Empty(t, managedIdentities)
	})

	t.Run("non-existent workspace id returns empty", func(t *testing.T) {
		managedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, nonExistentID)
		require.NoError(t, err)
		assert.Empty(t, managedIdentities)
	})

	require.NoError(t, testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID))
	require.NoError(t, testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, createdAlias.Metadata.ID, workspace1.Metadata.ID))

	t.Run("returns the managed identities added to the workspace, including aliases", func(t *testing.T) {
		managedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspace1.Metadata.ID)
		require.NoError(t, err)

		gotIDs := make([]string, len(managedIdentities))
		for i, mi := range managedIdentities {
			gotIDs[i] = mi.Metadata.ID
		}
		assert.ElementsMatch(t, []string{managedIdentity1.Metadata.ID, createdAlias.Metadata.ID}, gotIDs)
	})
}

func TestAddManagedIdentityToWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: ptr.Int32(int32(forTestMaxJobDuration.Minutes())),
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	t.Run("add managed identity to workspace", func(t *testing.T) {
		require.NoError(t, testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID))

		managedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspace1.Metadata.ID)
		require.NoError(t, err)
		require.Len(t, managedIdentities, 1)
		assert.Equal(t, managedIdentity1.Metadata.ID, managedIdentities[0].Metadata.ID)
	})

	t.Run("adding the same managed identity again is rejected", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID)
		assert.Equal(t, errors.EConflict, errors.ErrorCode(err))
	})

	type testCase struct {
		name              string
		managedIdentityID string
		workspaceID       string
	}

	testCases := []testCase{
		{
			name:              "non-existent workspace id",
			managedIdentityID: managedIdentity1.Metadata.ID,
			workspaceID:       nonExistentID,
		},
		{
			name:              "invalid workspace id",
			managedIdentityID: managedIdentity1.Metadata.ID,
			workspaceID:       invalidID,
		},
		{
			name:              "non-existent managed identity id",
			managedIdentityID: nonExistentID,
			workspaceID:       workspace1.Metadata.ID,
		},
		{
			name:              "invalid managed identity id",
			managedIdentityID: invalidID,
			workspaceID:       workspace1.Metadata.ID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, test.managedIdentityID, test.workspaceID)
			assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
		})
	}
}

func TestRemoveManagedIdentityFromWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: ptr.Int32(int32(forTestMaxJobDuration.Minutes())),
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	require.NoError(t, testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID))

	t.Run("remove managed identity from workspace", func(t *testing.T) {
		require.NoError(t, testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID))

		managedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, workspace1.Metadata.ID)
		require.NoError(t, err)
		assert.Empty(t, managedIdentities)
	})

	t.Run("removing again is a no-op", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, managedIdentity1.Metadata.ID, workspace1.Metadata.ID)
		assert.NoError(t, err)
	})

	t.Run("non-existent workspace id is a no-op", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, managedIdentity1.Metadata.ID, nonExistentID)
		assert.NoError(t, err)
	})

	t.Run("invalid workspace id returns an error", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, managedIdentity1.Metadata.ID, invalidID)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("non-existent managed identity id is a no-op", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, nonExistentID, workspace1.Metadata.ID)
		assert.NoError(t, err)
	})

	t.Run("invalid managed identity id returns an error", func(t *testing.T) {
		err := testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx, invalidID, workspace1.Metadata.ID)
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

func TestCreateManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	aliasGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "top-level-group-1-for-managed-identity-aliases",
		CreatedBy: "someone-g1",
	})
	require.NoError(t, err)

	// Create a managed identity prior to running tests so an alias can use it.
	aliasSourceIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Type:        models.ManagedIdentityAWSFederated,
		Name:        "a-managed-identity-for-testing-aliases",
		Description: "a description for this managed identity",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "creator-of-managed-identities",
		Data:        []byte("some-data-for-the-source-managed-identity"),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		toCreate        *models.ManagedIdentity
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "create a managed identity",
			toCreate: &models.ManagedIdentity{
				Name:        "positive-create-managed-identity",
				Description: "a managed identity created for testing",
				GroupID:     group1.Metadata.ID,
				Type:        models.ManagedIdentityAWSFederated,
				Data:        []byte("some-data"),
				CreatedBy:   "creator-of-managed-identities",
			},
		},
		{
			name: "create a managed identity alias",
			toCreate: &models.ManagedIdentity{
				Name:          "positive-create-managed-identity-alias",
				GroupID:       aliasGroup.Metadata.ID,
				AliasSourceID: &aliasSourceIdentity.Metadata.ID,
				CreatedBy:     "creator-of-managed-identities",
			},
		},
		{
			name: "duplicate name in same group",
			toCreate: &models.ManagedIdentity{
				Name:      "positive-create-managed-identity",
				GroupID:   group1.Metadata.ID,
				Type:      models.ManagedIdentityAWSFederated,
				Data:      []byte("some-data"),
				CreatedBy: "creator-of-managed-identities",
			},
			expectErrorCode: errors.EConflict,
		},
		{
			name: "non-existent group id",
			toCreate: &models.ManagedIdentity{
				Name:    "non-existent-group-id",
				GroupID: nonExistentID,
				Type:    models.ManagedIdentityAzureFederated,
				Data:    []byte("some-data"),
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid group id",
			toCreate: &models.ManagedIdentity{
				Name:    "invalid-group-id",
				GroupID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, created)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)

			assert.Equal(t, initialResourceVersion, created.Metadata.Version)
			assert.NotEmpty(t, created.Metadata.TRN)
			assert.Equal(t, test.toCreate.Name, created.Name)
			assert.Equal(t, test.toCreate.GroupID, created.GroupID)

			if test.toCreate.AliasSourceID != nil {
				// An alias backfills its type, description, and data from the source managed identity.
				assert.Equal(t, aliasSourceIdentity.Type, created.Type)
				assert.Equal(t, aliasSourceIdentity.Description, created.Description)
				assert.Equal(t, aliasSourceIdentity.Data, created.Data)
				assert.Equal(t, test.toCreate.AliasSourceID, created.AliasSourceID)
			} else {
				assert.Equal(t, test.toCreate.Type, created.Type)
				assert.Equal(t, test.toCreate.Description, created.Description)
				assert.Equal(t, test.toCreate.Data, created.Data)
			}
		})
	}
}

func TestUpdateManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	otherGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "top-level-group-1-for-managed-identities",
		CreatedBy: "someone-g1",
	})
	require.NoError(t, err)

	createdManagedIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "managed-identity-0",
		Description: "original description",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	t.Run("update description, data, and move to another group", func(t *testing.T) {
		toUpdate := *createdManagedIdentity
		toUpdate.Description = "updated description"
		toUpdate.Data = []byte("updated data")
		toUpdate.GroupID = otherGroup.Metadata.ID

		updated, err := testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, &toUpdate)
		require.NoError(t, err)
		require.NotNil(t, updated)

		assert.Equal(t, "updated description", updated.Description)
		assert.Equal(t, []byte("updated data"), updated.Data)
		assert.Equal(t, otherGroup.Metadata.ID, updated.GroupID)
		assert.Equal(t, createdManagedIdentity.Metadata.Version+1, updated.Metadata.Version)
		assert.Equal(t, trn.TypeManagedIdentity.Build(otherGroup.FullPath+"/"+createdManagedIdentity.Name), updated.Metadata.TRN)
	})

	t.Run("non-existent id", func(t *testing.T) {
		_, err := testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, &models.ManagedIdentity{
			Metadata: models.ResourceMetadata{
				ID:      nonExistentID,
				Version: createdManagedIdentity.Metadata.Version,
			},
		})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})

	t.Run("invalid id", func(t *testing.T) {
		_, err := testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, &models.ManagedIdentity{
			Metadata: models.ResourceMetadata{
				ID:      invalidID,
				Version: createdManagedIdentity.Metadata.Version,
			},
		})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

func TestDeleteManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	createdManagedIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		version         int
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "delete managed identity",
			id:      createdManagedIdentity.Metadata.ID,
			version: createdManagedIdentity.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdManagedIdentity.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "non-existent id",
			id:              nonExistentID,
			expectErrorCode: errors.EOptimisticLock,
		},
		{
			name:            "invalid id",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ManagedIdentities.DeleteManagedIdentity(ctx, &models.ManagedIdentity{
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

			managedIdentity, err := testClient.client.ManagedIdentities.GetManagedIdentityByID(ctx, test.id)
			require.NoError(t, err)
			assert.Nil(t, managedIdentity)
		})
	}
}

func TestGetManagedIdentities(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "top-level-group-1-for-managed-identities",
		CreatedBy: "someone-g1",
	})
	require.NoError(t, err)

	apple, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "identity-apple",
		GroupID:   group0.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("apple-data"),
	})
	require.NoError(t, err)

	banana, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "identity-banana",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa1",
		Type:      models.ManagedIdentityAzureFederated,
		Data:      []byte("banana-data"),
	})
	require.NoError(t, err)

	alias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "identity-apple-alias",
		GroupID:       group0.Metadata.ID,
		CreatedBy:     "someone-ma0",
		AliasSourceID: &apple.Metadata.ID,
	})
	require.NoError(t, err)

	allIDs := []string{apple.Metadata.ID, banana.Metadata.ID, alias.Metadata.ID}

	type testCase struct {
		name      string
		input     *GetManagedIdentitiesInput
		expectIDs []string
	}

	testCases := []testCase{
		{
			name:      "no filter returns all",
			input:     &GetManagedIdentitiesInput{},
			expectIDs: allIDs,
		},
		{
			name: "filter, search field, empty string matches everything",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{Search: ptr.String("")},
			},
			expectIDs: allIDs,
		},
		{
			name: "filter, search field, matches by name prefix",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{Search: ptr.String("identity-apple")},
			},
			expectIDs: []string{apple.Metadata.ID, alias.Metadata.ID},
		},
		{
			name: "filter, search field, matches an alias by resource path",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{Search: ptr.String(alias.GetResourcePath())},
			},
			expectIDs: []string{alias.Metadata.ID},
		},
		{
			name: "filter, search field, no match",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{Search: ptr.String("bogus")},
			},
			expectIDs: []string{},
		},
		{
			name: "filter, namespace paths",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{NamespacePaths: []string{group1.FullPath}},
			},
			expectIDs: []string{banana.Metadata.ID},
		},
		{
			name: "filter, alias source id",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{AliasSourceID: &apple.Metadata.ID},
			},
			expectIDs: []string{alias.Metadata.ID},
		},
		{
			name: "filter, managed identity ids",
			input: &GetManagedIdentitiesInput{
				Filter: &ManagedIdentityFilter{ManagedIdentityIDs: []string{apple.Metadata.ID, banana.Metadata.ID}},
			},
			expectIDs: []string{apple.Metadata.ID, banana.Metadata.ID},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.ManagedIdentities.GetManagedIdentities(ctx, test.input)
			require.NoError(t, err)

			gotIDs := make([]string, len(result.ManagedIdentities))
			for i, mi := range result.ManagedIdentities {
				gotIDs[i] = mi.Metadata.ID
			}
			assert.ElementsMatch(t, test.expectIDs, gotIDs)
		})
	}
}

func TestGetManagedIdentitiesWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:      "test-group-managed-identity-pagination",
		CreatedBy: "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		_, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
			Name:      fmt.Sprintf("test-managed-identity-pagination-%d", i),
			GroupID:   group.Metadata.ID,
			CreatedBy: "db-integration-tests",
			Type:      models.ManagedIdentityAWSFederated,
			Data:      []byte("some-data"),
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		ManagedIdentitySortableFieldCreatedAtAsc,
		ManagedIdentitySortableFieldCreatedAtDesc,
		ManagedIdentitySortableFieldUpdatedAtAsc,
		ManagedIdentitySortableFieldUpdatedAtDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := ManagedIdentitySortableField(sortByField.getValue())

		result, err := testClient.client.ManagedIdentities.GetManagedIdentities(ctx, &GetManagedIdentitiesInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for i := range result.ManagedIdentities {
			resources = append(resources, &result.ManagedIdentities[i])
		}

		return result.PageInfo, resources, nil
	})
}

func TestGetManagedIdentityAccessRules(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.NoError(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:          models.JobPlanType,
		Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID: managedIdentity1.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		searchID        string
		expectIDs       []string
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:      "positive",
			searchID:  managedIdentity1.Metadata.ID,
			expectIDs: []string{createdRule.Metadata.ID},
		},
		{
			name:      "successfully retrieve access rules for a managed identity alias",
			searchID:  createdAlias.Metadata.ID,
			expectIDs: []string{createdRule.Metadata.ID},
		},
		{
			name:      "non-existent id",
			searchID:  nonExistentID,
			expectIDs: []string{},
		},
		{
			name:            "invalid id",
			searchID:        invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &GetManagedIdentityAccessRulesInput{
				Filter: &ManagedIdentityAccessRuleFilter{
					ManagedIdentityID: &test.searchID,
				},
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			gotIDs := make([]string, len(result.ManagedIdentityAccessRules))
			for i, rule := range result.ManagedIdentityAccessRules {
				gotIDs[i] = rule.Metadata.ID
			}
			assert.ElementsMatch(t, test.expectIDs, gotIDs)
		})
	}
}

func TestGetManagedIdentityAccessRuleByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
	})
	require.NoError(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:          models.JobPlanType,
		Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID: managedIdentity1.Metadata.ID,
		AllowedUserIDs:    []string{user1.Metadata.ID},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		searchID        string
		expectRule      bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			searchID:   createdRule.Metadata.ID,
			expectRule: true,
		},
		{
			name:     "resource with id not found",
			searchID: nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			searchID:        invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rule, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRuleByID(ctx, test.searchID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectRule {
				require.NotNil(t, rule)
				assert.Equal(t, createdRule.Metadata.ID, rule.Metadata.ID)
				assert.Equal(t, createdRule.AllowedUserIDs, rule.AllowedUserIDs)
			} else {
				assert.Nil(t, rule)
			}
		})
	}
}

func TestGetManagedIdentityAccessRuleByTRN(t *testing.T) {
	ctx := t.Context()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name: "test-group",
	})
	require.NoError(t, err)

	managedIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:    "test-managed-identity",
		GroupID: group.Metadata.ID,
		Type:    models.ManagedIdentityAWSFederated,
		Data:    []byte("test-data"),
	})
	require.NoError(t, err)

	accessRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:          models.JobPlanType,
		ManagedIdentityID: managedIdentity.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name             string
		trn              string
		expectAccessRule bool
		expectErrorCode  errors.CodeType
	}

	testCases := []testCase{
		{
			name:             "get resource by TRN",
			trn:              accessRule.Metadata.TRN,
			expectAccessRule: true,
		},
		{
			name: "resource with TRN not found",
			trn:  trn.TypeManagedIdentityAccessRule.Build(group.FullPath, managedIdentity.Name, nonExistentGlobalID),
		},
		{
			name:            "managed identity rule trn cannot have less than three parts",
			trn:             trn.TypeManagedIdentityAccessRule.Build(nonExistentGlobalID),
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualAccessRule, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRuleByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			if test.expectAccessRule {
				require.NotNil(t, actualAccessRule)
				assert.Equal(t,
					trn.TypeManagedIdentityAccessRule.Build(group.FullPath, managedIdentity.Name, accessRule.GetGlobalID()),
					actualAccessRule.Metadata.TRN,
				)
			} else {
				assert.Nil(t, actualAccessRule)
			}
		})
	}
}

func TestCreateManagedIdentityAccessRule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
	})
	require.NoError(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:      "service-account-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
	})
	require.NoError(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name: "team-a",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		toCreate        *models.ManagedIdentityAccessRule
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name: "create an access rule",
			toCreate: &models.ManagedIdentityAccessRule{
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        managedIdentity1.Metadata.ID,
				AllowedUserIDs:           []string{user1.Metadata.ID},
				AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
				AllowedTeamIDs:           []string{team1.Metadata.ID},
				VerifyStateLineage:       true,
			},
		},
		{
			name: "non-existent managed identity id",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: nonExistentID,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid managed identity id",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: invalidID,
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			created, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, test.toCreate)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				assert.Nil(t, created)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, created)

			assert.Equal(t, initialResourceVersion, created.Metadata.Version)
			assert.NotEmpty(t, created.Metadata.TRN)
			assert.Equal(t, test.toCreate.RunStage, created.RunStage)
			assert.Equal(t, test.toCreate.ManagedIdentityID, created.ManagedIdentityID)
			assert.Equal(t, test.toCreate.AllowedUserIDs, created.AllowedUserIDs)
			assert.Equal(t, test.toCreate.AllowedServiceAccountIDs, created.AllowedServiceAccountIDs)
			assert.Equal(t, test.toCreate.AllowedTeamIDs, created.AllowedTeamIDs)
			assert.Equal(t, test.toCreate.VerifyStateLineage, created.VerifyStateLineage)
		})
	}
}

func TestUpdateManagedIdentityAccessRule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity0, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group0.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	user0, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
	})
	require.NoError(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-1",
		Email:    "user-1@example.invalid",
	})
	require.NoError(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:          models.JobPlanType,
		Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID: managedIdentity0.Metadata.ID,
		AllowedUserIDs:    []string{user0.Metadata.ID},
	})
	require.NoError(t, err)

	t.Run("update run stage and allowed users", func(t *testing.T) {
		toUpdate := &models.ManagedIdentityAccessRule{
			Metadata: models.ResourceMetadata{
				ID:      createdRule.Metadata.ID,
				Version: createdRule.Metadata.Version,
			},
			RunStage:           models.JobApplyType,
			AllowedUserIDs:     []string{user1.Metadata.ID},
			VerifyStateLineage: true,
		}

		updated, err := testClient.client.ManagedIdentities.UpdateManagedIdentityAccessRule(ctx, toUpdate)
		require.NoError(t, err)
		require.NotNil(t, updated)

		assert.Equal(t, models.JobApplyType, updated.RunStage)
		assert.Equal(t, []string{user1.Metadata.ID}, updated.AllowedUserIDs)
		assert.True(t, updated.VerifyStateLineage)
		assert.Equal(t, createdRule.Metadata.Version+1, updated.Metadata.Version)
	})

	t.Run("non-existent id", func(t *testing.T) {
		_, err := testClient.client.ManagedIdentities.UpdateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
			Metadata: models.ResourceMetadata{
				ID:      nonExistentID,
				Version: createdRule.Metadata.Version,
			},
		})
		assert.Equal(t, errors.EOptimisticLock, errors.ErrorCode(err))
	})

	t.Run("invalid id", func(t *testing.T) {
		_, err := testClient.client.ManagedIdentities.UpdateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
			Metadata: models.ResourceMetadata{
				ID:      invalidID,
				Version: createdRule.Metadata.Version,
			},
		})
		assert.Equal(t, errors.EInternal, errors.ErrorCode(err))
	})
}

func TestDeleteManagedIdentityAccessRule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		FullPath:  "top-level-group-0-for-managed-identities",
		CreatedBy: "someone-g0",
	})
	require.NoError(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:      "managed-identity-0",
		GroupID:   group1.Metadata.ID,
		CreatedBy: "someone-sa0",
		Type:      models.ManagedIdentityAWSFederated,
		Data:      []byte("managed-identity-0-data"),
	})
	require.NoError(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:          models.JobPlanType,
		Type:              models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID: managedIdentity1.Metadata.ID,
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		id              string
		version         int
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "delete access rule",
			id:      createdRule.Metadata.ID,
			version: createdRule.Metadata.Version,
		},
		{
			name:            "delete will fail because resource version doesn't match",
			id:              createdRule.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:            "non-existent id",
			id:              nonExistentID,
			expectErrorCode: errors.EOptimisticLock,
		},
		{
			name:            "invalid id",
			id:              invalidID,
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ManagedIdentities.DeleteManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
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

			rule, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRuleByID(ctx, test.id)
			require.NoError(t, err)
			assert.Nil(t, rule)
		})
	}
}
