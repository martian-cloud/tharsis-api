package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	jwsplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

func TestSigningKeyManager_GenerateToken(t *testing.T) {
	now := time.Now()
	expiration := now.Add(time.Hour)

	testCases := []struct {
		name               string
		input              *TokenInput
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys, *jwsplugin.MockProvider)
	}{
		{
			name: "successful token generation",
			input: &TokenInput{
				Subject:    "test-subject",
				Expiration: &expiration,
				Claims:     map[string]string{"role": "admin"},
				JwtID:      "test-jwt-id",
				Audience:   "test-audience",
			},
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, mockJWSPlugin *jwsplugin.MockProvider) {
				activeKey := &models.AsymSigningKey{
					Metadata:   models.ResourceMetadata{ID: "key-1"},
					Status:     models.AsymSigningKeyStatusActive,
					PubKeyID:   "pub-key-1",
					PluginData: []byte("plugin-data"),
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{*activeKey},
				}, nil)

				mockJWSPlugin.On("Sign", mock.Anything, mock.Anything, "key-1", []byte("plugin-data"), "pub-key-1").Return([]byte("signed-token"), nil)
			},
		},
		{
			name: "no active key found",
			input: &TokenInput{
				Subject: "test-subject",
			},
			expectErrorMessage: "no active signing key found",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, _ *jwsplugin.MockProvider) {
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, mock.Anything).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			mockJWSPlugin := jwsplugin.NewMockProvider(t)
			mockEventManager := &events.EventManager{}

			tc.setupMocks(mockAsymSigningKeys, mockJWSPlugin)

			manager := &signingKeyManager{
				jwsPlugin:    mockJWSPlugin,
				issuerURL:    "https://test.example.com",
				dbClient:     mockDBClient,
				eventManager: mockEventManager,
				keySet:       jwk.NewSet(),
				logger:       logger,
			}

			result, err := manager.GenerateToken(ctx, tc.input)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSigningKeyManager_VerifyToken(t *testing.T) {
	// Generate RSA key pair once for the successful test case
	rsaKey1, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKey1, _ := jwk.FromRaw(rsaKey1)
	jwk.AssignKeyID(privateKey1)

	publicKey1, _ := privateKey1.PublicKey()
	publicKey1.Set(jwk.AlgorithmKey, jwa.RS256)
	publicKey1.Set(jwk.KeyUsageKey, jwk.ForSignature)
	jwk.AssignKeyID(publicKey1)

	// Generate RSA key pair once for the unsuccessful test case
	rsaKey2, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKey2, _ := jwk.FromRaw(rsaKey2)
	jwk.AssignKeyID(privateKey2)

	testCases := []struct {
		name               string
		createToken        func() string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys)
	}{
		{
			name: "successful token verification",
			createToken: func() string {
				// Create token using the shared private key
				token := jwt.New()
				token.Set(jwt.IssuerKey, "https://test.example.com")
				token.Set(jwt.SubjectKey, "test-subject")
				token.Set(jwt.AudienceKey, "tharsis")

				hdrs := jws.NewHeaders()
				err := hdrs.Set(jws.TypeKey, "JWT")
				require.NoError(t, err)

				signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateKey1, jws.WithProtectedHeaders(hdrs)))
				require.NoError(t, err)

				return string(signed)
			},
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				// Use the shared public key for verification
				keyBytes, _ := json.Marshal(publicKey1)

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{
						{
							Metadata:  models.ResourceMetadata{ID: "key-1"},
							PublicKey: keyBytes,
						},
					},
				}, nil)
			},
		},
		{
			name: "invalid token format",
			createToken: func() string {
				// Create token using the shared private key
				token := jwt.New()
				token.Set(jwt.IssuerKey, "https://test.example.com")
				token.Set(jwt.SubjectKey, "test-subject")
				token.Set(jwt.AudienceKey, "tharsis")

				hdrs := jws.NewHeaders()
				err := hdrs.Set(jws.TypeKey, "JWT")
				require.NoError(t, err)

				signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, privateKey2, jws.WithProtectedHeaders(hdrs)))
				require.NoError(t, err)

				return string(signed)
			},
			expectErrorMessage: "failed to find key",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				keyBytes, _ := json.Marshal(publicKey1)

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{
						{
							Metadata:  models.ResourceMetadata{ID: "key-1"},
							PublicKey: keyBytes,
						},
					},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			mockJWSPlugin := jwsplugin.NewMockProvider(t)
			mockEventManager := &events.EventManager{}

			tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				jwsPlugin:    mockJWSPlugin,
				issuerURL:    "https://test.example.com",
				dbClient:     mockDBClient,
				eventManager: mockEventManager,
				keySet:       jwk.NewSet(),
				logger:       logger,
			}

			result, err := manager.VerifyToken(ctx, tc.createToken())

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSigningKeyManager_GetKeys(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys) jwk.Set
	}{
		{
			name: "successful key retrieval",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) jwk.Set {
				rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				privateKey, _ := jwk.FromRaw(rsaKey)
				publicKey, _ := privateKey.PublicKey()
				publicKey.Set(jwk.KeyIDKey, "test-key-id")
				keyBytes, _ := json.Marshal(publicKey)

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{
						{
							Metadata:  models.ResourceMetadata{ID: "key-1"},
							PublicKey: keyBytes,
						},
					},
				}, nil)

				expectedKeySet := jwk.NewSet()
				expectedKeySet.AddKey(publicKey)
				return expectedKeySet
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			mockJWSPlugin := jwsplugin.NewMockProvider(t)
			mockEventManager := &events.EventManager{}

			expectedKeySet := tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				jwsPlugin:    mockJWSPlugin,
				issuerURL:    "https://test.example.com",
				dbClient:     mockDBClient,
				eventManager: mockEventManager,
				keySet:       jwk.NewSet(),
				logger:       logger,
			}

			// Initialize key set
			require.NoError(t, manager.syncKeySet(ctx))

			result, err := manager.GetKeys(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)

				expectedBytes, _ := json.Marshal(expectedKeySet)
				assert.JSONEq(t, string(expectedBytes), string(result))
			}
		})
	}
}

func TestSigningKeyManager_GetOpenIDConfig(t *testing.T) {
	t.Run("returns correct OpenID configuration", func(t *testing.T) {
		logger, _ := logger.NewForTest()

		manager := &signingKeyManager{
			issuerURL: "https://test.example.com",
			logger:    logger,
		}

		config := manager.GetOpenIDConfig()

		assert.Equal(t, "https://test.example.com", config.Issuer)
		assert.Equal(t, "https://test.example.com/oauth/discovery/keys", config.JwksURI)
		assert.Equal(t, "", config.AuthorizationEndpoint)
		assert.Equal(t, []string{"id_token"}, config.ResponseTypesSupported)
		assert.Equal(t, []string{}, config.SubjectTypesSupported)
		assert.Equal(t, []string{"RS256"}, config.IDTokenSigningAlgValuesSupported)
	})
}

func TestNewSigningKeyManager(t *testing.T) {
	testCases := []struct {
		name               string
		config             *config.Config
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys, *jwsplugin.MockProvider)
	}{
		{
			name: "successful initialization with existing active key",
			config: &config.Config{
				JWTIssuerURL:                               "https://test.example.com",
				JWSProviderPluginType:                      "test-plugin",
				AsymmetricSigningKeyRotationPeriodDays:     0,
				AsymmetricSigningKeyDecommissionPeriodDays: 0,
			},
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, _ *jwsplugin.MockProvider) {
				// Mock cleanup of failed keys
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusCreating},
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{},
				}, nil)

				// Mock existing active key
				activeKey := models.AsymSigningKey{
					Metadata:   models.ResourceMetadata{ID: "key-1"},
					Status:     models.AsymSigningKeyStatusActive,
					PluginType: "test-plugin",
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusCreating, models.AsymSigningKeyStatusActive},
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{activeKey},
				}, nil)

				// Mock sync key set
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{activeKey},
				}, nil)
			},
		},
		{
			name: "key rotation not supported error",
			config: &config.Config{
				JWTIssuerURL:                               "https://test.example.com",
				JWSProviderPluginType:                      "test-plugin",
				AsymmetricSigningKeyRotationPeriodDays:     1,
				AsymmetricSigningKeyDecommissionPeriodDays: 1,
			},
			expectErrorMessage: "does not support key rotation",
			setupMocks: func(_ *db.MockAsymSigningKeys, mockJWSPlugin *jwsplugin.MockProvider) {
				mockJWSPlugin.On("SupportsKeyRotation").Return(false)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			mockJWSPlugin := jwsplugin.NewMockProvider(t)
			mockEventManager := &events.EventManager{}

			tc.setupMocks(mockAsymSigningKeys, mockJWSPlugin)

			result, err := newSigningKeyManager(ctx, logger, mockJWSPlugin, mockDBClient, mockEventManager, tc.config, false)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestGetPrivateClaims(t *testing.T) {
	t.Run("extracts private claims correctly", func(t *testing.T) {
		token := jwt.New()
		token.Set("tharsis_role", "admin")
		token.Set("tharsis_namespace", "test-namespace")
		token.Set("iss", "issuer")

		claims := getPrivateClaims(token)

		assert.Equal(t, "admin", claims["role"])
		assert.Equal(t, "test-namespace", claims["namespace"])
		assert.NotContains(t, claims, "iss")
	})
}

func TestSigningKeyManager_syncKeySet(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys)
	}{
		{
			name: "successful sync",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				privateKey, _ := jwk.FromRaw(rsaKey)
				publicKey, _ := privateKey.PublicKey()
				publicKey.Set(jwk.KeyIDKey, "test-key-id")
				keyBytes, _ := json.Marshal(publicKey)

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{
						{
							Metadata:  models.ResourceMetadata{ID: "key-1"},
							PublicKey: keyBytes,
						},
					},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				dbClient: mockDBClient,
				keySet:   jwk.NewSet(),
				logger:   logger,
			}

			err := manager.syncKeySet(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSigningKeyManager_getActiveKey(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys) *models.AsymSigningKey
	}{
		{
			name: "successful retrieval",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) *models.AsymSigningKey {
				activeKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{ID: "key-1"},
					Status:   models.AsymSigningKeyStatusActive,
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{activeKey},
				}, nil)

				return &activeKey
			},
		},
		{
			name:               "no active key found",
			expectErrorMessage: "no active signing key found",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) *models.AsymSigningKey {
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, mock.Anything).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{},
				}, nil)

				return nil
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			expectedKey := tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				dbClient: mockDBClient,
				logger:   logger,
			}

			result, err := manager.getActiveKey(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, expectedKey, result)
			}
		})
	}
}

func TestSigningKeyManager_cleanupFailedKeys(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys)
	}{
		{
			name: "successful cleanup",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				oldTime := time.Now().Add(-10 * time.Minute)
				failedKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{
						ID:                "key-1",
						CreationTimestamp: &oldTime,
					},
					Status: models.AsymSigningKeyStatusCreating,
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusCreating},
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{failedKey},
				}, nil)

				mockAsymSigningKeys.On("DeleteAsymSigningKey", mock.Anything, &failedKey).Return(nil)
			},
		},
		{
			name: "no failed keys to cleanup",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, mock.Anything).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				dbClient: mockDBClient,
				logger:   logger,
			}

			err := manager.cleanupFailedKeys(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSigningKeyManager_createKey(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys, *jwsplugin.MockProvider)
	}{
		{
			name: "successful key creation",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, mockJWSPlugin *jwsplugin.MockProvider) {
				createdKey := &models.AsymSigningKey{
					Metadata:   models.ResourceMetadata{ID: "key-1"},
					Status:     models.AsymSigningKeyStatusCreating,
					PluginType: "test-plugin",
				}

				rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				privateKey, _ := jwk.FromRaw(rsaKey)
				publicKey, _ := privateKey.PublicKey()
				publicKey.Set(jwk.KeyIDKey, "test-key-id")

				mockAsymSigningKeys.On("CreateAsymSigningKey", mock.Anything, mock.Anything).Return(createdKey, nil)
				mockJWSPlugin.On("Create", mock.Anything, "key-1").Return(&jwsplugin.CreateKeyResponse{
					PublicKey: publicKey,
					KeyData:   []byte("plugin-data"),
				}, nil)
				mockAsymSigningKeys.On("UpdateAsymSigningKey", mock.Anything, mock.Anything).Return(createdKey, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}
			mockJWSPlugin := jwsplugin.NewMockProvider(t)

			tc.setupMocks(mockAsymSigningKeys, mockJWSPlugin)

			manager := &signingKeyManager{
				jwsPlugin:             mockJWSPlugin,
				dbClient:              mockDBClient,
				logger:                logger,
				jwsProviderPluginType: "test-plugin",
			}

			result, err := manager.createKey(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSigningKeyManager_waitForActiveKey(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys)
	}{
		{
			name: "key becomes active",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys) {
				activeKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{ID: "key-1"},
					Status:   models.AsymSigningKeyStatusActive,
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{activeKey},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}

			tc.setupMocks(mockAsymSigningKeys)

			manager := &signingKeyManager{
				dbClient: mockDBClient,
				logger:   logger,
			}

			result, err := manager.waitForActiveKey(ctx)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestSigningKeyManager_rotateKey(t *testing.T) {
	testCases := []struct {
		name               string
		expectErrorMessage string
		setupMocks         func(*db.MockAsymSigningKeys, *db.MockTransactions, *jwsplugin.MockProvider)
	}{
		{
			name: "successful key rotation",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, mockTransactions *db.MockTransactions, mockJWSPlugin *jwsplugin.MockProvider) {
				expiredKey := &models.AsymSigningKey{
					Metadata: models.ResourceMetadata{ID: "expired-key"},
					Status:   models.AsymSigningKeyStatusActive,
				}

				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockAsymSigningKeys.On("UpdateAsymSigningKey", mock.Anything, mock.Anything).Return(expiredKey, nil)
				mockAsymSigningKeys.On("CreateAsymSigningKey", mock.Anything, mock.Anything).Return(&models.AsymSigningKey{
					Metadata: models.ResourceMetadata{ID: "new-key"},
				}, nil)

				rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				privateKey, _ := jwk.FromRaw(rsaKey)
				publicKey, _ := privateKey.PublicKey()
				publicKey.Set(jwk.KeyIDKey, "new-key-id")

				mockJWSPlugin.On("Create", mock.Anything, "new-key").Return(&jwsplugin.CreateKeyResponse{
					PublicKey: publicKey,
					KeyData:   []byte("plugin-data"),
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
				Transactions:    mockTransactions,
			}
			mockJWSPlugin := jwsplugin.NewMockProvider(t)

			tc.setupMocks(mockAsymSigningKeys, mockTransactions, mockJWSPlugin)

			manager := &signingKeyManager{
				jwsPlugin:             mockJWSPlugin,
				dbClient:              mockDBClient,
				logger:                logger,
				jwsProviderPluginType: "test-plugin",
			}

			expiredKey := &models.AsymSigningKey{
				Metadata: models.ResourceMetadata{ID: "expired-key"},
				Status:   models.AsymSigningKeyStatusActive,
			}

			err := manager.rotateKey(ctx, expiredKey)

			if tc.expectErrorMessage != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectErrorMessage)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
func TestSigningKeyManager_deleteDecommissionedKeys(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func(*db.MockAsymSigningKeys, *jwsplugin.MockProvider)
	}{
		{
			name: "deletes old decommissioned keys",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, mockJWSPlugin *jwsplugin.MockProvider) {
				oldTime := time.Now().Add(-24 * time.Hour)
				decommissionedKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{
						ID:                   "old-key",
						LastUpdatedTimestamp: &oldTime,
					},
					Status:     models.AsymSigningKeyStatusDecommissioning,
					PluginData: []byte("plugin-data"),
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusDecommissioning},
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{decommissionedKey},
				}, nil)

				mockAsymSigningKeys.On("DeleteAsymSigningKey", mock.Anything, &decommissionedKey).Return(nil)
				mockJWSPlugin.On("Delete", mock.Anything, "old-key", []byte("plugin-data")).Return(nil)
			},
		},
		{
			name: "no keys to delete",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, _ *jwsplugin.MockProvider) {
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, mock.Anything).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
			}
			mockJWSPlugin := jwsplugin.NewMockProvider(t)

			tc.setupMocks(mockAsymSigningKeys, mockJWSPlugin)

			manager := &signingKeyManager{
				jwsPlugin:                mockJWSPlugin,
				dbClient:                 mockDBClient,
				logger:                   logger,
				keyDecommissioningPeriod: 12 * time.Hour,
				jwsProviderPluginType:    "test-plugin",
			}

			err := manager.deleteDecommissionedKeys(ctx)
			require.NoError(t, err)
		})
	}
}

func TestSigningKeyManager_checkForExpiredKey(t *testing.T) {
	testCases := []struct {
		name       string
		setupMocks func(*db.MockAsymSigningKeys, *db.MockTransactions, *jwsplugin.MockProvider)
	}{
		{
			name: "rotates expired key",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, mockTransactions *db.MockTransactions, mockJWSPlugin *jwsplugin.MockProvider) {
				oldTime := time.Now().Add(-24 * time.Hour)
				expiredKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{
						ID:                "expired-key",
						CreationTimestamp: &oldTime,
					},
					Status: models.AsymSigningKeyStatusActive,
				}

				// Mock getActiveKey
				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, &db.GetAsymSigningKeysInput{
					Filter: &db.AsymSigningKeyFilter{
						Status: []models.AsymSigningKeyStatus{models.AsymSigningKeyStatusActive},
					},
					PaginationOptions: &pagination.Options{
						First: ptr.Int32(1),
					},
				}).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{expiredKey},
				}, nil)

				// Mock rotateKey operations
				mockTransactions.On("BeginTx", mock.Anything).Return(context.Background(), nil)
				mockTransactions.On("RollbackTx", mock.Anything).Return(nil)
				mockTransactions.On("CommitTx", mock.Anything).Return(nil)

				mockAsymSigningKeys.On("UpdateAsymSigningKey", mock.Anything, mock.Anything).Return(&expiredKey, nil)
				mockAsymSigningKeys.On("CreateAsymSigningKey", mock.Anything, mock.Anything).Return(&models.AsymSigningKey{
					Metadata: models.ResourceMetadata{ID: "new-key"},
				}, nil)

				rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
				privateKey, _ := jwk.FromRaw(rsaKey)
				publicKey, _ := privateKey.PublicKey()
				publicKey.Set(jwk.KeyIDKey, "new-key-id")

				mockJWSPlugin.On("Create", mock.Anything, "new-key").Return(&jwsplugin.CreateKeyResponse{
					PublicKey: publicKey,
					KeyData:   []byte("plugin-data"),
				}, nil)
			},
		},
		{
			name: "key not expired",
			setupMocks: func(mockAsymSigningKeys *db.MockAsymSigningKeys, _ *db.MockTransactions, _ *jwsplugin.MockProvider) {
				recentTime := time.Now().Add(-1 * time.Hour)
				activeKey := models.AsymSigningKey{
					Metadata: models.ResourceMetadata{
						ID:                "active-key",
						CreationTimestamp: &recentTime,
					},
					Status: models.AsymSigningKeyStatusActive,
				}

				mockAsymSigningKeys.On("GetAsymSigningKeys", mock.Anything, mock.Anything).Return(&db.AsymSigningKeysResult{
					AsymSigningKeys: []models.AsymSigningKey{activeKey},
				}, nil)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			logger, _ := logger.NewForTest()

			mockAsymSigningKeys := db.NewMockAsymSigningKeys(t)
			mockTransactions := db.NewMockTransactions(t)
			mockDBClient := &db.Client{
				AsymSigningKeys: mockAsymSigningKeys,
				Transactions:    mockTransactions,
			}
			mockJWSPlugin := jwsplugin.NewMockProvider(t)

			tc.setupMocks(mockAsymSigningKeys, mockTransactions, mockJWSPlugin)

			manager := &signingKeyManager{
				jwsPlugin:             mockJWSPlugin,
				dbClient:              mockDBClient,
				logger:                logger,
				keyRotationPeriod:     6 * time.Hour,
				jwsProviderPluginType: "test-plugin",
			}

			err := manager.checkForExpiredKey(ctx)
			require.NoError(t, err)
		})
	}
}
