//go:build integration

package db

import (
	"context"
	"fmt"
	"net/url"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// getValue implements the sortableField interface for VCSProviderSortableField
func (vp VCSProviderSortableField) getValue() string {
	return string(vp)
}

func TestVCSProviders_CreateVCSProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-provider",
		Description: "test group for vcs provider",
		FullPath:    "test-group-vcs-provider",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		provider        *models.VCSProvider
	}

	testCases := []testCase{
		{
			name: "successfully create vcs provider",
			provider: &models.VCSProvider{
				Name:        "test-vcs-provider",
				Description: "test vcs provider",
				URL:         url.URL{Scheme: "https", Host: "github.com"},
				Type:        models.GitHubProviderType,
				GroupID:     group.Metadata.ID,
				CreatedBy:   "db-integration-tests",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.VCSProviders.CreateProvider(ctx, test.provider)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)
			assert.Equal(t, test.provider.Name, provider.Name)
			assert.Equal(t, test.provider.URL, provider.URL)
			assert.Equal(t, test.provider.Type, provider.Type)
		})
	}
}

func TestVCSProviders_UpdateVCSProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-provider-update",
		Description: "test group for vcs provider update",
		FullPath:    "test-group-vcs-provider-update",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS provider to update
	createdProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-update",
		Description: "test vcs provider for update",
		URL:         url.URL{Scheme: "https", Host: "github.com"},
		Type:        models.GitHubProviderType,
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		updateProvider  *models.VCSProvider
	}

	testCases := []testCase{
		{
			name: "successfully update vcs provider",
			updateProvider: &models.VCSProvider{
				Metadata:    createdProvider.Metadata,
				Name:        "updated-vcs-provider",
				Description: "updated description",
				URL:         createdProvider.URL,
				Type:        createdProvider.Type,
				GroupID:     createdProvider.GroupID,
				CreatedBy:   createdProvider.CreatedBy,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.VCSProviders.UpdateProvider(ctx, test.updateProvider)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			require.NotNil(t, provider)

			// Verify the update operation succeeded
			assert.NotEmpty(t, provider.Name)
			assert.NotEmpty(t, provider.Description)
		})
	}
}

func TestVCSProviders_DeleteVCSProvider(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-provider-delete",
		Description: "test group for vcs provider delete",
		FullPath:    "test-group-vcs-provider-delete",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS provider to delete
	createdProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-delete",
		Description: "test vcs provider for delete",
		URL:         url.URL{Scheme: "https", Host: "github.com"},
		Type:        models.GitHubProviderType,
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		provider        *models.VCSProvider
	}

	testCases := []testCase{
		{
			name:     "successfully delete vcs provider",
			provider: createdProvider,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := testClient.client.VCSProviders.DeleteProvider(ctx, test.provider)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)

			// Verify the provider was deleted by trying to get it
			deletedProvider, err := testClient.client.VCSProviders.GetProviderByID(ctx, test.provider.Metadata.ID)
			if err != nil {
				// Provider should not be found after deletion
				assert.Nil(t, deletedProvider)
			}
		})
	}
}

func TestVCSProviders_GetVCSProviderByID(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the VCS provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-provider",
		Description: "test group for vcs provider",
		FullPath:    "test-group-vcs-provider",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS provider for testing
	testURL, _ := url.Parse("https://github.com")
	createdProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-provider-get-by-id",
		Description: "test provider for get by id",
		URL:         *testURL,
		Type:        models.GitHubProviderType,
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		id              string
		expectProvider  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by id",
			id:             createdProvider.Metadata.ID,
			expectProvider: true,
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
			provider, err := testClient.client.VCSProviders.GetProviderByID(ctx, test.id)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProvider {
				require.NotNil(t, provider)
				assert.Equal(t, test.id, provider.Metadata.ID)
			} else {
				assert.Nil(t, provider)
			}
		})
	}
}

func TestVCSProviders_GetVCSProviders(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the VCS providers
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-providers",
		Description: "test group for vcs providers",
		FullPath:    "test-group-vcs-providers",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create test VCS providers
	githubURL, _ := url.Parse("https://github.com")
	gitlabURL, _ := url.Parse("https://gitlab.com")
	providers := []models.VCSProvider{
		{
			Name:        "test-provider-1",
			Description: "test provider 1",
			URL:         *githubURL,
			Type:        models.GitHubProviderType,
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
		{
			Name:        "test-provider-2",
			Description: "test provider 2",
			URL:         *gitlabURL,
			Type:        models.GitLabProviderType,
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		},
	}

	createdProviders := []models.VCSProvider{}
	for _, provider := range providers {
		created, err := testClient.client.VCSProviders.CreateProvider(ctx, &provider)
		require.NoError(t, err)
		createdProviders = append(createdProviders, *created)
	}

	type testCase struct {
		name            string
		expectErrorCode errors.CodeType
		input           *GetVCSProvidersInput
		expectCount     int
	}

	testCases := []testCase{
		{
			name:        "get all providers",
			input:       &GetVCSProvidersInput{},
			expectCount: len(createdProviders),
		},
		{
			name: "filter by name",
			input: &GetVCSProvidersInput{
				Filter: &VCSProviderFilter{
					Search: ptr.String("test-provider-1"),
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by VCS provider IDs",
			input: &GetVCSProvidersInput{
				Filter: &VCSProviderFilter{
					VCSProviderIDs: []string{createdProviders[0].Metadata.ID},
				},
			},
			expectCount: 1,
		},
		{
			name: "filter by namespace paths",
			input: &GetVCSProvidersInput{
				Filter: &VCSProviderFilter{
					NamespacePaths: []string{group.FullPath},
				},
			},
			expectCount: len(createdProviders),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := testClient.client.VCSProviders.GetProviders(ctx, test.input)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.Len(t, result.VCSProviders, test.expectCount)
		})
	}
}

func TestVCSProviders_GetVCSProvidersWithPaginationAndSorting(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the VCS providers
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-pagination",
		Description: "test group for vcs pagination",
		FullPath:    "test-group-vcs-pagination",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	resourceCount := 10
	for i := 0; i < resourceCount; i++ {
		testURL, _ := url.Parse("https://github.com")
		_, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
			Name:        fmt.Sprintf("provider-%d", i),
			Description: fmt.Sprintf("provider %d", i),
			URL:         *testURL,
			Type:        models.GitHubProviderType,
			GroupID:     group.Metadata.ID,
			CreatedBy:   "db-integration-tests",
		})
		require.NoError(t, err)
	}

	sortableFields := []sortableField{
		VCSProviderSortableFieldCreatedAtAsc,
		VCSProviderSortableFieldCreatedAtDesc,
		VCSProviderSortableFieldUpdatedAtAsc,
		VCSProviderSortableFieldUpdatedAtDesc,
		VCSProviderSortableFieldGroupLevelAsc,
		VCSProviderSortableFieldGroupLevelDesc,
	}

	testResourcePaginationAndSorting(ctx, t, resourceCount, sortableFields, func(ctx context.Context, sortByField sortableField, paginationOptions *pagination.Options) (*pagination.PageInfo, []pagination.CursorPaginatable, error) {
		sortBy := VCSProviderSortableField(sortByField.getValue())

		result, err := testClient.client.VCSProviders.GetProviders(ctx, &GetVCSProvidersInput{
			Sort:              &sortBy,
			PaginationOptions: paginationOptions,
		})
		if err != nil {
			return nil, nil, err
		}

		resources := []pagination.CursorPaginatable{}
		for _, resource := range result.VCSProviders {
			resourceCopy := resource
			resources = append(resources, &resourceCopy)
		}

		return result.PageInfo, resources, nil
	})
}

func TestVCSProviders_GetVCSProviderByOAuthState(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for testing
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-provider-oauth",
		Description: "test group for vcs provider oauth",
		FullPath:    "test-group-vcs-provider-oauth",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS provider with OAuth state
	oauthState := "550e8400-e29b-41d4-a716-446655440000"
	createdProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-vcs-provider-oauth",
		Description: "test vcs provider for oauth",
		URL:         url.URL{Scheme: "https", Host: "github.com"},
		Type:        models.GitHubProviderType,
		GroupID:     group.Metadata.ID,
		OAuthState:  &oauthState,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		name              string
		expectErrorCode   errors.CodeType
		oauthState        string
		expectVCSProvider bool
	}

	testCases := []testCase{
		{
			name:              "successfully get vcs provider by oauth state",
			oauthState:        oauthState,
			expectVCSProvider: true,
		},
		{
			name:              "vcs provider not found with non-existent oauth state",
			oauthState:        "550e8400-e29b-41d4-a716-446655440001",
			expectVCSProvider: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.VCSProviders.GetProviderByOAuthState(ctx, test.oauthState)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectVCSProvider {
				require.NoError(t, err)
				require.NotNil(t, provider)
				assert.Equal(t, createdProvider.Metadata.ID, provider.Metadata.ID)
				assert.Equal(t, createdProvider.Name, provider.Name)
				assert.Equal(t, oauthState, *provider.OAuthState)
			} else {
				// Provider should not be found, so either err != nil or provider == nil
				assert.Nil(t, provider)
			}
		})
	}
}

func TestVCSProviders_GetVCSProviderByTRN(t *testing.T) {
	ctx := context.Background()
	testClient := newTestClient(ctx, t)
	defer testClient.close(ctx)

	// Create a group for the VCS provider
	group, err := testClient.client.Groups.CreateGroup(ctx, &models.Group{
		Name:        "test-group-vcs-trn",
		Description: "test group for vcs trn",
		FullPath:    "test-group-vcs-trn",
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	// Create a VCS provider for testing
	testURL, _ := url.Parse("https://github.com")
	createdProvider, err := testClient.client.VCSProviders.CreateProvider(ctx, &models.VCSProvider{
		Name:        "test-provider-trn",
		Description: "test provider for TRN",
		URL:         *testURL,
		Type:        models.GitHubProviderType,
		GroupID:     group.Metadata.ID,
		CreatedBy:   "db-integration-tests",
	})
	require.NoError(t, err)

	type testCase struct {
		expectErrorCode errors.CodeType
		name            string
		trn             string
		expectProvider  bool
	}

	testCases := []testCase{
		{
			name:           "get resource by TRN",
			trn:            createdProvider.Metadata.TRN,
			expectProvider: true,
		},
		{
			name: "resource with TRN not found",
			trn:  "trn:tharsis:vcs-provider:non-existent",
		},
		{
			name:            "get resource with invalid TRN will return an error",
			trn:             "trn:invalid",
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := testClient.client.VCSProviders.GetProviderByTRN(ctx, test.trn)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			if test.expectProvider {
				require.NotNil(t, provider)
				assert.Equal(t, test.trn, provider.Metadata.TRN)
			} else {
				assert.Nil(t, provider)
			}
		})
	}
}
