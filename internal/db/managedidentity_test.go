//go:build integration

package db

import (
	"context"
	"sort"
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

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// managedIdentityInfo aids convenience in accessing the information TestGetManagedIdentities about the created resources.
type managedIdentityInfo struct {
	createTime        time.Time
	updateTime        time.Time
	managedIdentityID string
	name              string
}

// managedIdentityInfoIDSlice makes a slice of managedIdentityInfo sortable by ID string
type managedIdentityInfoIDSlice []managedIdentityInfo

// managedIdentityInfoCreateSlice makes a slice of managedIdentityInfo sortable by creation time
type managedIdentityInfoCreateSlice []managedIdentityInfo

// managedIdentityInfoUpdateSlice makes a slice of managedIdentityInfo sortable by last updated time
type managedIdentityInfoUpdateSlice []managedIdentityInfo

// managedIdentityInfoNameSlice makes a slice of managedIdentityInfo sortable by name
type managedIdentityInfoNameSlice []managedIdentityInfo

func TestGetManagedIdentityByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectManagedIdentity *models.ManagedIdentity
		expectMsg             *string
		name                  string
		searchID              string
	}

	// Do only one positive test case, because the logic is theoretically the same for all managed identities.
	testCases := []testCase{
		{
			name:                  "positive",
			searchID:              managedIdentity1.Metadata.ID,
			expectManagedIdentity: managedIdentity1,
		},
		{
			name:                  "positive: successfully retrieve a managed identity alias",
			searchID:              createdAlias.Metadata.ID,
			expectManagedIdentity: createdAlias,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect managed identity and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualManagedIdentity, err := testClient.client.ManagedIdentities.GetManagedIdentityByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectManagedIdentity != nil {
				require.NotNil(t, actualManagedIdentity)
				compareManagedIdentities(t, test.expectManagedIdentity, actualManagedIdentity, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualManagedIdentity)
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
			trn:  types.ManagedIdentityModelType.BuildTRN(group.FullPath, "unknown"),
		},
		{
			name:            "managed identity trn must have two parts",
			trn:             types.ManagedIdentityModelType.BuildTRN("unknown"),
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
				assert.Equal(t, types.ManagedIdentityModelType.BuildTRN(group.FullPath, managedIdentity.Name), actualManagedIdentity.Metadata.TRN)
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

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	maxJobDuration := int32((time.Hour * 12).Minutes())
	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Description:    "workspace 0 for testing managed identity functions",
		FullPath:       "top-level-group-0-for-managed-identities/workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: &maxJobDuration,
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectMsg               *string
		name                    string
		workspaceID             string
		expectManagedIdentities []models.ManagedIdentity
		addToWorkspace          bool
	}

	// Do the not-added-to-workspace test case first.
	testCases := []testCase{
		{
			name:                    "not added to workspace",
			workspaceID:             workspace1.Metadata.ID,
			expectManagedIdentities: []models.ManagedIdentity{},
		},
		{
			name:                    "positive",
			workspaceID:             workspace1.Metadata.ID,
			addToWorkspace:          true,
			expectManagedIdentities: []models.ManagedIdentity{*managedIdentity1, *createdAlias},
		},
		{
			name:                    "negative, non-existent ID",
			workspaceID:             nonExistentID,
			expectManagedIdentities: []models.ManagedIdentity{},
			// expect error to be nil
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// If specified, add the managed identities to the workspace.
			if test.addToWorkspace {
				for _, identity := range test.expectManagedIdentities {
					err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
						identity.Metadata.ID, workspace1.Metadata.ID)
					require.Nil(t, err)
				}
			}

			actualManagedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, test.workspaceID)

			checkError(t, test.expectMsg, err)

			if test.expectManagedIdentities != nil {
				require.NotNil(t, actualManagedIdentities)
				require.Equal(t, len(test.expectManagedIdentities), len(actualManagedIdentities))
				for ix := range test.expectManagedIdentities {
					compareManagedIdentities(t, &test.expectManagedIdentities[ix], &actualManagedIdentities[ix],
						false, &timeBounds{
							createLow:  &createdLow,
							createHigh: &createdHigh,
							updateLow:  &createdLow,
							updateHigh: &createdHigh,
						})
				}
			} else {
				assert.Nil(t, actualManagedIdentities)
			}
		})
	}
}

func TestAddManagedIdentityToWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	maxJobDuration := int32((time.Hour * 12).Minutes())
	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Description:    "workspace 0 for testing managed identity functions",
		FullPath:       "top-level-group-0-for-managed-identities/workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: &maxJobDuration,
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		overrideManagedIdentityID *string
		expectAddFail             *string
		expectVerifyFail          *string
		name                      string
		workspaceID               string
		expectManagedIdentities   []models.ManagedIdentity
		addToWorkspace            bool
	}

	// Do the not-added-to-workspace test case first.
	testCases := []testCase{
		{
			name:                    "not added to workspace",
			workspaceID:             workspace1.Metadata.ID,
			expectManagedIdentities: []models.ManagedIdentity{},
		},
		{
			name:                    "positive",
			workspaceID:             workspace1.Metadata.ID,
			addToWorkspace:          true,
			expectManagedIdentities: []models.ManagedIdentity{*managedIdentity1},
		},
		{
			name:           "already-added",
			workspaceID:    workspace1.Metadata.ID,
			addToWorkspace: true,
			expectAddFail:  ptr.String("managed identity already assigned to workspace"),
		},
		{
			name:           "non-existent workspace ID",
			workspaceID:    nonExistentID,
			addToWorkspace: true,
			expectAddFail:  ptr.String("ERROR: insert or update on table \"workspace_managed_identity_relation\" violates foreign key constraint \"fk_workspace_id\" (SQLSTATE 23503)"),
		},
		{
			name:           "invalid workspace ID",
			workspaceID:    invalidID,
			addToWorkspace: true,
			expectAddFail:  invalidUUIDMsg,
		},
		{
			name:                      "non-existent managed identity ID",
			addToWorkspace:            true,
			overrideManagedIdentityID: ptr.String(nonExistentID),
			expectAddFail:             ptr.String("ERROR: invalid input syntax for type uuid: \"\" (SQLSTATE 22P02)"),
			// This particular error message seems odd to be happening here, but that's what it does.
		},
		{
			name:                      "invalid managed identity ID",
			addToWorkspace:            true,
			overrideManagedIdentityID: ptr.String(invalidID),
			expectAddFail:             invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// If specified, add the managed identity to the workspace.
			if test.addToWorkspace {

				managedIdentityID := managedIdentity1.Metadata.ID
				if test.overrideManagedIdentityID != nil {
					managedIdentityID = *test.overrideManagedIdentityID
				}

				err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
					managedIdentityID, test.workspaceID)
				if test.expectAddFail == nil {
					assert.Nil(t, err)
				} else {
					require.NotNil(t, err)
					assert.Contains(t, err.Error(), *test.expectAddFail)
					// If expected to fail to add, don't bother doing the fetch.
					return
				}
			}

			actualManagedIdentities, err := testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, test.workspaceID)

			checkError(t, test.expectVerifyFail, err)

			// For the positive test cases, verify everything matches.
			if test.expectManagedIdentities != nil {
				require.NotNil(t, actualManagedIdentities)
				require.Equal(t, len(test.expectManagedIdentities), len(actualManagedIdentities))
				for ix := range test.expectManagedIdentities {
					compareManagedIdentities(t, &test.expectManagedIdentities[ix], &actualManagedIdentities[ix],
						false, &timeBounds{
							createLow:  &createdLow,
							createHigh: &createdHigh,
							updateLow:  &createdLow,
							updateHigh: &createdHigh,
						})
				}
			} else {
				assert.Nil(t, actualManagedIdentities)
			}
		})
	}
}

func TestRemoveManagedIdentityFromWorkspace(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	maxJobDuration := int32((time.Hour * 12).Minutes())
	workspace1, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Description:    "workspace 0 for testing managed identity functions",
		FullPath:       "top-level-group-0-for-managed-identities/workspace-0-for-managed-identities",
		GroupID:        group1.Metadata.ID,
		CreatedBy:      "someone-w0",
		MaxJobDuration: &maxJobDuration,
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	type testCase struct {
		overrideManagedIdentityID *string
		expectMsg                 *string
		name                      string
		workspaceID               string
		addToWorkspace            bool
	}

	testCases := []testCase{
		{
			name:           "positive",
			workspaceID:    workspace1.Metadata.ID,
			addToWorkspace: true,
		},
		{
			name:           "not added, so cannot remove, but no error",
			workspaceID:    workspace1.Metadata.ID,
			addToWorkspace: false,
		},
		{
			name:        "non-existent workspace ID, but no error",
			workspaceID: nonExistentID,
		},
		{
			name:        "invalid workspace ID, but no error",
			workspaceID: invalidID,
		},
		{
			name:                      "non-existent managed identity ID, but no error",
			workspaceID:               workspace1.Metadata.ID,
			overrideManagedIdentityID: ptr.String(nonExistentID),
		},
		{
			name:                      "invalid managed identity ID",
			workspaceID:               workspace1.Metadata.ID,
			overrideManagedIdentityID: ptr.String(invalidID),
			expectMsg:                 invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// If specified add the managed identity to the workspace.
			if test.addToWorkspace {
				err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
					managedIdentity1.Metadata.ID, test.workspaceID)
				assert.Nil(t, err)
			}

			// Conditionally override the managed identity ID used for the removal attempt.
			managedIdentityID := managedIdentity1.Metadata.ID
			if test.overrideManagedIdentityID != nil {
				managedIdentityID = *test.overrideManagedIdentityID
			}

			err = testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx,
				managedIdentityID, workspace1.Metadata.ID)

			checkError(t, test.expectMsg, err)
		})
	}
}

func TestCreateManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		Name:        "top-level-group-0-for-managed-identities",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	aliasGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 1 for testing managed identity aliases",
		Name:        "top-level-group-1-for-managed-identity-aliases",
		FullPath:    "top-level-group-1-for-managed-identity-aliases",
		CreatedBy:   "someone-g1",
	})
	require.Nil(t, err)

	// Create a managed identity prior to running tests so an alias can use it.
	aliasSourceIdentity, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Type:        models.ManagedIdentityAWSFederated,
		Name:        "a-managed-identity-for-testing-aliases",
		Description: "A description for this managed identity",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "creator-of-managed-identities",
		Data:        []byte("some-data-for-the-source-managed-identity"),
	})
	require.Nil(t, err)
	assert.NotNil(t, aliasSourceIdentity)

	type testCase struct {
		toCreate      *models.ManagedIdentity
		expectCreated *models.ManagedIdentity
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.ManagedIdentity{
				Name:    "positive-create-managed-identity-nearly-empty",
				GroupID: group1.Metadata.ID,
				Type:    models.ManagedIdentityAWSFederated,
				Data:    []byte("some-data"),
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
					TRN:               types.ManagedIdentityModelType.BuildTRN(group1.FullPath + "/positive-create-managed-identity-nearly-empty"),
				},
				Name:    "positive-create-managed-identity-nearly-empty",
				GroupID: group1.Metadata.ID,
				Type:    models.ManagedIdentityAWSFederated,
				Data:    []byte("some-data"),
			},
		},

		{
			name: "positive full",
			toCreate: &models.ManagedIdentity{
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "positive-create-managed-identity-full",
				Description: "positive create managed identity",
				GroupID:     group1.Metadata.ID,
				Data:        []byte("this is a test of a full managed identity"),
				CreatedBy:   "creator-of-managed-identities",
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
					TRN:               types.ManagedIdentityModelType.BuildTRN(group1.FullPath + "/positive-create-managed-identity-full"),
				},
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "positive-create-managed-identity-full",
				Description: "positive create managed identity",
				GroupID:     group1.Metadata.ID,
				Data:        []byte("this is a test of a full managed identity"),
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
			expectCreated: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
					TRN:               types.ManagedIdentityModelType.BuildTRN(aliasGroup.FullPath + "/positive-create-managed-identity-alias"),
				},
				Type:          aliasSourceIdentity.Type,
				Name:          "positive-create-managed-identity-alias",
				Description:   aliasSourceIdentity.Description,
				GroupID:       aliasGroup.Metadata.ID,
				Data:          aliasSourceIdentity.Data,
				CreatedBy:     "creator-of-managed-identities",
				AliasSourceID: &aliasSourceIdentity.Metadata.ID,
			},
		},

		{
			name: "duplicate name in same group",
			toCreate: &models.ManagedIdentity{
				Name:    "positive-create-managed-identity-nearly-empty",
				GroupID: group1.Metadata.ID,
				Type:    models.ManagedIdentityAWSFederated,
				Data:    []byte("some-data"),
				// Resource path is not used when creating the object, but it is returned.
			},
			expectMsg: ptr.String("managed identity already exists in the specified group"),
		},

		{
			name: "non-existent group ID",
			toCreate: &models.ManagedIdentity{
				Name:    "non-existent-group-id",
				GroupID: nonExistentID,
				Type:    models.ManagedIdentityAzureFederated,
				Data:    []byte("some-data"),
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"managed_identities\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.ManagedIdentity{
				Name:    "non-existent-group-id",
				GroupID: invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, test.toCreate)
			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareManagedIdentities(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	otherGroup, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 1 for testing managed identity aliases",
		Name:        "top-level-group-1-for-managed-identity-aliases",
		FullPath:    "top-level-group-1-for-managed-identity-aliases",
		CreatedBy:   "someone-g1",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		toUpdate              *models.ManagedIdentity
		expectManagedIdentity *models.ManagedIdentity
		expectErrorCode       errors.CodeType
		name                  string
	}

	// Do only one positive test case, because the logic is theoretically the same for all managed identities.
	now := currentTime()
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      managedIdentity1.Metadata.ID,
					Version: managedIdentity1.Metadata.Version,
				},
				Description: "updated description",
				Type:        managedIdentity1.Type,
				Data:        []byte("updated data"),
				GroupID:     otherGroup.Metadata.ID,
			},
			expectManagedIdentity: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:                   managedIdentity1.Metadata.ID,
					Version:              managedIdentity1.Metadata.Version + 1,
					CreationTimestamp:    managedIdentity1.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
					TRN:                  types.ManagedIdentityModelType.BuildTRN(otherGroup.FullPath + "/" + managedIdentity1.Name),
				},
				Name:        "1-managed-identity-0",
				Description: "updated description",
				Type:        managedIdentity1.Type,
				Data:        []byte("updated data"),
				GroupID:     otherGroup.Metadata.ID, // to move the managed identity to another group
				CreatedBy:   managedIdentity1.CreatedBy,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: managedIdentity1.Metadata.Version,
				},
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "defective-id",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: managedIdentity1.Metadata.Version,
				},
			},
			expectErrorCode: errors.EInternal,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualManagedIdentity, err := testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, test.toUpdate)

			if test.expectErrorCode == "" {
				assert.Nil(t, err)
			} else {
				// Uses require rather than assert to avoid a nil pointer dereference.
				require.NotNil(t, err)
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
			}

			now := currentTime()
			if test.expectManagedIdentity != nil {
				require.NotNil(t, actualManagedIdentity)
				compareManagedIdentities(t, test.expectManagedIdentity, actualManagedIdentity, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualManagedIdentity)
			}
		})
	}
}

func TestGetManagedIdentitiesWithPagination(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		Name:        "top-level-group-0-for-managed-identities",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity0, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-1",
		Description: "managed identity 1 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa1",
		Type:        models.ManagedIdentityAzureFederated,
		Data:        []byte("managed-identity-1-data"),
	})
	require.Nil(t, err)

	managedIdentity2, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "2-managed-identity-2",
		Description: "managed identity 2 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa2",
		Type:        models.ManagedIdentityTharsisFederated,
		Data:        []byte("managed-identity-2-data"),
	})
	require.Nil(t, err)

	managedIdentity3, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "2-managed-identity-3",
		Description: "managed identity 3 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa3",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-3-data"),
	})
	require.Nil(t, err)

	managedIdentity4, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "5-managed-identity-4",
		Description: "managed identity 4 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa4",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-4-data"),
	})
	require.Nil(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group0.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.Nil(t, err)

	allManagedIdentities := []models.ManagedIdentity{
		*managedIdentity0, *managedIdentity1, *managedIdentity2,
		*managedIdentity3, *managedIdentity4, *createdAlias,
	}

	// Query for first page
	middleIndex := len(allManagedIdentities) / 2
	page1, err := testClient.client.ManagedIdentities.GetManagedIdentities(ctx, &GetManagedIdentitiesInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(middleIndex)),
		},
	})
	require.Nil(t, err)

	assert.Equal(t, middleIndex, len(page1.ManagedIdentities))
	assert.True(t, page1.PageInfo.HasNextPage)
	assert.False(t, page1.PageInfo.HasPreviousPage)

	cursor, err := page1.PageInfo.Cursor(&page1.ManagedIdentities[len(page1.ManagedIdentities)-1])
	require.Nil(t, err)

	remaining := len(allManagedIdentities) - middleIndex
	page2, err := testClient.client.ManagedIdentities.GetManagedIdentities(ctx, &GetManagedIdentitiesInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(int32(remaining)),
			After: cursor,
		},
	})
	require.Nil(t, err)

	assert.Equal(t, remaining, len(page2.ManagedIdentities))
	assert.True(t, page2.PageInfo.HasPreviousPage)
	assert.False(t, page2.PageInfo.HasNextPage)
}

func TestGetManagedIdentities(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		Name:        "top-level-group-0-for-managed-identities",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity0, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-1",
		Description: "managed identity 1 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa1",
		Type:        models.ManagedIdentityAzureFederated,
		Data:        []byte("managed-identity-1-data"),
	})
	require.Nil(t, err)

	managedIdentity2, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "2-managed-identity-2",
		Description: "managed identity 2 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa2",
		Type:        models.ManagedIdentityTharsisFederated,
		Data:        []byte("managed-identity-2-data"),
	})
	require.Nil(t, err)

	managedIdentity3, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "2-managed-identity-3",
		Description: "managed identity 3 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa3",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-3-data"),
	})
	require.Nil(t, err)

	managedIdentity4, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "5-managed-identity-4",
		Description: "managed identity 4 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa4",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-4-data"),
	})
	require.Nil(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group0.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.Nil(t, err)

	allManagedIdentities := []models.ManagedIdentity{
		*managedIdentity0, *managedIdentity1, *managedIdentity2,
		*managedIdentity3, *managedIdentity4, *createdAlias,
	}

	allManagedIdentityInfos := managedIdentityInfoFromManagedIdentities(allManagedIdentities)

	// Sort by ID string for those cases where explicit sorting is not specified.
	sort.Sort(managedIdentityInfoIDSlice(allManagedIdentityInfos))
	allManagedIdentityIDs := managedIdentityIDsFromManagedIdentityInfos(allManagedIdentityInfos)

	// Sort by creation times.
	sort.Sort(managedIdentityInfoCreateSlice(allManagedIdentityInfos))
	allManagedIdentityIDsByCreateTime := managedIdentityIDsFromManagedIdentityInfos(allManagedIdentityInfos)
	reverseManagedIdentityIDsByCreateTime := reverseStringSlice(allManagedIdentityIDsByCreateTime)

	// Sort by last update times.
	sort.Sort(managedIdentityInfoUpdateSlice(allManagedIdentityInfos))
	allManagedIdentityIDsByUpdateTime := managedIdentityIDsFromManagedIdentityInfos(allManagedIdentityInfos)
	reverseManagedIdentityIDsByUpdateTime := reverseStringSlice(allManagedIdentityIDsByUpdateTime)

	// Sort by names.
	sort.Sort(managedIdentityInfoNameSlice(allManagedIdentityInfos))
	allManagedIdentityIDsByName := managedIdentityIDsFromManagedIdentityInfos(allManagedIdentityInfos)

	dummyCursorFunc := func(cp pagination.CursorPaginatable) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetManagedIdentitiesInput
		name                        string
		expectPageInfo              pagination.PageInfo
		expectManagedIdentityIDs    []string
		getBeforeCursorFromPrevious bool
		getAfterCursorFromPrevious  bool
		expectHasStartCursor        bool
		expectHasEndCursor          bool
	}

	/*
		template test case:

		{
		name                        string
		input                       *GetManagedIdentitiesInput
		getAfterCursorFromPrevious  bool
		getBeforeCursorFromPrevious bool
		expectMsg                   *string
		expectManagedIdentityIDs    []string
		expectPageInfo              pagination.PageInfo
		expectStartCursorError      error
		expectEndCursorError        error
		expectHasStartCursor        bool
		expectHasEndCursor          bool
		}
	*/

	testCases := []testCase{
		// nil input likely causes a nil pointer dereference in GetManagedIdentities, so don't try it.

		{
			name: "non-nil but mostly empty input",
			input: &GetManagedIdentitiesInput{
				Sort:              nil,
				PaginationOptions: nil,
				Filter:            nil,
			},
			expectManagedIdentityIDs: allManagedIdentityIDs,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByCreateTime,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtDesc),
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByCreateTime,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtDesc),
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByUpdateTime,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "pagination: everything at once",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "pagination: first two",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime[:2],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: middle two",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectManagedIdentityIDs:   allManagedIdentityIDsByUpdateTime[2:4],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "pagination: final one",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectManagedIdentityIDs:   allManagedIdentityIDsByUpdateTime[4:],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		// When Last is supplied, the sort order is intended to be reversed.
		{
			name: "pagination: last three",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					Last: ptr.Int32(3),
				},
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByUpdateTime[:3],
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: true,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		/*

			The input.PaginationOptions.After field is tested earlier via getAfterCursorFromPrevious.

			The input.PaginationOptions.Before field is not really supported and does not work.
			If it did work, it could be tested by adapting the test cases corresponding to the
			next few cases after a similar block of text from group_test.go

		*/

		{
			name: "pagination, before and after, expect error",
			input: &GetManagedIdentitiesInput{
				Sort:              ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectManagedIdentityIDs:    []string{},
			expectPageInfo:              pagination.PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     true,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			// If there were more filter fields, this would allow nothing through the filters.
			name: "fully-populated types, everything allowed through filters",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				PaginationOptions: &pagination.Options{
					First: ptr.Int32(100),
				},
				Filter: &ManagedIdentityFilter{
					Search: ptr.String(""),
					// Passing an empty slice to NamespacePaths likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// NamespacePaths: []string{},
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByCreateTime,
			expectPageInfo: pagination.PageInfo{
				TotalCount:      int32(len(allManagedIdentityIDs)),
				Cursor:          dummyCursorFunc,
				HasNextPage:     false,
				HasPreviousPage: false,
			},
			expectHasStartCursor: true,
			expectHasEndCursor:   true,
		},

		{
			name: "filter, search field, empty string",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String(""),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByName,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, search field, 1",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String("1"),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByName[0:2],
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, search field, 2",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String("2"),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByName[2:4],
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, search field, 5",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String("5"),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByName[4:5],
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, search field, bogus",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String("bogus"),
				},
			},
			expectManagedIdentityIDs: []string{},
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, namespace paths, positive",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					NamespacePaths: []string{"top-level-group-0-for-managed-identities"},
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByName,
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "filter, namespace paths, negative",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					NamespacePaths: []string{"top-level-group-9-for-managed-identities"},
				},
			},
			expectManagedIdentityIDs: []string{},
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},
		{
			name: "filter, search for a managed identity alias, positive",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				Filter: &ManagedIdentityFilter{
					Search: ptr.String(createdAlias.GetResourcePath()),
				},
			},
			expectManagedIdentityIDs: []string{createdAlias.Metadata.ID},
			expectPageInfo:           pagination.PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},
	}

	// Combinations of filter conditions are not (yet) tested.

	var (
		previousEndCursorValue   *string
		previousStartCursorValue *string
	)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// For some pagination tests, a previous case's cursor value gets piped into the next case.
			if test.getAfterCursorFromPrevious || test.getBeforeCursorFromPrevious {

				// Make sure there's a place to put it.
				require.NotNil(t, test.input.PaginationOptions)

				if test.getAfterCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousEndCursorValue)
					test.input.PaginationOptions.After = previousEndCursorValue
				}

				if test.getBeforeCursorFromPrevious {
					// Make sure there's a previous value to use.
					require.NotNil(t, previousStartCursorValue)
					test.input.PaginationOptions.Before = previousStartCursorValue
				}

				// Clear the values so they won't be used twice.
				previousEndCursorValue = nil
				previousStartCursorValue = nil
			}

			managedIdentitiesActual, err := testClient.client.ManagedIdentities.GetManagedIdentities(ctx, test.input)

			checkError(t, test.expectMsg, err)

			// If there was no error, check the results.
			if err == nil {

				// Never returns nil if error is nil.
				require.NotNil(t, managedIdentitiesActual.PageInfo)
				assert.NotNil(t, managedIdentitiesActual.ManagedIdentities)
				pageInfo := managedIdentitiesActual.PageInfo
				managedIdentities := managedIdentitiesActual.ManagedIdentities

				// Check the managed identities result by comparing a list of the managed identity IDs.
				actualManagedIdentityIDs := []string{}
				for _, managedIdentity := range managedIdentities {
					actualManagedIdentityIDs = append(actualManagedIdentityIDs, managedIdentity.Metadata.ID)
				}

				// If no sort direction was specified, sort the results here for repeatability.
				if test.input.Sort == nil {
					sort.Strings(actualManagedIdentityIDs)
				}

				assert.Equal(t, len(test.expectManagedIdentityIDs), len(actualManagedIdentityIDs))
				assert.Equal(t, test.expectManagedIdentityIDs, actualManagedIdentityIDs)

				assert.Equal(t, test.expectPageInfo.HasNextPage, pageInfo.HasNextPage)
				assert.Equal(t, test.expectPageInfo.HasPreviousPage, pageInfo.HasPreviousPage)
				assert.Equal(t, test.expectPageInfo.TotalCount, pageInfo.TotalCount)
				assert.Equal(t, test.expectPageInfo.Cursor != nil, pageInfo.Cursor != nil)

				// Compare the cursor function results only if there is at least one managed identity returned.
				// If there are no managed identities returned, there is no argument to pass to the cursor function.
				// Also, don't try to reverse engineer to compare the cursor string values.
				if len(managedIdentities) > 0 {
					resultStartCursor, resultStartCursorError := pageInfo.Cursor(&managedIdentities[0])
					resultEndCursor, resultEndCursorError := pageInfo.Cursor(&managedIdentities[len(managedIdentities)-1])
					assert.Equal(t, test.expectStartCursorError, resultStartCursorError)
					assert.Equal(t, test.expectHasStartCursor, resultStartCursor != nil)
					assert.Equal(t, test.expectEndCursorError, resultEndCursorError)
					assert.Equal(t, test.expectHasEndCursor, resultEndCursor != nil)

					// Capture the ending cursor values for the next case.
					previousEndCursorValue = resultEndCursor
					previousStartCursorValue = resultStartCursor
				}
			}
		})
	}
}

func TestDeleteManagedIdentity(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	type testCase struct {
		toDelete  *models.ManagedIdentity
		expectMsg *string
		name      string
	}

	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      managedIdentity1.Metadata.ID,
					Version: managedIdentity1.Metadata.Version,
				},
			},
		},

		{
			name: "negative, non-existent ID",
			toDelete: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
				Description: "looking for a non-existent ID",
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toDelete: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
				Description: "looking for a defective ID",
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ManagedIdentities.DeleteManagedIdentity(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

func TestGetManagedIdentityAccessRules(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	createdAlias, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:          "an-alias-created-for-testing",
		GroupID:       group1.Metadata.ID,
		CreatedBy:     "someone-ma1",
		AliasSourceID: &managedIdentity1.Metadata.ID,
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           group1.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	})
	require.Nil(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:                 models.JobPlanType,
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID:        managedIdentity1.Metadata.ID,
		AllowedUserIDs:           []string{user1.Metadata.ID},
		AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
		AllowedTeamIDs:           []string{team1.Metadata.ID},
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectMsg                        *string
		name                             string
		searchID                         string
		expectManagedIdentityAccessRules []models.ManagedIdentityAccessRule
	}

	// TODO: Add test cases to cover the expanded functionality of the more general GetManagedIdentityAccessRules function.

	// Do only one positive test case,
	// because the logic is theoretically the same for all managed identity access rules.
	testCases := []testCase{
		{
			name:                             "positive",
			searchID:                         managedIdentity1.Metadata.ID,
			expectManagedIdentityAccessRules: []models.ManagedIdentityAccessRule{*createdRule},
		},
		{
			name:                             "positive: successfully retrieve access rules for a managed identity alias",
			searchID:                         createdAlias.Metadata.ID,
			expectManagedIdentityAccessRules: []models.ManagedIdentityAccessRule{*createdRule},
		},
		{
			name:                             "negative, non-existent ID",
			searchID:                         nonExistentID,
			expectManagedIdentityAccessRules: []models.ManagedIdentityAccessRule{},
			// expect error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualResult, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &GetManagedIdentityAccessRulesInput{
				Filter: &ManagedIdentityAccessRuleFilter{
					ManagedIdentityID: &test.searchID,
				},
			})

			checkError(t, test.expectMsg, err)

			if test.expectManagedIdentityAccessRules != nil {
				actualManagedIdentityAccessRules := actualResult.ManagedIdentityAccessRules
				require.NotNil(t, actualManagedIdentityAccessRules)
				require.Equal(t, len(test.expectManagedIdentityAccessRules), len(actualManagedIdentityAccessRules))
				for ix := range test.expectManagedIdentityAccessRules {
					expectedRule := &test.expectManagedIdentityAccessRules[ix]
					actualRule := &actualManagedIdentityAccessRules[ix]
					compareManagedIdentityAccessRules(t, expectedRule, actualRule, false, &timeBounds{
						createLow:  &createdLow,
						createHigh: &createdHigh,
						updateLow:  &createdLow,
						updateHigh: &createdHigh,
					})
				}
			} else {
				assert.Nil(t, actualResult)
			}
		})
	}
}

func TestGetManagedIdentityAccessRuleByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           group1.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	})
	require.Nil(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:                 models.JobPlanType,
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID:        managedIdentity1.Metadata.ID,
		AllowedUserIDs:           []string{user1.Metadata.ID},
		AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
		AllowedTeamIDs:           []string{team1.Metadata.ID},
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		expectManagedIdentityAccessRule *models.ManagedIdentityAccessRule
		expectMsg                       *string
		name                            string
		searchID                        string
	}

	// Do only one positive test case,
	// because the logic is theoretically the same for all managed identity access rules.
	testCases := []testCase{
		{
			name:                            "positive",
			searchID:                        createdRule.Metadata.ID,
			expectManagedIdentityAccessRule: createdRule,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect rule and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: ptr.String(ErrInvalidID.Error()),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualManagedIdentityAccessRule, err := testClient.client.ManagedIdentities.GetManagedIdentityAccessRuleByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectManagedIdentityAccessRule != nil {
				require.NotNil(t, actualManagedIdentityAccessRule)

				compareManagedIdentityAccessRules(t,
					test.expectManagedIdentityAccessRule, actualManagedIdentityAccessRule, false, &timeBounds{
						createLow:  &createdLow,
						createHigh: &createdHigh,
						updateLow:  &createdLow,
						updateHigh: &createdHigh,
					})
			} else {
				assert.Nil(t, actualManagedIdentityAccessRule)
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
			trn:  types.ManagedIdentityAccessRuleModelType.BuildTRN(group.FullPath, managedIdentity.Name, nonExistentGlobalID),
		},
		{
			name:            "managed identity rule TRN cannot have less than three parts",
			trn:             types.ManagedIdentityAccessRuleModelType.BuildTRN(nonExistentGlobalID),
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
					types.ManagedIdentityAccessRuleModelType.BuildTRN(group.FullPath, managedIdentity.Name, accessRule.GetGlobalID()),
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
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           group1.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	})
	require.Nil(t, err)

	type testCase struct {
		toCreate      *models.ManagedIdentityAccessRule
		expectCreated *models.ManagedIdentityAccessRule
		expectMsg     *string
		name          string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive, nearly empty",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: managedIdentity1.Metadata.ID,
			},
			expectCreated: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ManagedIdentityID: managedIdentity1.Metadata.ID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.ManagedIdentityAccessRule{
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        managedIdentity1.Metadata.ID,
				AllowedUserIDs:           []string{user1.Metadata.ID},
				AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
				AllowedTeamIDs:           []string{team1.Metadata.ID},
				VerifyStateLineage:       true,
			},
			expectCreated: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        managedIdentity1.Metadata.ID,
				AllowedUserIDs:           []string{user1.Metadata.ID},
				AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
				AllowedTeamIDs:           []string{team1.Metadata.ID},
				VerifyStateLineage:       true,
			},
		},

		{
			name: "non-existent managed identity ID",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"managed_identity_rules\" violates foreign key constraint \"fk_managed_identity_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: invalidID,
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualCreated, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, test.toCreate)

			checkError(t, test.expectMsg, err)

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareManagedIdentityAccessRules(t, test.expectCreated, actualCreated, false, &timeBounds{
					createLow:  whenCreated,
					createHigh: &now,
					updateLow:  whenCreated,
					updateHigh: &now,
				})
			} else {
				// the negative and defective cases
				assert.Nil(t, actualCreated)
			}
		})
	}
}

func TestUpdateManagedIdentityAccessRule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	createdLow := currentTime()

	group0, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity0, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group0.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	user0, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-1",
		Email:    "user-1@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	serviceAccount0, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           group0.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-1",
		Description:       "service account 1 for testing managed identities",
		GroupID:           group0.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	team0, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	})
	require.Nil(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-b",
		Description: "team b for managed identity tests",
	})
	require.Nil(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:                 models.JobPlanType,
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID:        managedIdentity0.Metadata.ID,
		AllowedUserIDs:           []string{user0.Metadata.ID},
		AllowedServiceAccountIDs: []string{serviceAccount0.Metadata.ID},
		AllowedTeamIDs:           []string{team0.Metadata.ID},
	})
	require.Nil(t, err)

	createdHigh := currentTime()

	type testCase struct {
		toUpdate              *models.ManagedIdentityAccessRule
		expectManagedIdentity *models.ManagedIdentityAccessRule
		expectMsg             *string
		name                  string
	}

	// Do only one positive test case,
	// because the logic is theoretically the same for all managed identity access rules.
	now := currentTime()
	positiveRule := createdRule
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      positiveRule.Metadata.ID,
					Version: positiveRule.Metadata.Version,
				},
				RunStage:                 models.JobApplyType,
				AllowedUserIDs:           []string{user1.Metadata.ID},
				AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
				AllowedTeamIDs:           []string{team1.Metadata.ID},
				VerifyStateLineage:       true,
			},
			expectManagedIdentity: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRule.Metadata.ID,
					Version:              positiveRule.Metadata.Version + 1,
					CreationTimestamp:    positiveRule.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        managedIdentity0.Metadata.ID,
				AllowedUserIDs:           []string{user1.Metadata.ID},
				AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
				AllowedTeamIDs:           []string{team1.Metadata.ID},
				VerifyStateLineage:       true,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveRule.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveRule.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualManagedIdentity, err := testClient.client.ManagedIdentities.UpdateManagedIdentityAccessRule(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := currentTime()
			if test.expectManagedIdentity != nil {
				require.NotNil(t, actualManagedIdentity)
				compareManagedIdentityAccessRules(t,
					test.expectManagedIdentity, actualManagedIdentity, false, &timeBounds{
						createLow:  &createdLow,
						createHigh: &createdHigh,
						updateLow:  &createdLow,
						updateHigh: &now,
					})
			} else {
				assert.Nil(t, actualManagedIdentity)
			}
		})
	}
}

func TestDeleteManagedIdentityAccessRule(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group1, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	})
	require.Nil(t, err)

	managedIdentity1, err := testClient.client.ManagedIdentities.CreateManagedIdentity(ctx, &models.ManagedIdentity{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     group1.Metadata.ID,
		CreatedBy:   "someone-sa0",
		Type:        models.ManagedIdentityAWSFederated,
		Data:        []byte("managed-identity-0-data"),
	})
	require.Nil(t, err)

	user1, err := testClient.client.Users.CreateUser(ctx, &models.User{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	})
	require.Nil(t, err)

	serviceAccount1, err := testClient.client.ServiceAccounts.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           group1.Metadata.ID,
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	})
	require.Nil(t, err)

	team1, err := testClient.client.Teams.CreateTeam(ctx, &models.Team{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	})
	require.Nil(t, err)

	createdRule, err := testClient.client.ManagedIdentities.CreateManagedIdentityAccessRule(ctx, &models.ManagedIdentityAccessRule{
		RunStage:                 models.JobPlanType,
		Type:                     models.ManagedIdentityAccessRuleEligiblePrincipals,
		ManagedIdentityID:        managedIdentity1.Metadata.ID,
		AllowedUserIDs:           []string{user1.Metadata.ID},
		AllowedServiceAccountIDs: []string{serviceAccount1.Metadata.ID},
		AllowedTeamIDs:           []string{team1.Metadata.ID},
	})
	require.Nil(t, err)

	type testCase struct {
		toDelete  *models.ManagedIdentityAccessRule
		expectMsg *string
		name      string
	}

	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      createdRule.Metadata.ID,
					Version: createdRule.Metadata.Version,
				},
			},
		},
		{
			name: "negative, non-existent ID",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: createdRule.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: createdRule.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.ManagedIdentities.DeleteManagedIdentityAccessRule(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)
		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

func ptrManagedIdentitySortableField(arg ManagedIdentitySortableField) *ManagedIdentitySortableField {
	return &arg
}

func (miis managedIdentityInfoIDSlice) Len() int {
	return len(miis)
}

func (miis managedIdentityInfoIDSlice) Swap(i, j int) {
	miis[i], miis[j] = miis[j], miis[i]
}

func (miis managedIdentityInfoIDSlice) Less(i, j int) bool {
	return miis[i].managedIdentityID < miis[j].managedIdentityID
}

func (miis managedIdentityInfoCreateSlice) Len() int {
	return len(miis)
}

func (miis managedIdentityInfoCreateSlice) Swap(i, j int) {
	miis[i], miis[j] = miis[j], miis[i]
}

func (miis managedIdentityInfoCreateSlice) Less(i, j int) bool {
	return miis[i].createTime.Before(miis[j].createTime)
}

func (miis managedIdentityInfoUpdateSlice) Len() int {
	return len(miis)
}

func (miis managedIdentityInfoUpdateSlice) Swap(i, j int) {
	miis[i], miis[j] = miis[j], miis[i]
}

func (miis managedIdentityInfoUpdateSlice) Less(i, j int) bool {
	return miis[i].updateTime.Before(miis[j].updateTime)
}

func (miis managedIdentityInfoNameSlice) Len() int {
	return len(miis)
}

func (miis managedIdentityInfoNameSlice) Swap(i, j int) {
	miis[i], miis[j] = miis[j], miis[i]
}

func (miis managedIdentityInfoNameSlice) Less(i, j int) bool {
	return miis[i].name < miis[j].name
}

// managedIdentityInfoFromManagedIdentities returns a slice of managedIdentityInfo, not necessarily sorted in any order.
func managedIdentityInfoFromManagedIdentities(managedIdentities []models.ManagedIdentity) []managedIdentityInfo {
	result := []managedIdentityInfo{}

	for _, managedIdentity := range managedIdentities {
		result = append(result, managedIdentityInfo{
			createTime:        *managedIdentity.Metadata.CreationTimestamp,
			updateTime:        *managedIdentity.Metadata.LastUpdatedTimestamp,
			managedIdentityID: managedIdentity.Metadata.ID,
			name:              managedIdentity.Name,
		})
	}

	return result
}

// managedIdentityIDsFromManagedIdentityInfos preserves order
func managedIdentityIDsFromManagedIdentityInfos(managedIdentityInfos []managedIdentityInfo) []string {
	result := []string{}
	for _, managedIdentityInfo := range managedIdentityInfos {
		result = append(result, managedIdentityInfo.managedIdentityID)
	}
	return result
}

// compareManagedIdentities compares two managed identity objects, including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareManagedIdentities(t *testing.T, expected, actual *models.ManagedIdentity,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Data, actual.Data)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}

// compareManagedIdentityAccessRules compares two managed identity access rule objects,
// including bounds for creation and updated times.
// If times is nil, it compares the exact metadata timestamps.
func compareManagedIdentityAccessRules(t *testing.T, expected, actual *models.ManagedIdentityAccessRule,
	checkID bool, times *timeBounds,
) {
	assert.Equal(t, expected.RunStage, actual.RunStage)
	assert.Equal(t, expected.ManagedIdentityID, actual.ManagedIdentityID)
	assert.Equal(t, expected.AllowedUserIDs, actual.AllowedUserIDs)
	assert.Equal(t, expected.AllowedServiceAccountIDs, actual.AllowedServiceAccountIDs)
	assert.Equal(t, expected.AllowedTeamIDs, actual.AllowedTeamIDs)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)
	assert.NotEmpty(t, actual.Metadata.TRN)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}
