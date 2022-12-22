//go:build integration

package db

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// warmupWorkspaceVCSProviderLinks holds the inputs to and outputs from createWarmupWorkspaceVCSProviderLinks.
type warmupWorkspaceVCSProviderLinks struct {
	groups     []models.Group
	workspaces []models.Workspace
	providers  []models.VCSProvider
	links      []models.WorkspaceVCSProviderLink
}

func TestGetLinksByProviderID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			standardWarmupWorkspaceVCSProviderLinks,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()

	type testCase struct {
		expectLinks []models.WorkspaceVCSProviderLink
		expectMsg   *string
		name        string
		searchID    string
	}

	positiveProvider := warmupItems.providers[0]
	testCases := []testCase{
		{
			name:        "positive",
			searchID:    positiveProvider.Metadata.ID,
			expectLinks: []models.WorkspaceVCSProviderLink{warmupItems.links[0]},
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect link and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualLinks, err := testClient.client.WorkspaceVCSProviderLinks.GetLinksByProviderID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectLinks != nil {
				require.Equal(t, len(actualLinks), 1) // There should only be one result.
				compareWorkspaceVCSProviderLinks(t, &test.expectLinks[0], &actualLinks[0], false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Empty(t, actualLinks)
			}
		})
	}
}

func TestGetLinkByID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			standardWarmupWorkspaceVCSProviderLinks,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()

	type testCase struct {
		expectLink *models.WorkspaceVCSProviderLink
		expectMsg  *string
		name       string
		searchID   string
	}

	positiveLink := warmupItems.links[0]
	testCases := []testCase{
		{
			name:       "positive",
			searchID:   positiveLink.Metadata.ID,
			expectLink: &positiveLink,
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect link and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualLink, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectLink != nil {
				require.NotNil(t, actualLink)
				compareWorkspaceVCSProviderLinks(t, test.expectLink, actualLink, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualLink)
			}
		})
	}
}

func TestGetLinkByWorkspaceID(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			standardWarmupWorkspaceVCSProviderLinks,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	createdHigh := currentTime()

	type testCase struct {
		expectLink *models.WorkspaceVCSProviderLink
		expectMsg  *string
		name       string
		searchID   string
	}

	positiveWorkspace := warmupItems.workspaces[0]
	testCases := []testCase{
		{
			name:       "positive",
			searchID:   positiveWorkspace.Metadata.ID,
			expectLink: &warmupItems.links[0],
		},
		{
			name:     "negative, non-existent ID",
			searchID: nonExistentID,
			// expect link and error to be nil
		},
		{
			name:      "defective-id",
			searchID:  invalidID,
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualLink, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByWorkspaceID(ctx, test.searchID)

			checkError(t, test.expectMsg, err)

			if test.expectLink != nil {
				require.NotNil(t, actualLink)
				compareWorkspaceVCSProviderLinks(t, test.expectLink, actualLink, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &createdHigh,
				})
			} else {
				assert.Nil(t, actualLink)
			}
		})
	}
}

func TestCreateLink(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			[]models.WorkspaceVCSProviderLink{},
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	moduleDirectory := "this/is/where/configuration/is"
	tagRegex := "\\d+.\\d+$"
	tokenNonce := uuid.New().String()
	webhookID := uuid.New().String()

	type testCase struct {
		toCreate       *models.WorkspaceVCSProviderLink
		expectCreated  *models.WorkspaceVCSProviderLink
		expectMsg      *string
		expectErrorMsg string // For Tharsis errors.
		name           string
	}

	now := currentTime()
	testCases := []testCase{
		{
			name: "positive: nearly empty",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         warmupItems.workspaces[0].Metadata.ID,
				ProviderID:          warmupItems.providers[0].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				// Rest of the fields can be empty.
			},
			expectCreated: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID:         warmupItems.workspaces[0].Metadata.ID,
				ProviderID:          warmupItems.providers[0].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				GlobPatterns:        []string(nil),
			},
		},
		{
			name: "positive: full",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         warmupItems.workspaces[1].Metadata.ID,
				ProviderID:          warmupItems.providers[1].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				CreatedBy:           "creator-of-workspace-vcs-provider-links",
				WebhookID:           webhookID,
				ModuleDirectory:     &moduleDirectory,
				TagRegex:            &tagRegex,
				GlobPatterns:        []string{"**/**"},
			},
			expectCreated: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					Version:           initialResourceVersion,
					CreationTimestamp: &now,
				},
				WorkspaceID:         warmupItems.workspaces[1].Metadata.ID,
				ProviderID:          warmupItems.providers[1].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				CreatedBy:           "creator-of-workspace-vcs-provider-links",
				WebhookID:           webhookID,
				ModuleDirectory:     &moduleDirectory,
				TagRegex:            &tagRegex,
				GlobPatterns:        []string{"**/**"},
			},
		},
		{
			name: "duplicate name in same group",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         warmupItems.workspaces[0].Metadata.ID,
				ProviderID:          warmupItems.providers[0].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				// Rest of the fields can be empty.
			},
			expectErrorMsg: "workspace is already linked with a vcs provider",
		},
		{
			name: "non-existent workspace ID",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         nonExistentID,
				ProviderID:          warmupItems.providers[0].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				// Rest of the fields can be empty.
			},
			expectErrorMsg: "workspace does not exist",
		},
		{
			name: "defective workspace ID",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         invalidID,
				ProviderID:          warmupItems.providers[0].Metadata.ID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				// Rest of the fields can be empty.
			},
			expectMsg: invalidUUIDMsg1,
		},
		{
			name: "defective provider ID",
			toCreate: &models.WorkspaceVCSProviderLink{
				WorkspaceID:         warmupItems.workspaces[0].Metadata.ID,
				ProviderID:          invalidID,
				RepositoryPath:      "owner/repository",
				TokenNonce:          tokenNonce,
				Branch:              "main",
				AutoSpeculativePlan: false,
				// Rest of the fields can be empty.
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualCreated, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, test.toCreate)

			if test.expectErrorMsg != "" {
				assert.Equal(t, test.expectErrorMsg, errors.ErrorMessage(err))
			} else {
				checkError(t, test.expectMsg, err)
			}

			if test.expectCreated != nil {
				// the positive case
				require.NotNil(t, actualCreated)

				// The creation process must set the creation and last updated timestamps
				// between when the test case was created and when it the result is checked.
				whenCreated := test.expectCreated.Metadata.CreationTimestamp
				now := currentTime()

				compareWorkspaceVCSProviderLinks(t, test.expectCreated, actualCreated, false, &timeBounds{
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

func TestUpdateLink(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	createdLow := currentTime()
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			standardWarmupWorkspaceVCSProviderLinks,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}
	createdHigh := currentTime()
	warmupWorkspace := warmupItems.workspaces[0]
	warmupProvider := warmupItems.providers[0]

	type testCase struct {
		toUpdate   *models.WorkspaceVCSProviderLink
		expectLink *models.WorkspaceVCSProviderLink
		expectMsg  *string
		name       string
	}

	// Do only one positive test case, because the logic is theoretically the same for all workspace vcs provider links.
	now := currentTime()
	positiveLink := warmupItems.links[0]
	testCases := []testCase{
		{
			name: "positive",
			toUpdate: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:      positiveLink.Metadata.ID,
					Version: positiveLink.Metadata.Version,
				},
				WorkspaceID:         warmupWorkspace.Metadata.ID,
				ProviderID:          warmupProvider.Metadata.ID,
				RepositoryPath:      "owner/repository",
				Branch:              "updated/branch",
				AutoSpeculativePlan: false,
			},
			expectLink: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:                   positiveLink.Metadata.ID,
					Version:              positiveLink.Metadata.Version + 1,
					CreationTimestamp:    positiveLink.Metadata.CreationTimestamp,
					LastUpdatedTimestamp: &now,
				},
				WorkspaceID:         warmupWorkspace.Metadata.ID,
				ProviderID:          warmupProvider.Metadata.ID,
				RepositoryPath:      "owner/repository",
				Branch:              "updated/branch",
				TokenNonce:          positiveLink.TokenNonce,
				AutoSpeculativePlan: false,
				CreatedBy:           positiveLink.CreatedBy,
			},
		},
		{
			name: "negative, non-existent ID",
			toUpdate: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:      nonExistentID,
					Version: positiveLink.Metadata.Version,
				},
			},
			expectMsg: resourceVersionMismatch,
		},
		{
			name: "defective-id",
			toUpdate: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:      invalidID,
					Version: positiveLink.Metadata.Version,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			actualLink, err :=
				testClient.client.WorkspaceVCSProviderLinks.UpdateLink(ctx, test.toUpdate)

			checkError(t, test.expectMsg, err)

			now := currentTime()
			if test.expectLink != nil {
				require.NotNil(t, actualLink)
				compareWorkspaceVCSProviderLinks(t, test.expectLink, actualLink, false, &timeBounds{
					createLow:  &createdLow,
					createHigh: &createdHigh,
					updateLow:  &createdLow,
					updateHigh: &now,
				})
			} else {
				assert.Nil(t, actualLink)
			}
		})
	}
}

func TestDeleteLink(t *testing.T) {

	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Because we cannot create a link with a specific ID without going into the really
	// low-level stuff, create the warmup links and then find the relevant ID.
	warmupItems, err := createWarmupWorkspaceVCSProviderLinks(ctx, testClient,
		warmupWorkspaceVCSProviderLinks{
			standardWarmupGroupsForWorkspaceVCSProviderLinks,
			standardWarmupWorkspacesForWorkspaceVCSProviderLinks,
			standardWarmupVCSProvidersForWorkspaceVCSProviderLinks,
			standardWarmupWorkspaceVCSProviderLinks,
		})
	assert.Nil(t, err)
	if err != nil {
		// No point if warmup objects weren't all created.
		return
	}

	type testCase struct {
		toDelete  *models.WorkspaceVCSProviderLink
		expectMsg *string
		name      string
	}

	testCases := []testCase{

		{
			name: "positive",
			toDelete: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:      warmupItems.links[0].Metadata.ID,
					Version: warmupItems.links[0].Metadata.Version,
				},
			},
		},

		{
			name: "negative, non-existent ID",
			toDelete: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: nonExistentID,
				},
			},
			expectMsg: resourceVersionMismatch,
		},

		{
			name: "defective-id",
			toDelete: &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID: invalidID,
				},
			},
			expectMsg: invalidUUIDMsg1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {

			err := testClient.client.WorkspaceVCSProviderLinks.DeleteLink(ctx, test.toDelete)

			checkError(t, test.expectMsg, err)

		})
	}
}

//////////////////////////////////////////////////////////////////////////////

// Common utility structures and functions:

// Standard warmup group(s) for tests in this module:
// The create function will derive the parent path and name from the full path.
var standardWarmupGroupsForWorkspaceVCSProviderLinks = []models.Group{
	{
		Description: "top level group 0 for testing workspace vcs provider link functions",
		FullPath:    "top-level-group-0-for-workspace-vcs-provider-links",
		CreatedBy:   "someone-g0",
	},
}

// Standard warmup workspace(s) for tests in this module:
var standardWarmupWorkspacesForWorkspaceVCSProviderLinks = []models.Workspace{
	{
		Description: "workspace 0 for testing workspace vcs provider link functions",
		FullPath:    "top-level-group-0-for-workspace-vcs-provider-links/workspace-0-for-workspace-vcs-provider-links",
		CreatedBy:   "someone-w0",
	},
	{
		Description: "workspace 1 for testing workspace vcs provider link functions",
		FullPath:    "top-level-group-0-for-workspace-vcs-provider-links/workspace-1-for-workspace-vcs-provider-links",
		CreatedBy:   "someone-w1",
	},
}

// standard warmup vcs provider(s) for tests in this module:
var standardWarmupVCSProvidersForWorkspaceVCSProviderLinks = []models.VCSProvider{
	{
		Name:              "0-vcs-provider-0",
		Description:       "vcs provider 0 for testing workspace vcs provider links",
		GroupID:           "top-level-group-0-for-workspace-vcs-provider-links",
		CreatedBy:         "someone-vp0",
		Hostname:          "github.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitHubProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
	{
		Name:              "1-vcs-provider-1",
		Description:       "vcs provider 1 for testing workspace vcs provider links",
		GroupID:           "top-level-group-0-for-workspace-vcs-provider-links",
		CreatedBy:         "someone-vp0",
		Hostname:          "gitlab.com",
		OAuthClientID:     "a-client-id",
		OAuthClientSecret: "a-client-secret",
		OAuthState:        ptr.String(uuid.New().String()),
		Type:              models.GitLabProviderType,
		// Resource path is not used when creating the object, but it is returned.
	},
}

var standardWarmupWorkspaceVCSProviderLinks = []models.WorkspaceVCSProviderLink{
	{
		WorkspaceID:    "top-level-group-0-for-workspace-vcs-provider-links/workspace-0-for-workspace-vcs-provider-links",
		ProviderID:     "top-level-group-0-for-workspace-vcs-provider-links/0-vcs-provider-0",
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "owner/repository",
		CreatedBy:      "someone-wpl0",
		Branch:         "main",
	},
	{
		WorkspaceID:    "top-level-group-0-for-workspace-vcs-provider-links/workspace-1-for-workspace-vcs-provider-links",
		ProviderID:     "top-level-group-0-for-workspace-vcs-provider-links/1-vcs-provider-1",
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "owner/repository",
		CreatedBy:      "someone-wpl1",
		Branch:         "main",
	},
}

// createWarmupWorkspaceVCSProviderLinks creates workspace vcs provider links for testing.
func createWarmupWorkspaceVCSProviderLinks(ctx context.Context, testClient *testClient,
	input warmupWorkspaceVCSProviderLinks) (*warmupWorkspaceVCSProviderLinks, error) {

	// It is necessary to create at least one group and workspace
	// in order to provide the necessary IDs for the workspace vcs provider links.

	resultGroups, groupPath2ID, err := createInitialGroups(ctx, testClient, input.groups)
	if err != nil {
		return nil, err
	}

	resultWorkspaces, err := createInitialWorkspaces(ctx, testClient, groupPath2ID, input.workspaces)
	if err != nil {
		return nil, err
	}

	resultVCSProviders, err := createInitialVCSProviders(ctx, testClient,
		groupPath2ID, input.providers)
	if err != nil {
		return nil, err
	}

	workspacePath2ID := make(map[string]string, len(resultWorkspaces))
	for _, workspace := range resultWorkspaces {
		workspacePath2ID[workspace.FullPath] = workspace.Metadata.ID
	}

	providerPath2ID := make(map[string]string, len(resultVCSProviders))
	for _, provider := range resultVCSProviders {
		providerPath2ID[provider.ResourcePath] = provider.Metadata.ID
	}

	resultLinks, err := createInitialWorkspaceVCSProviderLinks(ctx, testClient,
		workspacePath2ID, providerPath2ID, input.links)
	if err != nil {
		return nil, err
	}

	return &warmupWorkspaceVCSProviderLinks{
		groups:     resultGroups,
		workspaces: resultWorkspaces,
		providers:  resultVCSProviders,
		links:      resultLinks,
	}, nil
}

// createInitialWorkspaceVCSProviderLinks creates some warmup workspace vcs provider links for a test.
func createInitialWorkspaceVCSProviderLinks(
	ctx context.Context,
	testClient *testClient,
	workspaceMap map[string]string,
	vcsProviderMap map[string]string,
	toCreate []models.WorkspaceVCSProviderLink,
) (
	[]models.WorkspaceVCSProviderLink, error) {

	result := []models.WorkspaceVCSProviderLink{}

	for _, input := range toCreate {
		input.WorkspaceID = workspaceMap[input.WorkspaceID]
		input.ProviderID = vcsProviderMap[input.ProviderID]
		created, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &input)
		if err != nil {
			return nil, fmt.Errorf("failed to create initial workspace vcs provider link: %s", err)
		}

		result = append(result, *created)
	}

	// In order to make the created-at and last-updated-at orders differ,
	// update every third object without changing any values.
	for ix, toUpdate := range result {
		if ix%3 == 0 {
			updated, err := testClient.client.WorkspaceVCSProviderLinks.UpdateLink(ctx, &toUpdate)
			if err != nil {
				return nil, fmt.Errorf("failed to update initial workspace vcs provider link: %s", err)
			}
			result[ix] = *updated
		}
	}

	return result, nil
}

// compareWorkspaceVCSProviderLinks compares two workspace vcs provider link objects,
// including bounds for creation and updated times. If times is nil, it compares
// the exact metadata timestamps.
func compareWorkspaceVCSProviderLinks(t *testing.T, expected, actual *models.WorkspaceVCSProviderLink,
	checkID bool, times *timeBounds) {

	assert.Equal(t, expected.WorkspaceID, actual.WorkspaceID)
	assert.Equal(t, expected.AutoSpeculativePlan, actual.AutoSpeculativePlan)
	assert.Equal(t, expected.Branch, actual.Branch)
	assert.Equal(t, expected.GlobPatterns, actual.GlobPatterns)
	assert.Equal(t, expected.ModuleDirectory, actual.ModuleDirectory)
	assert.Equal(t, expected.ProviderID, actual.ProviderID)
	assert.Equal(t, expected.RepositoryPath, actual.RepositoryPath)
	assert.Equal(t, expected.TagRegex, actual.TagRegex)
	assert.Equal(t, expected.TokenNonce, actual.TokenNonce)
	assert.Equal(t, expected.WebhookID, actual.WebhookID)
	assert.Equal(t, expected.CreatedBy, actual.CreatedBy)
	assert.Equal(t, expected.WebhookDisabled, actual.WebhookDisabled)

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
