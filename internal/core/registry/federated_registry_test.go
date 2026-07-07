package registry

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	mtypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestNewFederatedRegistryToken(t *testing.T) {
	testCases := []struct {
		name               string
		setupMocks         func(*auth.MockSigningKeyManager)
		input              *FederatedRegistryTokenInput
		expectErrorMessage string
	}{
		{
			name: "successful token generation",
			setupMocks: func(mockSigningKeyManager *auth.MockSigningKeyManager) {
				mockSigningKeyManager.On("GenerateToken", mock.Anything, mock.MatchedBy(func(input *auth.TokenInput) bool {
					return input.Subject == gid.ToGlobalID(mtypes.FederatedRegistryModelType, "registry-id") &&
						input.Audience == "test-audience" &&
						input.Claims["type"] == auth.FederatedRegistryTokenType
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
			setupMocks: func(mockSigningKeyManager *auth.MockSigningKeyManager) {
				mockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return(nil, errors.New("token generation error"))
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
			mockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			// Setup mocks
			tc.setupMocks(mockSigningKeyManager)

			// Set identity provider in input
			tc.input.IdentityProvider = mockSigningKeyManager

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
		name               string
		setupMocks         func(*db.MockFederatedRegistries, *db.MockGroups)
		input              *GetFederatedRegistriesInput
		expectedRegistries []*models.FederatedRegistry
		expectErrorMessage string
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
