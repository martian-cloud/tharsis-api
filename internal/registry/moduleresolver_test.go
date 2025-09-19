package registry

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	mtypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestCommonRegistrySource_Source(t *testing.T) {
	source := &commonRegistrySource{
		source: "test-source",
	}

	result := source.Source()

	assert.Equal(t, "test-source", result)
}

func TestCommonRegistrySource_Host(t *testing.T) {
	source := &commonRegistrySource{
		host: "test-host",
	}

	result := source.Host()

	assert.Equal(t, "test-host", result)
}

func TestCommonRegistrySource_Namespace(t *testing.T) {
	source := &commonRegistrySource{
		namespace: "test-namespace",
	}

	result := source.Namespace()

	assert.Equal(t, "test-namespace", result)
}

func TestCommonRegistrySource_Name(t *testing.T) {
	source := &commonRegistrySource{
		name: "test-name",
	}

	result := source.Name()

	assert.Equal(t, "test-name", result)
}

func TestCommonRegistrySource_TargetSystem(t *testing.T) {
	source := &commonRegistrySource{
		targetSystem: "test-system",
	}

	result := source.TargetSystem()

	assert.Equal(t, "test-system", result)
}

func TestCommonRegistrySource_IsTharsisModule(t *testing.T) {
	source := &commonRegistrySource{}

	result := source.IsTharsisModule()

	assert.False(t, result)
}

func TestCommonRegistrySource_GetAttestations(t *testing.T) {
	source := &commonRegistrySource{}

	attestations, err := source.GetAttestations(context.Background(), "1.0.0", "digest123")

	require.NoError(t, err)
	assert.Empty(t, attestations)
}

func TestCommonRegistrySource_LocalRegistryModule(t *testing.T) {
	source := &commonRegistrySource{}

	module, err := source.LocalRegistryModule(context.Background())

	require.NoError(t, err)
	assert.Nil(t, module)
}

func TestCommonRegistrySource_ResolveDigest(t *testing.T) {
	source := &commonRegistrySource{}

	digest, err := source.ResolveDigest(context.Background(), "1.0.0")

	require.NoError(t, err)
	assert.Nil(t, digest)
}

func TestLocalTharsisRegistrySource_IsTharsisModule(t *testing.T) {
	source := &localTharsisRegistrySource{}

	result := source.IsTharsisModule()

	assert.True(t, result, "IsTharsisModule should return true")
}

func TestLocalTharsisRegistrySource_GetAttestations(t *testing.T) {
	testCases := []struct {
		name                 string
		moduleDigest         string
		setupMocks           func(*db.MockTerraformModuleAttestations)
		expectedAttestations []string
		expectErrorMessage   string
	}{
		{
			name:         "successful attestation retrieval",
			moduleDigest: "abcdef123456",
			setupMocks: func(mockAttestations *db.MockTerraformModuleAttestations) {
				mockAttestations.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *db.GetModuleAttestationsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.Digest == "abcdef123456"
				})).Return(&db.ModuleAttestationsResult{
					ModuleAttestations: []models.TerraformModuleAttestation{
						{Data: "attestation1"},
						{Data: "attestation2"},
					},
				}, nil)
			},
			expectedAttestations: []string{"attestation1", "attestation2"},
		},
		{
			name:         "no attestations found",
			moduleDigest: "abcdef123456",
			setupMocks: func(mockAttestations *db.MockTerraformModuleAttestations) {
				mockAttestations.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *db.GetModuleAttestationsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.Digest == "abcdef123456"
				})).Return(&db.ModuleAttestationsResult{
					ModuleAttestations: []models.TerraformModuleAttestation{},
				}, nil)
			},
			expectedAttestations: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockAttestations := db.NewMockTerraformModuleAttestations(t)
			mockDBClient.TerraformModuleAttestations = mockAttestations

			tc.setupMocks(mockAttestations)

			// Create the localTharsisRegistrySource
			source := &localTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					dbClient: mockDBClient,
					source:   "test/module/aws",
				},
				moduleID: "module-id",
			}

			// Call the method
			attestations, err := source.GetAttestations(context.Background(), "1.0.0", tc.moduleDigest)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedAttestations, attestations)
			}
		})
	}
}

func TestLocalTharsisRegistrySource_LocalRegistryModule(t *testing.T) {
	testCases := []struct {
		name               string
		setupMocks         func(*db.MockTerraformModules)
		expectedModule     *models.TerraformModule
		expectErrorMessage string
	}{
		{
			name: "successful module retrieval",
			setupMocks: func(mockModules *db.MockTerraformModules) {
				mockModules.On("GetModuleByID", mock.Anything, "module-id").Return(&models.TerraformModule{
					Metadata: models.ResourceMetadata{ID: "module-id"},
					Name:     "test-module",
					System:   "aws",
				}, nil)
			},
			expectedModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: "module-id"},
				Name:     "test-module",
				System:   "aws",
			},
		},
		{
			name: "module not found",
			setupMocks: func(mockModules *db.MockTerraformModules) {
				mockModules.On("GetModuleByID", mock.Anything, "module-id").Return(nil, nil)
			},
			expectErrorMessage: "module not found for source",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockModules := db.NewMockTerraformModules(t)
			mockDBClient.TerraformModules = mockModules

			tc.setupMocks(mockModules)

			// Create the localTharsisRegistrySource
			source := &localTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					dbClient: mockDBClient,
					source:   "test/module/aws",
				},
				moduleID: "module-id",
			}

			// Call the method
			module, err := source.LocalRegistryModule(context.Background())

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
			}
		})
	}
}

func TestLocalTharsisRegistrySource_ResolveDigest(t *testing.T) {
	testCases := []struct {
		name               string
		version            string
		setupMocks         func(*db.MockTerraformModuleVersions)
		expectedDigest     []byte
		expectErrorMessage string
	}{
		{
			name:    "successful digest resolution",
			version: "1.0.0",
			setupMocks: func(mockVersions *db.MockTerraformModuleVersions) {
				mockVersions.On("GetModuleVersions", mock.Anything, mock.MatchedBy(func(input *db.GetModuleVersionsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.SemanticVersion == "1.0.0"
				})).Return(&db.ModuleVersionsResult{
					ModuleVersions: []models.TerraformModuleVersion{
						{
							SHASum: []byte("digest123"),
						},
					},
				}, nil)
			},
			expectedDigest: []byte("digest123"),
		},
		{
			name:    "no versions found",
			version: "1.0.0",
			setupMocks: func(mockVersions *db.MockTerraformModuleVersions) {
				mockVersions.On("GetModuleVersions", mock.Anything, mock.MatchedBy(func(input *db.GetModuleVersionsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.SemanticVersion == "1.0.0"
				})).Return(&db.ModuleVersionsResult{
					ModuleVersions: []models.TerraformModuleVersion{},
				}, nil)
			},
			expectErrorMessage: "unable to find the module package for module",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockVersions := db.NewMockTerraformModuleVersions(t)
			mockDBClient.TerraformModuleVersions = mockVersions

			tc.setupMocks(mockVersions)

			// Create the localTharsisRegistrySource
			source := &localTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					dbClient: mockDBClient,
					source:   "test/module/aws",
				},
				moduleID: "module-id",
			}

			// Call the method
			digest, err := source.ResolveDigest(context.Background(), tc.version)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedDigest, digest)
			}
		})
	}
}

func TestLocalTharsisRegistrySource_ResolveSemanticVersion(t *testing.T) {
	testCases := []struct {
		name               string
		wantVersion        *string
		setupMocks         func(*db.MockTerraformModuleVersions)
		expectedVersion    string
		expectErrorMessage string
	}{
		{
			name:        "get latest version",
			wantVersion: nil,
			setupMocks: func(mockVersions *db.MockTerraformModuleVersions) {
				statusFilter := models.TerraformModuleVersionStatusUploaded
				mockVersions.On("GetModuleVersions", mock.Anything, mock.MatchedBy(func(input *db.GetModuleVersionsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.Status == statusFilter
				})).Return(&db.ModuleVersionsResult{
					ModuleVersions: []models.TerraformModuleVersion{
						{SemanticVersion: "1.0.0"},
						{SemanticVersion: "0.9.0"},
						{SemanticVersion: "0.8.0"},
					},
				}, nil)
			},
			expectedVersion: "1.0.0",
		},
		{
			name:        "get specific version",
			wantVersion: ptr.String("0.9.0"),
			setupMocks: func(mockVersions *db.MockTerraformModuleVersions) {
				statusFilter := models.TerraformModuleVersionStatusUploaded
				mockVersions.On("GetModuleVersions", mock.Anything, mock.MatchedBy(func(input *db.GetModuleVersionsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.Status == statusFilter
				})).Return(&db.ModuleVersionsResult{
					ModuleVersions: []models.TerraformModuleVersion{
						{SemanticVersion: "1.0.0"},
						{SemanticVersion: "0.9.0"},
						{SemanticVersion: "0.8.0"},
					},
				}, nil)
			},
			expectedVersion: "0.9.0",
		},
		{
			name:        "no matching version",
			wantVersion: ptr.String("2.0.0"),
			setupMocks: func(mockVersions *db.MockTerraformModuleVersions) {
				statusFilter := models.TerraformModuleVersionStatusUploaded
				mockVersions.On("GetModuleVersions", mock.Anything, mock.MatchedBy(func(input *db.GetModuleVersionsInput) bool {
					return *input.Filter.ModuleID == "module-id" && *input.Filter.Status == statusFilter
				})).Return(&db.ModuleVersionsResult{
					ModuleVersions: []models.TerraformModuleVersion{
						{SemanticVersion: "1.0.0"},
						{SemanticVersion: "0.9.0"},
					},
				}, nil)
			},
			expectErrorMessage: "no matching version found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockVersions := db.NewMockTerraformModuleVersions(t)
			mockDBClient.TerraformModuleVersions = mockVersions

			tc.setupMocks(mockVersions)

			// Create the localTharsisRegistrySource
			source := &localTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					dbClient: mockDBClient,
					source:   "test/module/aws",
				},
				moduleID: "module-id",
			}

			// Call the method
			version, err := source.ResolveSemanticVersion(context.Background(), tc.wantVersion)

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

func TestFederatedTharsisRegistrySource_IsTharsisModule(t *testing.T) {
	// Create the federatedTharsisRegistrySource
	source := &federatedTharsisRegistrySource{}

	// Call the method
	result := source.IsTharsisModule()

	// Verify the result
	assert.True(t, result, "IsTharsisModule should return true")
}

func TestFederatedTharsisRegistrySource_GetAttestations(t *testing.T) {
	semanticVersion := "1.0.0"
	moduleDigest := "abcdef123456"

	// Test cases
	testCases := []struct {
		name                 string
		moduleVersionSetup   func(*MockFederatedRegistryClient)
		attestationsSetup    func(*MockFederatedRegistryClient)
		expectedAttestations []string
	}{
		{
			name: "successful attestation retrieval",
			moduleVersionSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleVersion", mock.Anything, mock.MatchedBy(func(input *GetModuleVersionInput) bool {
					return input.ModuleVersion == semanticVersion
				})).Return(&types.TerraformModuleVersion{
					Metadata: types.ResourceMetadata{ID: "module-version-id"},
				}, nil)
			},
			attestationsSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *GetModuleAttestationsInput) bool {
					return input.ModuleVersionID == "module-version-id" && input.ModuleDigest == "abcdef123456"
				})).Return([]*types.TerraformModuleAttestation{
					{Data: "attestation1"},
					{Data: "attestation2"},
				}, nil)
			},
			expectedAttestations: []string{"attestation1", "attestation2"},
		},
		{
			name: "no attestations found",
			moduleVersionSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleVersion", mock.Anything, mock.MatchedBy(func(input *GetModuleVersionInput) bool {
					return input.ModuleVersion == semanticVersion
				})).Return(&types.TerraformModuleVersion{
					Metadata: types.ResourceMetadata{ID: "module-version-id"},
				}, nil)
			},
			attestationsSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleAttestations", mock.Anything, mock.MatchedBy(func(input *GetModuleAttestationsInput) bool {
					return input.ModuleVersionID == "module-version-id" && input.ModuleDigest == "abcdef123456"
				})).Return([]*types.TerraformModuleAttestation{}, nil)
			},
			expectedAttestations: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockFederatedRegistryClient(t)

			// Setup mocks
			tc.moduleVersionSetup(mockClient)
			tc.attestationsSetup(mockClient)

			// Create the federatedTharsisRegistrySource
			source := &federatedTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					source: "test/module/aws",
				},
				federatedRegistry: &models.FederatedRegistry{
					Hostname: "registry.example.com",
				},
				federatedRegistryClient: mockClient,
			}

			// Call the method
			attestations, err := source.GetAttestations(context.Background(), semanticVersion, moduleDigest)
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, tc.expectedAttestations, attestations)
		})
	}
}

func TestFederatedTharsisRegistrySource_ResolveDigest(t *testing.T) {
	semanticVersion := "1.0.0"

	// Test cases
	testCases := []struct {
		name               string
		moduleVersionSetup func(*MockFederatedRegistryClient)
		expectedDigest     []byte
		expectErrorMessage string
	}{
		{
			name: "successful digest resolution",
			moduleVersionSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleVersion", mock.Anything, mock.MatchedBy(func(input *GetModuleVersionInput) bool {
					return input.ModuleVersion == semanticVersion
				})).Return(&types.TerraformModuleVersion{
					SHASum: "abcdef1234567890",
				}, nil)
			},
			expectedDigest: func() []byte {
				digest, _ := hex.DecodeString("abcdef1234567890")
				return digest
			}(),
		},
		{
			name: "invalid digest format",
			moduleVersionSetup: func(mockClient *MockFederatedRegistryClient) {
				mockClient.On("GetModuleVersion", mock.Anything, mock.MatchedBy(func(input *GetModuleVersionInput) bool {
					return input.ModuleVersion == semanticVersion
				})).Return(&types.TerraformModuleVersion{
					SHASum: "invalid-hex",
				}, nil)
			},
			expectErrorMessage: "failed to decode federated registry module digest",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock client
			mockClient := NewMockFederatedRegistryClient(t)

			// Setup mocks
			tc.moduleVersionSetup(mockClient)

			// Create the federatedTharsisRegistrySource
			source := &federatedTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					source: "test/module/aws",
				},
				federatedRegistry: &models.FederatedRegistry{
					Hostname: "registry.example.com",
				},
				federatedRegistryClient: mockClient,
			}

			// Call the method
			digest, err := source.ResolveDigest(context.Background(), semanticVersion)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedDigest, digest)
			}
		})
	}
}

func TestFederatedTharsisRegistrySource_ResolveSemanticVersion(t *testing.T) {
	// Setup test server for module registry endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/namespace/module/aws/versions" {
			// Check for authorization header
			authHeader := r.Header.Get("AUTHORIZATION")
			if authHeader != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}

			// Return versions
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"modules": [
					{
						"versions": [
							{"version": "1.0.0"},
							{"version": "0.9.0"},
							{"version": "0.8.0"}
						]
					}
				]
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name                string
		wantVersion         *string
		tokenGeneratorSetup func(*auth.MockSigningKeyManager)
		expectedVersion     string
		expectErrorMessage  string
	}{
		{
			name: "get latest version",
			tokenGeneratorSetup: func(MockSigningKeyManager *auth.MockSigningKeyManager) {
				MockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)
			},
			expectedVersion: "1.0.0",
		},
		{
			name:        "get specific version",
			wantVersion: ptr.String("0.9.0"),
			tokenGeneratorSetup: func(MockSigningKeyManager *auth.MockSigningKeyManager) {
				MockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("test-token"), nil)
			},
			expectedVersion: "0.9.0",
		},
		{
			name:        "token generation error",
			wantVersion: nil,
			tokenGeneratorSetup: func(MockSigningKeyManager *auth.MockSigningKeyManager) {
				MockSigningKeyManager.On("GenerateToken", mock.Anything, mock.Anything).Return(nil, errors.New("token generation error"))
			},
			expectedVersion:    "",
			expectErrorMessage: "token generation error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock identity provider
			MockSigningKeyManager := auth.NewMockSigningKeyManager(t)

			// Setup mocks
			tc.tokenGeneratorSetup(MockSigningKeyManager)

			// Create the federatedTharsisRegistrySource
			source := &federatedTharsisRegistrySource{
				commonRegistrySource: commonRegistrySource{
					source:       "registry.example.com/namespace/module/aws",
					host:         "registry.example.com",
					namespace:    "namespace",
					name:         "module",
					targetSystem: "aws",
					registryURL:  serverURL,
					httpClient:   server.Client(),
				},
				federatedRegistry: &models.FederatedRegistry{
					Hostname: "registry.example.com",
				},
				identityProvider: MockSigningKeyManager,
			}

			// Call the method
			version, err := source.ResolveSemanticVersion(context.Background(), tc.wantVersion)

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

func TestGenericRegistrySource_IsTharsisModule(t *testing.T) {
	source := &genericRegistrySource{
		commonRegistrySource: commonRegistrySource{},
	}

	result := source.IsTharsisModule()

	assert.False(t, result, "IsTharsisModule should return false")
}

func TestGenericRegistrySource_GetAttestations(t *testing.T) {
	source := &genericRegistrySource{
		commonRegistrySource: commonRegistrySource{},
	}

	attestations, err := source.GetAttestations(context.Background(), "1.0.0", "digest123")

	require.NoError(t, err)
	assert.Empty(t, attestations)
}

func TestGenericRegistrySource_LocalRegistryModule(t *testing.T) {
	source := &genericRegistrySource{
		commonRegistrySource: commonRegistrySource{},
	}

	module, err := source.LocalRegistryModule(context.Background())

	require.NoError(t, err)
	assert.Nil(t, module)
}

func TestGenericRegistrySource_ResolveDigest(t *testing.T) {
	source := &genericRegistrySource{
		commonRegistrySource: commonRegistrySource{},
	}

	digest, err := source.ResolveDigest(context.Background(), "1.0.0")

	require.NoError(t, err)
	assert.Nil(t, digest)
}

func TestCommonRegistrySource_GetVersionsUsingModuleRegistryProtocol(t *testing.T) {
	// Setup test server for module registry endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/namespace/module-name/aws/versions" {
			// Check for authorization header
			authHeader := r.Header.Get("AUTHORIZATION")
			if authHeader == "Bearer test-token" {
				// Return versions
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{
					"modules": [
						{
							"versions": [
								{"version": "1.0.0"},
								{"version": "0.9.0"}
							]
						}
					]
				}`))
			} else if authHeader == "Bearer invalid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
			} else {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("forbidden"))
			}
		} else if r.URL.Path == "/namespace/invalid-module/aws/versions" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		} else if r.URL.Path == "/namespace/bad-json/aws/versions" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{invalid json`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name               string
		source             commonRegistrySource
		token              string
		expectedVersions   map[string]bool
		expectErrorMessage string
	}{
		{
			name: "successful version retrieval",
			source: commonRegistrySource{
				namespace:    "namespace",
				name:         "module-name",
				targetSystem: "aws",
				registryURL:  serverURL,
				httpClient:   server.Client(),
			},
			token: "test-token",
			expectedVersions: map[string]bool{
				"1.0.0": true,
				"0.9.0": true,
			},
		},
		{
			name: "unauthorized access",
			source: commonRegistrySource{
				namespace:    "namespace",
				name:         "module-name",
				targetSystem: "aws",
				registryURL:  serverURL,
				httpClient:   server.Client(),
			},
			token:              "invalid-token",
			expectErrorMessage: "unauthorized",
		},
		{
			name: "forbidden access",
			source: commonRegistrySource{
				namespace:    "namespace",
				name:         "module-name",
				targetSystem: "aws",
				registryURL:  serverURL,
				httpClient:   server.Client(),
			},
			token:              "",
			expectErrorMessage: "forbidden",
		},
		{
			name: "module not found",
			source: commonRegistrySource{
				namespace:    "namespace",
				name:         "invalid-module",
				targetSystem: "aws",
				registryURL:  serverURL,
				httpClient:   server.Client(),
			},
			token:              "test-token",
			expectErrorMessage: "failed to get module versions",
		},
		{
			name: "invalid json response",
			source: commonRegistrySource{
				namespace:    "namespace",
				name:         "bad-json",
				targetSystem: "aws",
				registryURL:  serverURL,
				httpClient:   server.Client(),
			},
			token:              "test-token",
			expectErrorMessage: "failed to unmarshal body",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create error handler function
			errorHandler := func(msg string) error {
				return errors.New(msg)
			}

			// Call the method
			versions, err := tc.source.getVersionsUsingModuleRegistryProtocol(context.Background(), tc.token, errorHandler)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedVersions, versions)
			}
		})
	}
}

func TestGenericRegistrySource_ResolveSemanticVersion(t *testing.T) {
	// Setup test server for module registry endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/namespace/module-name/aws/versions" {
			// Check for authorization header
			authHeader := r.Header.Get("AUTHORIZATION")
			if authHeader != "Bearer test-token" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("unauthorized"))
				return
			}

			// Return versions
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"modules": [
					{
						"versions": [
							{"version": "1.0.0"},
							{"version": "0.9.0"},
							{"version": "0.8.0"}
						]
					}
				]
			}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name               string
		wantVersion        *string
		tokenGetterSetup   func() (string, error)
		expectedVersion    string
		expectErrorMessage string
	}{
		{
			name: "get latest version",
			tokenGetterSetup: func() (string, error) {
				return "test-token", nil
			},
			expectedVersion: "1.0.0",
		},
		{
			name:        "get specific version",
			wantVersion: ptr.String("0.9.0"),
			tokenGetterSetup: func() (string, error) {
				return "test-token", nil
			},
			expectedVersion: "0.9.0",
		},
		{
			name:        "version not found",
			wantVersion: ptr.String("2.0.0"),
			tokenGetterSetup: func() (string, error) {
				return "test-token", nil
			},
			expectErrorMessage: "no matching version found",
		},
		{
			name: "token generation error",
			tokenGetterSetup: func() (string, error) {
				return "", errors.New("token generation error")
			},
			expectErrorMessage: "token generation error",
		},
		{
			name: "unauthorized access",
			tokenGetterSetup: func() (string, error) {
				return "invalid-token", nil
			},
			expectErrorMessage: "token in environment variable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the tokenGetter function
			tokenGetter := func(_ context.Context, _ string) (string, error) {
				return tc.tokenGetterSetup()
			}

			// Create the genericRegistrySource
			source := &genericRegistrySource{
				commonRegistrySource: commonRegistrySource{
					source:       "registry.example.com/namespace/module-name/aws",
					host:         "registry.example.com",
					namespace:    "namespace",
					name:         "module-name",
					targetSystem: "aws",
					registryURL:  serverURL,
					httpClient:   server.Client(),
				},
				tokenGetter: tokenGetter,
			}

			// Call the method
			version, err := source.ResolveSemanticVersion(context.Background(), tc.wantVersion)

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

// TestParseModuleRegistrySource tests the ParseModuleRegistrySource function
func TestParseModuleRegistrySource(t *testing.T) {
	localRegistryHost := "local.tharsis.com"
	federatedRegistryHost := "federated.tharsis.com"
	genericRegistryHost := "generic.tharsis.com"

	// Test cases
	tests := []struct {
		name                    string
		moduleSource            string
		setupMocks              func(*db.MockGroups, *db.MockTerraformModules, *db.MockTerraformModuleVersions)
		tokenGetterReturn       string
		federatedRegistryReturn *models.FederatedRegistry
		expectErrorMsg          string
		expectSourceType        string
	}{
		{
			name:           "local module source",
			moduleSource:   "./local-module",
			expectErrorMsg: "local modules are not supported",
		},
		{
			name:           "invalid module source",
			moduleSource:   "invalid-module-source",
			expectErrorMsg: "invalid module source",
		},
		{
			name:         "local tharsis registry source",
			moduleSource: fmt.Sprintf("%s/namespace/module-name/aws", localRegistryHost),
			setupMocks: func(mockGroups *db.MockGroups, mockModules *db.MockTerraformModules, _ *db.MockTerraformModuleVersions) {
				// Setup mocks for local tharsis registry
				mockGroups.On("GetGroupByTRN", mock.Anything, mtypes.GroupModelType.BuildTRN("namespace")).Return(&models.Group{
					Metadata: models.ResourceMetadata{ID: "group-id"},
				}, nil)

				mockModules.On("GetModules", mock.Anything, mock.MatchedBy(func(input *db.GetModulesInput) bool {
					return *input.Filter.RootGroupID == "group-id" &&
						*input.Filter.Name == "module-name" &&
						*input.Filter.System == "aws"
				})).Return(&db.ModulesResult{
					Modules: []models.TerraformModule{
						{
							Metadata: models.ResourceMetadata{ID: "module-id"},
						},
					},
				}, nil)
			},
			expectSourceType: "localTharsisRegistrySource",
		},
		{
			name:         "federated registry source",
			moduleSource: fmt.Sprintf("%s/namespace/module-name/aws", federatedRegistryHost),
			setupMocks:   func(*db.MockGroups, *db.MockTerraformModules, *db.MockTerraformModuleVersions) {},
			federatedRegistryReturn: &models.FederatedRegistry{
				Hostname: federatedRegistryHost,
			},
			expectSourceType: "federatedTharsisRegistrySource",
		},
		{
			name:              "generic registry source",
			moduleSource:      fmt.Sprintf("%s/namespace/module-name/aws", genericRegistryHost),
			setupMocks:        func(*db.MockGroups, *db.MockTerraformModules, *db.MockTerraformModuleVersions) {},
			tokenGetterReturn: "test-token",
			expectSourceType:  "genericRegistrySource",
		},
		{
			name:         "module not found in namespace",
			moduleSource: fmt.Sprintf("%s/namespace/module-name/aws", localRegistryHost),
			setupMocks: func(mockGroups *db.MockGroups, mockModules *db.MockTerraformModules, _ *db.MockTerraformModuleVersions) {
				mockGroups.On("GetGroupByTRN", mock.Anything, mtypes.GroupModelType.BuildTRN("namespace")).Return(&models.Group{
					Metadata: models.ResourceMetadata{ID: "group-id"},
				}, nil)

				mockModules.On("GetModules", mock.Anything, mock.MatchedBy(func(input *db.GetModulesInput) bool {
					return *input.Filter.RootGroupID == "group-id" &&
						*input.Filter.Name == "module-name" &&
						*input.Filter.System == "aws"
				})).Return(&db.ModulesResult{
					Modules: []models.TerraformModule{},
				}, nil)
			},
			expectErrorMsg: "module with name module-name and system aws not found in namespace namespace",
		},
		{
			name:           "module with subdir not supported",
			moduleSource:   fmt.Sprintf("%s/namespace/module-name/aws//subdir", localRegistryHost),
			setupMocks:     func(*db.MockGroups, *db.MockTerraformModules, *db.MockTerraformModuleVersions) {},
			expectErrorMsg: "subdir not supported when reading module from registry",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockGroupsInterface := &db.MockGroups{}
			mockModulesInterface := &db.MockTerraformModules{}
			mockModuleVersionsInterface := &db.MockTerraformModuleVersions{}

			mockDBClient.Groups = mockGroupsInterface
			mockDBClient.TerraformModules = mockModulesInterface
			mockDBClient.TerraformModuleVersions = mockModuleVersionsInterface

			if test.setupMocks != nil {
				test.setupMocks(mockGroupsInterface, mockModulesInterface, mockModuleVersionsInterface)
			}

			logger, _ := logger.NewForTest()

			// Create mock federated registry client
			mockFederatedRegistryClient := &MockFederatedRegistryClient{}

			// Create resolver
			resolver := &moduleResolver{
				dbClient:                mockDBClient,
				federatedRegistryClient: mockFederatedRegistryClient,
				logger:                  logger,
				tharsisAPIEndpoint:      fmt.Sprintf("http://%s", localRegistryHost),
				getRegistryEndpoint: func(_ *http.Client, host string) (*url.URL, error) {
					return url.Parse(fmt.Sprintf("http://%s", host))
				},
			}

			// Create token getter function
			tokenGetter := func(_ context.Context, _ string) (string, error) {
				return test.tokenGetterReturn, nil
			}

			// Create federated registry getter function
			federatedRegistryGetter := func(_ context.Context, _ string) (*models.FederatedRegistry, error) {
				return test.federatedRegistryReturn, nil
			}

			// Call the function
			result, err := resolver.ParseModuleRegistrySource(context.Background(), test.moduleSource, tokenGetter, federatedRegistryGetter)

			// Check results
			if test.expectErrorMsg != "" {
				require.Error(t, err)
				if test.expectErrorMsg != "" {
					assert.Contains(t, err.Error(), test.expectErrorMsg)
				}
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				// Check the type of the returned source
				switch test.expectSourceType {
				case "localTharsisRegistrySource":
					_, ok := result.(*localTharsisRegistrySource)
					assert.True(t, ok, "Expected localTharsisRegistrySource but got different type")
				case "federatedTharsisRegistrySource":
					_, ok := result.(*federatedTharsisRegistrySource)
					assert.True(t, ok, "Expected federatedTharsisRegistrySource but got different type")
				case "genericRegistrySource":
					_, ok := result.(*genericRegistrySource)
					assert.True(t, ok, "Expected genericRegistrySource but got different type")
				}

				// Check common properties
				assert.Equal(t, test.moduleSource, result.Source())
			}
		})
	}
}

// TestGetModuleByAddress tests the GetModuleByAddress function
func TestGetModuleByAddress(t *testing.T) {
	testCases := []struct {
		name               string
		namespace          string
		moduleName         string
		system             string
		setupMocks         func(*db.MockGroups, *db.MockTerraformModules)
		expectedModule     *models.TerraformModule
		expectErrorMessage string
	}{
		{
			name:       "successful module retrieval",
			namespace:  "test-namespace",
			moduleName: "test-module",
			system:     "aws",
			setupMocks: func(mockGroups *db.MockGroups, mockModules *db.MockTerraformModules) {
				mockGroups.On("GetGroupByTRN", mock.Anything, mtypes.GroupModelType.BuildTRN("test-namespace")).Return(&models.Group{
					Metadata: models.ResourceMetadata{ID: "group-id"},
				}, nil)

				mockModules.On("GetModules", mock.Anything, mock.MatchedBy(func(input *db.GetModulesInput) bool {
					return *input.Filter.RootGroupID == "group-id" &&
						*input.Filter.Name == "test-module" &&
						*input.Filter.System == "aws"
				})).Return(&db.ModulesResult{
					Modules: []models.TerraformModule{
						{
							Metadata: models.ResourceMetadata{ID: "module-id"},
							Name:     "test-module",
							System:   "aws",
						},
					},
				}, nil)
			},
			expectedModule: &models.TerraformModule{
				Metadata: models.ResourceMetadata{ID: "module-id"},
				Name:     "test-module",
				System:   "aws",
			},
		},
		{
			name:       "namespace not found",
			namespace:  "non-existent-namespace",
			moduleName: "test-module",
			system:     "aws",
			setupMocks: func(mockGroups *db.MockGroups, _ *db.MockTerraformModules) {
				mockGroups.On("GetGroupByTRN", mock.Anything, mtypes.GroupModelType.BuildTRN("non-existent-namespace")).Return(nil, nil)
			},
			expectErrorMessage: "namespace non-existent-namespace not found",
		},
		{
			name:       "module not found in namespace",
			namespace:  "test-namespace",
			moduleName: "non-existent-module",
			system:     "aws",
			setupMocks: func(mockGroups *db.MockGroups, mockModules *db.MockTerraformModules) {
				mockGroups.On("GetGroupByTRN", mock.Anything, mtypes.GroupModelType.BuildTRN("test-namespace")).Return(&models.Group{
					Metadata: models.ResourceMetadata{ID: "group-id"},
				}, nil)

				mockModules.On("GetModules", mock.Anything, mock.MatchedBy(func(input *db.GetModulesInput) bool {
					return *input.Filter.RootGroupID == "group-id" &&
						*input.Filter.Name == "non-existent-module" &&
						*input.Filter.System == "aws"
				})).Return(&db.ModulesResult{
					Modules: []models.TerraformModule{},
				}, nil)
			},
			expectErrorMessage: "module with name non-existent-module and system aws not found in namespace test-namespace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			mockDBClient := &db.Client{}
			mockGroups := db.NewMockGroups(t)
			mockModules := db.NewMockTerraformModules(t)
			mockDBClient.Groups = mockGroups
			mockDBClient.TerraformModules = mockModules

			tc.setupMocks(mockGroups, mockModules)

			// Call the function
			module, err := GetModuleByAddress(context.Background(), mockDBClient, tc.namespace, tc.moduleName, tc.system)

			// Verify results
			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedModule, module)
			}
		})
	}
}

// TestGetLatestMatchingVersion tests the getLatestMatchingVersion function
// with minimal overhead.
func TestGetLatestMatchingVersion(t *testing.T) {

	versions := map[string]bool{
		"0.0.1": true,
		"0.0.2": true,
		"0.0.3": true,
		"2.1.0": true,
	}

	// Test cases:
	tests := []struct {
		expectError error
		constraints *string
		name        string
		expected    string
	}{
		{
			name:        "invalid range string",
			constraints: ptr.String(""),
			expected:    "",
			expectError: fmt.Errorf("failed to parse wanted version range string: Malformed constraint: "),
		},
		{
			name:        "no constraint, return latest of all",
			constraints: nil,
			expected:    "2.1.0",
		},
		{
			name:        "exact match",
			constraints: ptr.String("0.0.2"),
			expected:    "0.0.2",
		},
		{
			name:        "exact match but does not exist",
			constraints: ptr.String("1.2.1"),
			expected:    "",
			expectError: fmt.Errorf("no matching version found"),
		},
		{
			name:        "less than",
			constraints: ptr.String("< 1.0"),
			expected:    "0.0.3",
		},
		{
			name:        "less than or equal, 0.0.1",
			constraints: ptr.String("<= 0.0.1"),
			expected:    "0.0.1",
		},
		{
			name:        "less than or equal, 0.0.2",
			constraints: ptr.String("<= 0.0.2"),
			expected:    "0.0.2",
		},
		{
			name:        "less than or equal, 0.0.3",
			constraints: ptr.String("<= 0.0.3"),
			expected:    "0.0.3",
		},
		{
			name:        "greater than",
			constraints: ptr.String("> 1.0"),
			expected:    "2.1.0",
		},
		{
			name:        "between exclusive",
			constraints: ptr.String("> 0.0.1 , < 0.0.3"),
			expected:    "0.0.2",
		},
		{
			name:        "between inclusive",
			constraints: ptr.String(">= 0.0.1 , <= 0.0.3"),
			expected:    "0.0.3",
		},
		{
			name:        "contradictory",
			constraints: ptr.String("< 0.0.1 , > 0.0.3"),
			expected:    "nil",
			expectError: fmt.Errorf("no matching version found"),
		},
	}

	for _, test := range tests {
		got, err := getLatestMatchingVersion(versions, test.constraints)
		assert.Equal(t, test.expectError, err)
		if (err == nil) && (test.expectError == nil) {
			// Don't report noise if there is or should have been an error.
			assert.Equal(t, test.expected, got)
		}
	}
}

// The End.
