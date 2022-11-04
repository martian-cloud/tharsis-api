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
)

// Some constants and pseudo-constants are declared/defined in dbclient_test.go.

// managedIdentityInfo aids convenience in accessing the information TestGetManagedIdentities
// needs about the warmup managed identities.
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

// warmupManagedIdentities holds the inputs to and outputs from createWarmupManagedIdentities.
type warmupManagedIdentities struct {
	groups            []models.Group
	workspaces        []models.Workspace
	teams             []models.Team
	users             []models.User
	serviceAccounts   []models.ServiceAccount
	managedIdentities []models.ManagedIdentity
	rules             []models.ManagedIdentityAccessRule
}

func TestGetManagedIdentityByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()

	type testCase struct {
		expectManagedIdentity *models.ManagedIdentity
		expectMsg             *string
		name                  string
		searchID              string
	}

	// Do only one positive test case, because the logic is theoretically the same for all managed identities.
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{
		{
			name:                  "positive",
			searchID:              positiveManagedIdentity.Metadata.ID,
			expectManagedIdentity: &positiveManagedIdentity,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect managed identity and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualManagedIdentity, err :=
				testClient.client.ManagedIdentities.GetManagedIdentityByID(ctx, test.searchID)

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

func TestGetManagedIdentityByPath(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()

	type testCase struct {
		expectManagedIdentity *models.ManagedIdentity
		expectMsg             *string
		name                  string
		searchID              string
	}

	// Do only one positive test case, because the logic is theoretically the same for all managed identities.
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{
		{
			name:                  "positive",
			searchID:              positiveManagedIdentity.ResourcePath,
			expectManagedIdentity: &positiveManagedIdentity,
		},
		{
			name:     "negative, non-existent ID",
			searchID: "even-non-existent-search-paths/must-contain-a-slash",
			// expect managed identity and error to be nil
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualManagedIdentity, err :=
				testClient.client.ManagedIdentities.GetManagedIdentityByPath(ctx, test.searchID)

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

func TestGetManagedIdentitiesForWorkspace(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()

	type testCase struct {
		expectMsg               *string
		name                    string
		workspaceID             string
		expectManagedIdentities []models.ManagedIdentity
		addToWorkspace          bool
	}

	// Do the not-added-to-workspace test case first.
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{
		{
			name:                    "not added to workspace",
			workspaceID:             warmupItems.workspaces[0].Metadata.ID,
			expectManagedIdentities: []models.ManagedIdentity{},
		},
		{
			name:                    "positive",
			workspaceID:             warmupItems.workspaces[0].Metadata.ID,
			addToWorkspace:          true,
			expectManagedIdentities: []models.ManagedIdentity{positiveManagedIdentity},
		},
		{
			name:                    "negative, non-existent ID",
			workspaceID:             nonExistentID,
			expectManagedIdentities: []models.ManagedIdentity{},
			// expect error to be nil
		},
		{
			name:        "negative, invalid ID",
			workspaceID: invalidID,
			expectMsg:   invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// If specified, add the managed identity to the workspace.
			if test.addToWorkspace {
				err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
					warmupItems.managedIdentities[0].Metadata.ID, warmupItems.workspaces[0].Metadata.ID)
				require.Nil(t, err)
			}

			actualManagedIdentities, err :=
				testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, test.workspaceID)

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

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
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
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{
		{
			name:                    "not added to workspace",
			workspaceID:             warmupItems.workspaces[0].Metadata.ID,
			expectManagedIdentities: []models.ManagedIdentity{},
		},
		{
			name:                    "positive",
			workspaceID:             warmupItems.workspaces[0].Metadata.ID,
			addToWorkspace:          true,
			expectManagedIdentities: []models.ManagedIdentity{positiveManagedIdentity},
		},
		{
			name:           "already-added",
			workspaceID:    warmupItems.workspaces[0].Metadata.ID,
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
			expectAddFail:  invalidUUIDMsg1,
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
			expectAddFail:             invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// If specified, add the managed identity to the workspace.
			if test.addToWorkspace {

				managedIdentityID := warmupItems.managedIdentities[0].Metadata.ID
				if test.overrideManagedIdentityID != nil {
					managedIdentityID = *test.overrideManagedIdentityID
				}

				err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
					managedIdentityID, test.workspaceID)
				if test.expectAddFail == nil {
					assert.Nil(t, err)
				} else {
					require.NotNil(t, err)
					assert.Equal(t, *test.expectAddFail, err.Error())
					// If expected to fail to add, don't bother doing the fetch.
					return
				}
			}

			actualManagedIdentities, err :=
				testClient.client.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, test.workspaceID)

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

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

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
			workspaceID:    warmupItems.workspaces[0].Metadata.ID,
			addToWorkspace: true,
		},
		{
			name:           "not added, so cannot remove, but no error",
			workspaceID:    warmupItems.workspaces[0].Metadata.ID,
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
			workspaceID:               warmupItems.workspaces[0].Metadata.ID,
			overrideManagedIdentityID: ptr.String(nonExistentID),
		},
		{
			name:                      "invalid managed identity ID",
			workspaceID:               warmupItems.workspaces[0].Metadata.ID,
			overrideManagedIdentityID: ptr.String(invalidID),
			expectMsg:                 invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			// If specified add the managed identity to the workspace.
			if test.addToWorkspace {
				err = testClient.client.ManagedIdentities.AddManagedIdentityToWorkspace(ctx,
					warmupItems.managedIdentities[0].Metadata.ID, test.workspaceID)
				assert.Nil(t, err)
			}

			// Conditionally override the managed identity ID used for the removal attempt.
			managedIdentityID := warmupItems.managedIdentities[0].Metadata.ID
			if test.overrideManagedIdentityID != nil {
				managedIdentityID = *test.overrideManagedIdentityID
			}

			err = testClient.client.ManagedIdentities.RemoveManagedIdentityFromWorkspace(ctx,
				managedIdentityID, warmupItems.workspaces[0].Metadata.ID)

			checkError(t, test.expectMsg, err)
		})
	}
}

func TestCreateManagedIdentity(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			[]models.ManagedIdentity{},
			[]models.ManagedIdentityAccessRule{},
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	warmupGroup := warmupItems.groups[0]
	warmupGroupID := warmupGroup.Metadata.ID

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
				GroupID: warmupGroupID,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Name:         "positive-create-managed-identity-nearly-empty",
				GroupID:      warmupGroupID,
				ResourcePath: warmupGroup.FullPath + "/positive-create-managed-identity-nearly-empty",
				Data:         []byte{},
			},
		},

		{
			name: "positive full",
			toCreate: &models.ManagedIdentity{
				Type:        models.ManagedIdentityAWSFederated,
				Name:        "positive-create-managed-identity-full",
				Description: "positive create managed identity",
				GroupID:     warmupGroupID,
				Data:        []byte("this is a test of a full managed identity"),
				CreatedBy:   "creator-of-managed-identities",
				// Resource path is not used when creating the object, but it is returned.
			},
			expectCreated: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				Type:         models.ManagedIdentityAWSFederated,
				ResourcePath: warmupGroup.FullPath + "/positive-create-managed-identity-full",
				Name:         "positive-create-managed-identity-full",
				Description:  "positive create managed identity",
				GroupID:      warmupGroupID,
				Data:         []byte("this is a test of a full managed identity"),
				CreatedBy:    "creator-of-managed-identities",
			},
		},

		{
			name: "duplicate name in same group",
			toCreate: &models.ManagedIdentity{
				Name:    "positive-create-managed-identity-nearly-empty",
				GroupID: warmupGroupID,
				// Resource path is not used when creating the object, but it is returned.
			},
			expectMsg: ptr.String("managed identity name already exists in the specified group"),
		},

		{
			name: "non-existent group ID",
			toCreate: &models.ManagedIdentity{
				Name:    "non-existent-group-id",
				GroupID: nonExistentID,
			},
			expectMsg: ptr.String("ERROR: insert or update on table \"managed_identities\" violates foreign key constraint \"fk_group_id\" (SQLSTATE 23503)"),
		},

		{
			name: "defective group ID",
			toCreate: &models.ManagedIdentity{
				Name:    "non-existent-group-id",
				GroupID: invalidID,
			},
			expectMsg: invalidUUIDMsg1,
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

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()
	warmupGroup := warmupItems.groups[0]

	type testCase struct {
		toUpdate              *models.ManagedIdentity
		expectManagedIdentity *models.ManagedIdentity
		expectMsg             *string
		name                  string
	}

	// Do only one positive test case, because the logic is theoretically the same for all managed identities.
	now := currentTime()
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      positiveManagedIdentity.Metadata.ID,
					Version: positiveManagedIdentity.Metadata.Version,
				},
				Description: "updated description",
				Data:        []byte("updated data"),
			},
			expectManagedIdentity: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:                   positiveManagedIdentity.Metadata.ID,
					Version:              positiveManagedIdentity.Metadata.Version + 1,
					CreationTimestamp:    positiveManagedIdentity.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				ResourcePath: warmupGroup.FullPath + "/" + positiveManagedIdentity.Name,
				Name:         "1-managed-identity-0",
				Description:  "updated description",
				GroupID:      warmupGroup.Metadata.ID,
				Data:         []byte("updated data"),
				CreatedBy:    positiveManagedIdentity.CreatedBy,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveManagedIdentity.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveManagedIdentity.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualManagedIdentity, err :=
				testClient.client.ManagedIdentities.UpdateManagedIdentity(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

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

func TestGetManagedIdentities(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	allManagedIdentityInfos := managedIdentityInfoFromManagedIdentities(warmupItems.managedIdentities)

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

	dummyCursorFunc := func(item interface{}) (*string, error) { return ptr.String("dummy-cursor-value"), nil }

	type testCase struct {
		expectStartCursorError      error
		expectEndCursorError        error
		expectMsg                   *string
		input                       *GetManagedIdentitiesInput
		name                        string
		expectPageInfo              PageInfo
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
		expectPageInfo              PageInfo
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
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "populated pagination, sort in ascending order of creation time, nil filter",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByCreateTime,
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "sort in descending order of creation time",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldCreatedAtDesc),
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByCreateTime,
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "populated pagination, sort in ascending order of last update time, nil filter",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: nil,
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime,
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "sort in descending order of last update time",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtDesc),
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByUpdateTime,
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "pagination: everything at once",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime,
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
			expectHasStartCursor:     true,
			expectHasEndCursor:       true,
		},

		{
			name: "pagination: first two",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByUpdateTime[:2],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(2),
				},
			},
			getAfterCursorFromPrevious: true,
			expectManagedIdentityIDs:   allManagedIdentityIDsByUpdateTime[2:4],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
			},
			getAfterCursorFromPrevious: true,
			expectManagedIdentityIDs:   allManagedIdentityIDsByUpdateTime[4:],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					Last: ptr.Int32(3),
				},
			},
			expectManagedIdentityIDs: reverseManagedIdentityIDsByUpdateTime[:3],
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{},
			},
			getAfterCursorFromPrevious:  true,
			getBeforeCursorFromPrevious: true,
			expectMsg:                   ptr.String("only before or after can be defined, not both"),
			expectManagedIdentityIDs:    []string{},
			expectPageInfo:              PageInfo{},
		},

		{
			name: "pagination, first one and last two, expect error",
			input: &GetManagedIdentitiesInput{
				Sort: ptrManagedIdentitySortableField(ManagedIdentitySortableFieldUpdatedAtAsc),
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(1),
					Last:  ptr.Int32(2),
				},
			},
			expectMsg: ptr.String("only first or last can be defined, not both"),
			expectPageInfo: PageInfo{
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
				PaginationOptions: &PaginationOptions{
					First: ptr.Int32(100),
				},
				Filter: &ManagedIdentityFilter{
					Search: ptr.String(""),
					// Passing an empty slice to NamespacePaths likely causes an SQL syntax error ("... IN ()"), so don't try it.
					// NamespacePaths: []string{},
				},
			},
			expectManagedIdentityIDs: allManagedIdentityIDsByCreateTime,
			expectPageInfo: PageInfo{
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
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
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
			expectPageInfo:           PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
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
			expectPageInfo:           PageInfo{TotalCount: int32(2), Cursor: dummyCursorFunc},
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
			expectManagedIdentityIDs: allManagedIdentityIDsByName[4:],
			expectPageInfo:           PageInfo{TotalCount: int32(1), Cursor: dummyCursorFunc},
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
			expectPageInfo:           PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
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
			expectPageInfo:           PageInfo{TotalCount: int32(len(allManagedIdentityIDs)), Cursor: dummyCursorFunc},
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
			expectPageInfo:           PageInfo{TotalCount: int32(0), Cursor: dummyCursorFunc},
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

	// Because we cannot create a managed identity with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

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
					ID:      warmupItems.managedIdentities[0].Metadata.ID,
					Version: warmupItems.managedIdentities[0].Metadata.Version,
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
			expectMsg: invalidUUIDMsg1,
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

	// Because we cannot create a managed identity access rule with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()

	type testCase struct {
		expectManagedIdentityAccessRules []models.ManagedIdentityAccessRule
		expectMsg                        *string
		name                             string
		searchID                         string
	}

	// TODO: Add test cases to cover the expanded functionality of the more general GetManagedIdentityAccessRules function.

	// Do only one positive test case,
	// because the logic is theoretically the same for all managed identity access rules.
	testCases := []testCase{
		{
			name:                             "positive",
			searchID:                         warmupItems.managedIdentities[0].Metadata.ID,
			expectManagedIdentityAccessRules: warmupItems.rules[0:1],
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
			expectMsg: invalidUUIDMsg2,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualResult, err :=
				testClient.client.ManagedIdentities.GetManagedIdentityAccessRules(ctx, &GetManagedIdentityAccessRulesInput{
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

func TestGetManagedIdentityAccessRule(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity access rule with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
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
			searchID:                        warmupItems.rules[0].Metadata.ID,
			expectManagedIdentityAccessRule: &warmupItems.rules[0],
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect rule and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualManagedIdentityAccessRule, err :=
				testClient.client.ManagedIdentities.GetManagedIdentityAccessRule(ctx, test.searchID)

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

func TestCreateManagedIdentityAccessRule(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a managed identity access rule with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toCreate      *models.ManagedIdentityAccessRule
		expectCreated *models.ManagedIdentityAccessRule
		expectMsg     *string
		name          string
	}

	now := currentTime()
	positiveManagedIdentity := warmupItems.managedIdentities[0]
	testCases := []testCase{

		{
			name: "positive, nearly empty",
			toCreate: &models.ManagedIdentityAccessRule{
				ManagedIdentityID: positiveManagedIdentity.Metadata.ID,
			},
			expectCreated: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				ManagedIdentityID: positiveManagedIdentity.Metadata.ID,
			},
		},

		{
			name: "positive full",
			toCreate: &models.ManagedIdentityAccessRule{
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        positiveManagedIdentity.Metadata.ID,
				AllowedUserIDs:           []string{warmupItems.users[0].Metadata.ID},
				AllowedServiceAccountIDs: []string{warmupItems.serviceAccounts[0].Metadata.ID},
				AllowedTeamIDs:           []string{warmupItems.teams[0].Metadata.ID},
			},
			expectCreated: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        positiveManagedIdentity.Metadata.ID,
				AllowedUserIDs:           []string{warmupItems.users[0].Metadata.ID},
				AllowedServiceAccountIDs: []string{warmupItems.serviceAccounts[0].Metadata.ID},
				AllowedTeamIDs:           []string{warmupItems.teams[0].Metadata.ID},
			},
		},

		{
			name: "duplicate run stage",
			toCreate: &models.ManagedIdentityAccessRule{
				RunStage:          models.JobApplyType,
				ManagedIdentityID: positiveManagedIdentity.Metadata.ID,
			},
			expectMsg: ptr.String("Rule for run stage apply already exists"),
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
			expectMsg: invalidUUIDMsg1,
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

	// Because we cannot create a managed identity access rule with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
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
	positiveRule := warmupItems.rules[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      positiveRule.Metadata.ID,
					Version: positiveRule.Metadata.Version,
				},
				RunStage:                 models.JobApplyType,
				AllowedUserIDs:           []string{warmupItems.users[1].Metadata.ID},
				AllowedServiceAccountIDs: []string{warmupItems.serviceAccounts[1].Metadata.ID},
				AllowedTeamIDs:           []string{warmupItems.teams[1].Metadata.ID},
			},
			expectManagedIdentity: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:                   positiveRule.Metadata.ID,
					Version:              positiveRule.Metadata.Version + 1,
					CreationTimestamp:    positiveRule.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				RunStage:                 models.JobApplyType,
				ManagedIdentityID:        warmupItems.managedIdentities[0].Metadata.ID,
				AllowedUserIDs:           []string{warmupItems.users[1].Metadata.ID},
				AllowedServiceAccountIDs: []string{warmupItems.serviceAccounts[1].Metadata.ID},
				AllowedTeamIDs:           []string{warmupItems.teams[1].Metadata.ID},
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
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualManagedIdentity, err :=
				testClient.client.ManagedIdentities.UpdateManagedIdentityAccessRule(ctx, test.toUpdate)

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

	// Because we cannot create a managed identity access rule with a specific ID without going into the really
	// low-level stuff, create the warmup managed identities and then find the relevant ID.
	warmupItems, err := createWarmupManagedIdentities(ctx, testClient,
		warmupManagedIdentities{
			standardWarmupGroupsForManagedIdentities,
			standardWarmupWorkspacesForManagedIdentities,
			standardWarmupTeamsForManagedIdentities,
			standardWarmupUsersForManagedIdentities,
			standardWarmupServiceAccountsForManagedIdentities,
			standardWarmupManagedIdentities,
			standardWarmupManagedIdentityAccessRules,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toDelete  *models.ManagedIdentityAccessRule
		expectMsg *string
		name      string
	}

	positiveRule := warmupItems.rules[0]
	testCases := []testCase{
		{
			name: "positive",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      positiveRule.Metadata.ID,
					Version: positiveRule.Metadata.Version,
				},
			},
		},
		{
			name: "negative, non-existent ID",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveRule.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toDelete: &models.ManagedIdentityAccessRule{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveRule.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
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

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForManagedIdentities = []models.Group{
	{
		Description: "top level group 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForManagedIdentities = []models.Workspace{
	{
		Description: "workspace 0 for testing managed identity functions",
		FullPath:    "top-level-group-0-for-managed-identities/workspace-0-for-managed-identities",
		CreatedBy:   "someone-w0",
	},
}

// Standard warmup team(s) for test in this module:
var standardWarmupTeamsForManagedIdentities = []models.Team{
	{
		Name:        "team-a",
		Description: "team a for managed identity tests",
	},
	{
		Name:        "team-b",
		Description: "team b for managed identity tests",
	},
}

// Standard warmup user(s) for tests in this module:
var standardWarmupUsersForManagedIdentities = []models.User{
	{
		Username: "user-0",
		Email:    "user-0@example.invalid",
		Admin:    false,
	},
	{
		Username: "user-1",
		Email:    "user-1@example.invalid",
		Admin:    true,
	},
}

// Standard service account(s) for tests in this module:
var standardWarmupServiceAccountsForManagedIdentities = []models.ServiceAccount{
	{
		ResourcePath:      "sa-resource-path-0",
		Name:              "service-account-0",
		Description:       "service account 0 for testing managed identities",
		GroupID:           "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:         "someone-sa0",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
	{
		ResourcePath:      "sa-resource-path-1",
		Name:              "service-account-1",
		Description:       "service account 1 for testing managed identities",
		GroupID:           "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:         "someone-sa1",
		OIDCTrustPolicies: []models.OIDCTrustPolicy{},
	},
}

// Standard managed identities for tests in this module:
// The leading digit is to enable search filter testing.
var standardWarmupManagedIdentities = []models.ManagedIdentity{
	{
		Name:        "1-managed-identity-0",
		Description: "managed identity 0 for testing managed identities",
		GroupID:     "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:   "someone-sa0",
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "1-managed-identity-1",
		Description: "managed identity 1 for testing managed identities",
		GroupID:     "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:   "someone-sa1",
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "2-managed-identity-2",
		Description: "managed identity 2 for testing managed identities",
		GroupID:     "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:   "someone-sa2",
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "2-managed-identity-3",
		Description: "managed identity 3 for testing managed identities",
		GroupID:     "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:   "someone-sa3",
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:        "5-managed-identity-4",
		Description: "managed identity 4 for testing managed identities",
		GroupID:     "top-level-group-0-for-managed-identities", // will be fixed later
		CreatedBy:   "someone-sa4",
		// Resource path is not used when creating the object, but it is returned.
	},
}

// Standard managed identity access rules for tests in this module:
var standardWarmupManagedIdentityAccessRules = []models.ManagedIdentityAccessRule{
	{
		RunStage:                 models.JobPlanType,
		ManagedIdentityID:        "1-managed-identity-0",        // will be fixed later
		AllowedUserIDs:           []string{"user-0"},            // will be fixed later
		AllowedServiceAccountIDs: []string{"service-account-0"}, // will be fixed later
		AllowedTeamIDs:           []string{"team-a"},            // will be fixed later
	},
}

// createWarmupManagedIdentities creates some warmup managed identities for a test
// The warmup managed identities to create can be standard or otherwise.
func createWarmupManagedIdentities(ctx context.Context, testClient *testClient,
	input warmupManagedIdentities) (*warmupManagedIdentities, error) {

	// It is necessary to create at least one group and workspace
	// in order to provide the necessary IDs for the managed identities.

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	resultTeams, teamName2ID, err := createInitialTeams(ctx, testClient, input.teams)
	if err != nil {
		return nil, err
	}

	resultUsers, username2ID, err := createInitialUsers(ctx, testClient, input.users)
	if err != nil {
		return nil, err
	}

	resultServiceAccounts, serviceAccountName2ID, err := createInitialServiceAccounts(ctx, testClient,
		groupPath2ID, input.serviceAccounts)
	if err != nil {
		return nil, err
	}

	resultManagedIdentities, err := createInitialManagedIdentities(ctx, testClient,
		groupPath2ID, input.managedIdentities)
	if err != nil {
		return nil, err
	}

	managedIdentityName2ID := make(map[string]string)
	for _, mi := range resultManagedIdentities {
		managedIdentityName2ID[mi.Name] = mi.Metadata.ID
	}

	resultManagedIdentityAccessRules, err := createInitialManagedIdentityAccessRules(ctx, testClient,
		managedIdentityName2ID, username2ID, serviceAccountName2ID, teamName2ID, input.rules)
	if err != nil {
		return nil, err
	}

	return &warmupManagedIdentities{
		groups:            resultGroups,
		workspaces:        resultWorkspaces,
		teams:             resultTeams,
		users:             resultUsers,
		serviceAccounts:   resultServiceAccounts,
		managedIdentities: resultManagedIdentities,
		rules:             resultManagedIdentityAccessRules,
	}, nil
}

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
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.ResourcePath, actual.ResourcePath)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Description, actual.Description)
	assert.Equal(t, expected.GroupID, actual.GroupID)
	assert.Equal(t, expected.Data, actual.Data)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

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
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.RunStage, actual.RunStage)
	assert.Equal(t, expected.ManagedIdentityID, actual.ManagedIdentityID)
	assert.Equal(t, expected.AllowedUserIDs, actual.AllowedUserIDs)
	assert.Equal(t, expected.AllowedServiceAccountIDs, actual.AllowedServiceAccountIDs)
	assert.Equal(t, expected.AllowedTeamIDs, actual.AllowedTeamIDs)

	if checkID {
		assert.Equal(t, expected.Metadata.ID, actual.Metadata.ID)
	}
	assert.Equal(t, expected.Metadata.Version, actual.Metadata.Version)

	// Compare timestamps.
	if times != nil {
		compareTime(t, times.createLow, times.createHigh, actual.Metadata.CreationTimestamp)
		compareTime(t, times.updateLow, times.updateHigh, actual.Metadata.LastUpdatedTimestamp)
	} else {
		assert.Equal(t, expected.Metadata.CreationTimestamp, actual.Metadata.CreationTimestamp)
		assert.Equal(t, expected.Metadata.LastUpdatedTimestamp, actual.Metadata.LastUpdatedTimestamp)
	}
}

// The End.
