//go:build integration

package db

import (
	"context"
	"net/url"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestWorkspaceVCSProviderLinks_CreateLink(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-link",
		Description: "test group for vcs provider link",
		FullPath:    "test-group-vcs-link",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-vcs-link",
		Description:    "test workspace for vcs provider link",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-link",
		Description: "test vcs provider for link",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		workspaceID     string
		providerID      string
		repositoryPath  string
		branch          string
	}

	testCases := []testCase{
		{
			name:           "successfully create resource",
			workspaceID:    workspace.Metadata.ID,
			providerID:     vcsProvider.Metadata.ID,
			repositoryPath: "test-org/test-repo",
			branch:         "main",
		},
		{
			name:            "create will fail because workspace does not exist",
			workspaceID:     nonExistentID,
			providerID:      vcsProvider.Metadata.ID,
			repositoryPath:  "test-org/test-repo",
			branch:          "main",
			expectErrorCode: errors.ENotFound,
		},
		{
			name:            "create will fail because provider does not exist",
			workspaceID:     workspace.Metadata.ID,
			providerID:      nonExistentID,
			repositoryPath:  "test-org/test-repo",
			branch:          "main",
			expectErrorCode: errors.EConflict,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
				WorkspaceID:         test.workspaceID,
				ProviderID:          test.providerID,
				TokenNonce:          uuid.New().String(),
				RepositoryPath:      test.repositoryPath,
				Branch:              test.branch,
				CreatedBy:           "db-integration-tests",
				AutoSpeculativePlan: true,
				WebhookDisabled:     false,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, link)
			assert.Equal(t, test.workspaceID, link.WorkspaceID)
			assert.Equal(t, test.providerID, link.ProviderID)
			assert.Equal(t, test.repositoryPath, link.RepositoryPath)
			assert.Equal(t, test.branch, link.Branch)
		})
	}
}

func TestWorkspaceVCSProviderLinks_GetLinkByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-link",
		Description: "test group for get link",
		FullPath:    "test-group-get-link",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-get-link",
		Description:    "test workspace for get link",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-get-link",
		Description: "test vcs provider for get link",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectLink      bool
	}

	testCases := []testCase{
		{
			name:       "get resource by id",
			id:         link.Metadata.ID,
			expectLink: true,
		},
		{
			name: "resource with id not found",
			id:   nonExistentID,
		},
		{
			name:            "get resource with invalid id will return an error",
			id:              invalidID,
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLink {
				require.NotNil(t, result)
				assert.Equal(t, test.id, result.Metadata.ID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestWorkspaceVCSProviderLinks_GetLinksByProviderID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-links",
		Description: "test group for get links",
		FullPath:    "test-group-get-links",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-get-links",
		Description:    "test workspace for get links",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-get-links",
		Description: "test vcs provider for get links",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	_, err = testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name              string
		providerID        string
		expectResultCount int
	}

	testCases := []testCase{
		{
			name:              "return links for existing provider",
			providerID:        vcsProvider.Metadata.ID,
			expectResultCount: 1,
		},
		{
			name:              "return empty for non-existent provider",
			providerID:        nonExistentID,
			expectResultCount: 0,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			links, err := testClient.client.WorkspaceVCSProviderLinks.GetLinksByProviderID(ctx, test.providerID)

			require.NoError(t, err)
			assert.Len(t, links, test.expectResultCount)
		})
	}
}

func TestWorkspaceVCSProviderLinks_GetLinkByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-trn",
		Description: "test group for get trn",
		FullPath:    "test-group-get-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-get-trn",
		Description:    "test workspace for get trn",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-get-trn",
		Description: "test vcs provider for get trn",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	// Test getting the link by TRN
	retrievedLink, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByTRN(ctx, link.Metadata.TRN)
	require.NoError(t, err)
	require.NotNil(t, retrievedLink)

	// Verify the retrieved link matches the created one
	assert.Equal(t, link.Metadata.ID, retrievedLink.Metadata.ID)
	assert.Equal(t, link.WorkspaceID, retrievedLink.WorkspaceID)
	assert.Equal(t, link.ProviderID, retrievedLink.ProviderID)
	assert.Equal(t, link.RepositoryPath, retrievedLink.RepositoryPath)
	assert.Equal(t, link.Metadata.TRN, retrievedLink.Metadata.TRN)

	// Test with invalid TRN
	_, err = testClient.client.WorkspaceVCSProviderLinks.GetLinkByTRN(ctx, "invalid-trn")
	assert.Error(t, err)
}

func TestWorkspaceVCSProviderLinks_GetLinkByWorkspaceID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-get-workspace",
		Description: "test group for get workspace",
		FullPath:    "test-group-get-workspace",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-get-workspace",
		Description:    "test workspace for get workspace",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-get-workspace",
		Description: "test vcs provider for get workspace",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		workspaceID     string
		expectLink      bool
	}

	testCases := []testCase{
		{
			name:        "get resource by workspace id",
			workspaceID: workspace.Metadata.ID,
			expectLink:  true,
		},
		{
			name:        "resource with workspace id not found",
			workspaceID: nonExistentID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.WorkspaceVCSProviderLinks.GetLinkByWorkspaceID(ctx, test.workspaceID)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectLink {
				require.NotNil(t, result)
				assert.Equal(t, link.Metadata.ID, result.Metadata.ID)
				assert.Equal(t, link.WorkspaceID, result.WorkspaceID)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestWorkspaceVCSProviderLinks_UpdateLink(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-update-link",
		Description: "test group for update link",
		FullPath:    "test-group-update-link",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-update-link",
		Description:    "test workspace for update link",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-update-link",
		Description: "test vcs provider for update link",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		version         int
		branch          string
	}

	testCases := []testCase{
		{
			name:    "successfully update resource",
			version: 1,
			branch:  "develop",
		},
		{
			name:            "update will fail because resource version doesn't match",
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
			branch:          "feature",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			updatedLink, err := testClient.client.WorkspaceVCSProviderLinks.UpdateLink(ctx, &models.WorkspaceVCSProviderLink{
				Metadata: models.ResourceMetadata{
					ID:      link.Metadata.ID,
					Version: test.version,
				},
				Branch: test.branch,
			})

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, updatedLink)
			assert.Equal(t, test.branch, updatedLink.Branch)
		})
	}
}

func TestWorkspaceVCSProviderLinks_DeleteLink(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-delete-link",
		Description: "test group for delete link",
		FullPath:    "test-group-delete-link",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	workspace, err := testClient.client.Workspaces.CreateWorkspace(ctx, &models.Workspace{
		Name:           "test-workspace-delete-link",
		Description:    "test workspace for delete link",
		GroupID:        group.Metadata.ID,
		CreatedBy:      "db-integration-tests",
		MaxJobDuration: ptr.Int32(1),
	})
	require.NoError(t, err)

	vcsProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-delete-link",
		Description: "test vcs provider for delete link",
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
		Type:        models.GitLabProviderType,
		URL:         url.URL{Scheme: "https", Host: "gitlab.example.com"},
	})
	require.NoError(t, err)

	link, err := testClient.client.WorkspaceVCSProviderLinks.CreateLink(ctx, &models.WorkspaceVCSProviderLink{
		WorkspaceID:    workspace.Metadata.ID,
		ProviderID:     vcsProvider.Metadata.ID,
		TokenNonce:     uuid.New().String(),
		RepositoryPath: "test-org/test-repo",
		Branch:         "main",
		CreatedBy:      "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		id              string
		version         int
	}

	testCases := []testCase{
		{
			name:            "delete will fail because resource version doesn't match",
			id:              link.Metadata.ID,
			expectErrorCode: errors.EOptimisticLock,
			version:         -1,
		},
		{
			name:    "successfully delete resource",
			id:      link.Metadata.ID,
			version: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.WorkspaceVCSProviderLinks.DeleteLink(ctx, &models.WorkspaceVCSProviderLink{
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
		})
	}
}
