package registry

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestFederatedRegistryClient_GetModuleVersion(t *testing.T) {
	testCases := []struct {
		name               string
		setupMocks         func(*auth.MockIdentityProvider, *mockSdkClient)
		input              *GetModuleVersionInput
		expectedVersion    *types.TerraformModuleVersion
		expectErrorMessage string
	}{
		{
			name: "successful module version retrieval",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, mockSDKClient *mockSdkClient) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *auth.TokenInput) bool {
					return input.Typ == auth.FederatedRegistryTokenType &&
						input.Audience == "test-audience"
				})).Return([]byte("test-token"), nil)

				// Mock SDK client response
				mockSDKClient.On("GetModuleVersion", mock.Anything, mock.MatchedBy(func(input *types.GetTerraformModuleVersionInput) bool {
					return *input.ModulePath == "test-namespace/test-module/aws" &&
						*input.Version == "1.0.0"
				})).Return(&types.TerraformModuleVersion{
					Metadata: types.ResourceMetadata{ID: "module-version-id"},
					Version:  "1.0.0",
					SHASum:   "abcdef1234567890",
				}, nil)
			},
			input: &GetModuleVersionInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
					Audience: "test-audience",
				},
				ModuleNamespace: "test-namespace",
				ModuleName:      "test-module",
				ModuleSystem:    "aws",
				ModuleVersion:   "1.0.0",
			},
			expectedVersion: &types.TerraformModuleVersion{
				Metadata: types.ResourceMetadata{ID: "module-version-id"},
				Version:  "1.0.0",
				SHASum:   "abcdef1234567890",
			},
		},
		{
			name: "sdk client error",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, mockSDKClient *mockSdkClient) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)
				mockSDKClient.On("GetModuleVersion", mock.Anything, mock.Anything).Return(nil, errors.New("sdk client error"))
			},
			input: &GetModuleVersionInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
					Audience: "test-audience",
				},
				ModuleVersion: "1.0.0",
			},
			expectErrorMessage: "sdk client error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockIdentityProvider := auth.NewMockIdentityProvider(t)
			mockSDKClient := newMockSdkClient(t)

			// Setup mocks
			tc.setupMocks(mockIdentityProvider, mockSDKClient)

			// Create client with mock SDK client builder
			client := &federatedRegistryClient{
				identityProvider: mockIdentityProvider,
				sdkClientBuilder: func(_ *config.Config) (sdkClient, error) {
					return mockSDKClient, nil
				},
				endpointResolver: func(_ *http.Client, host string) (*url.URL, error) {
					return url.Parse("http://" + host)
				},
			}

			// Call the method
			version, err := client.GetModuleVersion(context.Background(), tc.input)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVersion, version)
			}
		})
	}
}

func TestFederatedRegistryClient_GetModuleAttestations(t *testing.T) {
	testCases := []struct {
		name                 string
		setupMocks           func(*auth.MockIdentityProvider, *mockSdkClient)
		input                *GetModuleAttestationsInput
		expectedAttestations []*types.TerraformModuleAttestation
		expectErrorMessage   string
	}{
		{
			name: "successful attestation retrieval - single page",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, mockSDKClient *mockSdkClient) {
				// Mock token generation
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)

				// Mock SDK client response - single page
				mockSDKClient.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *types.GetTerraformModuleAttestationsInput) bool {
					return *input.Filter.TerraformModuleVersionID == "module-version-id" &&
						*input.Filter.Digest == "abcdef123456"
				})).Return(&types.GetTerraformModuleAttestationsOutput{
					ModuleAttestations: []types.TerraformModuleAttestation{
						{Data: "attestation1"},
						{Data: "attestation2"},
					},
					PageInfo: &types.PageInfo{
						HasNextPage: false,
					},
				}, nil)
			},
			input: &GetModuleAttestationsInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
				},
				ModuleVersionID: "module-version-id",
				ModuleDigest:    "abcdef123456",
			},
			expectedAttestations: []*types.TerraformModuleAttestation{
				{Data: "attestation1"},
				{Data: "attestation2"},
			},
		},
		{
			name: "successful attestation retrieval - multiple pages",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, mockSDKClient *mockSdkClient) {
				// Mock token generation
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)

				// Mock SDK client response - first page
				mockSDKClient.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *types.GetTerraformModuleAttestationsInput) bool {
					return *input.Filter.TerraformModuleVersionID == "module-version-id" &&
						*input.Filter.Digest == "abcdef123456" &&
						input.PaginationOptions.Cursor == nil
				})).Return(&types.GetTerraformModuleAttestationsOutput{
					ModuleAttestations: []types.TerraformModuleAttestation{
						{Data: "attestation1"},
						{Data: "attestation2"},
					},
					PageInfo: &types.PageInfo{
						HasNextPage: true,
						Cursor:      "next-cursor",
					},
				}, nil)

				// Mock SDK client response - second page
				mockSDKClient.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *types.GetTerraformModuleAttestationsInput) bool {
					return *input.Filter.TerraformModuleVersionID == "module-version-id" &&
						*input.Filter.Digest == "abcdef123456" &&
						*input.PaginationOptions.Cursor == "next-cursor"
				})).Return(&types.GetTerraformModuleAttestationsOutput{
					ModuleAttestations: []types.TerraformModuleAttestation{
						{Data: "attestation3"},
					},
					PageInfo: &types.PageInfo{
						HasNextPage: false,
					},
				}, nil)
			},
			input: &GetModuleAttestationsInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
				},
				ModuleVersionID: "module-version-id",
				ModuleDigest:    "abcdef123456",
			},
			expectedAttestations: []*types.TerraformModuleAttestation{
				{Data: "attestation1"},
				{Data: "attestation2"},
				{Data: "attestation3"},
			},
		},
		{
			name: "token generation error",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, _ *mockSdkClient) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return(nil, errors.New("token generation error"))
			},
			input: &GetModuleAttestationsInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
				},
				ModuleVersionID: "module-version-id",
				ModuleDigest:    "abcdef123456",
			},
			expectErrorMessage: "failed to create Tharsis client",
		},
		{
			name: "sdk client error",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider, mockSDKClient *mockSdkClient) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)
				mockSDKClient.On("GetModuleAttestations", mock.Anything, mock.Anything).Return(nil, errors.New("sdk client error"))
			},
			input: &GetModuleAttestationsInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
				},
				ModuleVersionID: "module-version-id",
				ModuleDigest:    "abcdef123456",
			},
			expectErrorMessage: "sdk client error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockIdentityProvider := auth.NewMockIdentityProvider(t)
			mockSDKClient := &mockSdkClient{}

			// Setup mocks
			tc.setupMocks(mockIdentityProvider, mockSDKClient)

			// Create client with mock SDK client builder
			client := &federatedRegistryClient{
				identityProvider: mockIdentityProvider,
				sdkClientBuilder: func(_ *config.Config) (sdkClient, error) {
					return mockSDKClient, nil
				},
				endpointResolver: func(_ *http.Client, host string) (*url.URL, error) {
					return url.Parse("http://" + host)
				},
			}

			// Call the method
			attestations, err := client.GetModuleAttestations(context.Background(), tc.input)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, len(tc.expectedAttestations), len(attestations))
				for i, attestation := range attestations {
					assert.Equal(t, tc.expectedAttestations[i].Data, attestation.Data)
				}
			}
		})
	}
}

func TestNewFederatedRegistryToken(t *testing.T) {
	testCases := []struct {
		name               string
		setupMocks         func(*auth.MockIdentityProvider)
		input              *FederatedRegistryTokenInput
		expectErrorMessage string
	}{
		{
			name: "successful token generation",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *auth.TokenInput) bool {
					return input.Subject == gid.ToGlobalID(gid.FederatedRegistryType, "registry-id") &&
						input.Audience == "test-audience" &&
						input.Typ == auth.FederatedRegistryTokenType
				})).Return([]byte("test-token"), nil)
			},
			input: &FederatedRegistryTokenInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Audience: "test-audience",
				},
			},
		},
		{
			name: "token generation error",
			setupMocks: func(mockIdentityProvider *auth.MockIdentityProvider) {
				mockIdentityProvider.On("GenerateToken", mock.Anything, mock.Anything).Return(nil, errors.New("token generation error"))
			},
			input: &FederatedRegistryTokenInput{
				FederatedRegistry: &models.FederatedRegistry{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Audience: "test-audience",
				},
			},
			expectErrorMessage: "token generation error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockIdentityProvider := auth.NewMockIdentityProvider(t)

			// Setup mocks
			tc.setupMocks(mockIdentityProvider)

			// Set identity provider in input
			tc.input.IdentityProvider = mockIdentityProvider

			// Call the function
			token, err := NewFederatedRegistryToken(context.Background(), tc.input)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "test-token", token)
			}
		})
	}
}

func TestGetFederatedRegistries(t *testing.T) {
	testCases := []struct {
		name                string
		setupMocks          func(*db.MockFederatedRegistries, *db.MockGroups)
		input               *GetFederatedRegistriesInput
		expectedRegistries  []*models.FederatedRegistry
		expectErrorMessage  string
	}{
		{
			name: "successful retrieval - single registry",
			setupMocks: func(mockFederatedRegistries *db.MockFederatedRegistries, _ *db.MockGroups) {
				mockFederatedRegistries.On("GetFederatedRegistries", mock.Anything, mock.MatchedBy(func(input *db.GetFederatedRegistriesInput) bool {
					return len(input.Filter.GroupPaths) > 0
				})).Return(&db.FederatedRegistriesResult{
					FederatedRegistries: []*models.FederatedRegistry{
						{
							Metadata: models.ResourceMetadata{ID: "registry-id"},
							Hostname: "registry.example.com",
							GroupID:  "group-id",
						},
					},
				}, nil)
			},
			input: &GetFederatedRegistriesInput{
				DBClient:  &db.Client{},
				GroupPath: "test-group",
			},
			expectedRegistries: []*models.FederatedRegistry{
				{
					Metadata: models.ResourceMetadata{ID: "registry-id"},
					Hostname: "registry.example.com",
					GroupID:  "group-id",
				},
			},
		},
		{
			name: "successful retrieval - multiple registries with different groups",
			setupMocks: func(mockFederatedRegistries *db.MockFederatedRegistries, mockGroups *db.MockGroups) {
				mockFederatedRegistries.On("GetFederatedRegistries", mock.Anything, mock.MatchedBy(func(input *db.GetFederatedRegistriesInput) bool {
					return slices.Contains(input.Filter.GroupPaths, "parent/child") &&
						slices.Contains(input.Filter.GroupPaths, "parent")
				})).Return(&db.FederatedRegistriesResult{
					FederatedRegistries: []*models.FederatedRegistry{
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-1"},
							Hostname: "registry1.example.com",
							GroupID:  "group-id-1",
						},
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-2"},
							Hostname: "registry2.example.com",
							GroupID:  "group-id-2",
						},
					},
				}, nil)

				mockGroups.On("GetGroups", mock.Anything, mock.MatchedBy(func(input *db.GetGroupsInput) bool {
					return slices.Contains(input.Filter.GroupIDs, "group-id-1") &&
						slices.Contains(input.Filter.GroupIDs, "group-id-2")
				})).Return(&db.GroupsResult{
					Groups: []models.Group{
						{
							Metadata: models.ResourceMetadata{ID: "group-id-1"},
							FullPath: "parent/child",
						},
						{
							Metadata: models.ResourceMetadata{ID: "group-id-2"},
							FullPath: "parent",
						},
					},
				}, nil)
			},
			input: &GetFederatedRegistriesInput{
				DBClient:  &db.Client{},
				GroupPath: "parent/child",
			},
			expectedRegistries: []*models.FederatedRegistry{
				{
					Metadata: models.ResourceMetadata{ID: "registry-id-1"},
					Hostname: "registry1.example.com",
					GroupID:  "group-id-1",
				},
				{
					Metadata: models.ResourceMetadata{ID: "registry-id-2"},
					Hostname: "registry2.example.com",
					GroupID:  "group-id-2",
				},
			},
		},
		{
			name: "successful retrieval - multiple registries with same hostname",
			setupMocks: func(mockFederatedRegistries *db.MockFederatedRegistries, mockGroups *db.MockGroups) {
				mockFederatedRegistries.On("GetFederatedRegistries", mock.Anything, mock.Anything).Return(&db.FederatedRegistriesResult{
					FederatedRegistries: []*models.FederatedRegistry{
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-1"},
							Hostname: "registry.example.com",
							GroupID:  "group-id-1",
						},
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-2"},
							Hostname: "registry.example.com",
							GroupID:  "group-id-2",
						},
					},
				}, nil)

				mockGroups.On("GetGroups", mock.Anything, mock.MatchedBy(func(input *db.GetGroupsInput) bool {
					return len(input.Filter.GroupIDs) == 2
				})).Return(&db.GroupsResult{
					Groups: []models.Group{
						{
							Metadata: models.ResourceMetadata{ID: "group-id-1"},
							FullPath: "parent",
						},
						{
							Metadata: models.ResourceMetadata{ID: "group-id-2"},
							FullPath: "parent/child",
						},
					},
				}, nil)
			},
			input: &GetFederatedRegistriesInput{
				DBClient:  &db.Client{},
				GroupPath: "parent/child",
			},
			expectedRegistries: []*models.FederatedRegistry{
				{
					Metadata: models.ResourceMetadata{ID: "registry-id-2"},
					Hostname: "registry.example.com",
					GroupID:  "group-id-2",
				},
			},
		},
		{
			name: "error - missing groups",
			setupMocks: func(mockFederatedRegistries *db.MockFederatedRegistries, mockGroups *db.MockGroups) {
				mockFederatedRegistries.On("GetFederatedRegistries", mock.Anything, mock.Anything).Return(&db.FederatedRegistriesResult{
					FederatedRegistries: []*models.FederatedRegistry{
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-1"},
							Hostname: "registry.example.com",
							GroupID:  "group-id-1",
						},
						{
							Metadata: models.ResourceMetadata{ID: "registry-id-2"},
							Hostname: "registry.example.com",
							GroupID:  "group-id-2",
						},
					},
				}, nil)

				mockGroups.On("GetGroups", mock.Anything, mock.Anything).Return(&db.GroupsResult{
					Groups: []models.Group{
						{
							Metadata: models.ResourceMetadata{ID: "group-id-1"},
							FullPath: "parent",
						},
						// Missing group-id-2
					},
				}, nil)
			},
			input: &GetFederatedRegistriesInput{
				DBClient:  &db.Client{},
				GroupPath: "parent/child",
			},
			expectErrorMessage: "cannot create tokens since some groups have been deleted",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mocks
			mockFederatedRegistries := db.NewMockFederatedRegistries(t)
			mockGroups := db.NewMockGroups(t)

			// Setup mocks
			tc.setupMocks(mockFederatedRegistries, mockGroups)

			// Set mock interfaces in input
			tc.input.DBClient.FederatedRegistries = mockFederatedRegistries
			tc.input.DBClient.Groups = mockGroups

			// Call the function
			registries, err := GetFederatedRegistries(context.Background(), tc.input)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)

				// Verify the exact registries returned
				require.Equal(t, len(tc.expectedRegistries), len(registries), "Number of registries doesn't match")

				// Create maps for easier comparison
				expectedMap := make(map[string]*models.FederatedRegistry)
				for _, reg := range tc.expectedRegistries {
					expectedMap[reg.Metadata.ID] = reg
				}

				actualMap := make(map[string]*models.FederatedRegistry)
				for _, reg := range registries {
					actualMap[reg.Metadata.ID] = reg
				}

				// Verify each expected registry is in the result
				for id, expectedReg := range expectedMap {
					actualReg, exists := actualMap[id]
					assert.True(t, exists, "Expected registry with ID %s not found in result", id)
					if exists {
						assert.Equal(t, expectedReg.Hostname, actualReg.Hostname, "Registry hostname mismatch for ID %s", id)
						assert.Equal(t, expectedReg.GroupID, actualReg.GroupID, "Registry GroupID mismatch for ID %s", id)
					}
				}

				// Verify no unexpected registries
				for id := range actualMap {
					_, exists := expectedMap[id]
					assert.True(t, exists, "Unexpected registry with ID %s found in result", id)
				}
			}
		})
	}
}
