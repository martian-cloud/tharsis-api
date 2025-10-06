package resolver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestConfigQuery(t *testing.T) {
	type testCase struct {
		name            string
		isAdmin         bool
		noCaller        bool
		expectErrorCode errors.CodeType
	}

	testCases := []testCase{
		{
			name:    "admin user gets full config",
			isAdmin: true,
		},
		{
			name:    "non-admin user gets filtered config",
			isAdmin: false,
		},
		{
			name:            "no caller returns error",
			noCaller:        true,
			expectErrorCode: errors.EUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test config
			testConfig := &config.Config{
				ServerPort:            "8000",
				TLSCertFile:           "/path/to/cert",
				AdminUserEmail:        "admin@test.com",
				ObjectStorePluginData: map[string]string{"key": "value"},
				FederatedRegistryTrustPolicies: []config.FederatedRegistryTrustPolicy{
					{IssuerURL: "https://test.com"},
				},
				InternalRunners: []config.RunnerConfig{
					{Name: "test-runner"},
				},
			}

			// Create mock context and state
			state := &State{
				Config: testConfig,
			}
			ctx := state.Attach(context.Background())

			if !tc.noCaller {
				// Mock auth caller
				mockCaller := &auth.MockCaller{}
				mockCaller.On("IsAdmin").Return(tc.isAdmin)
				ctx = auth.WithCaller(ctx, mockCaller)
			}

			// Call configQuery
			result, err := configQuery(ctx)

			if tc.expectErrorCode != "" {
				assert.Error(t, err)
				assert.Nil(t, result)
				assert.Equal(t, tc.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, "8000", result.cfg.ServerPort)

			if tc.isAdmin {
				// Admin should see real values
				assert.Equal(t, "/path/to/cert", result.cfg.TLSCertFile)
				assert.Equal(t, "admin@test.com", result.cfg.AdminUserEmail)
				assert.Len(t, result.cfg.ObjectStorePluginData, 1)
				assert.Len(t, result.cfg.FederatedRegistryTrustPolicies, 1)
				assert.Len(t, result.cfg.InternalRunners, 1)
			} else {
				// Non-admin should see filtered values
				assert.Equal(t, "***", result.cfg.TLSCertFile)
				assert.Equal(t, "***", result.cfg.AdminUserEmail)
				assert.Empty(t, result.cfg.ObjectStorePluginData)
				assert.Empty(t, result.cfg.FederatedRegistryTrustPolicies)
				assert.Empty(t, result.cfg.InternalRunners)
			}
		})
	}
}

func TestFilterSensitiveFields(t *testing.T) {
	type testCase struct {
		name             string
		inputConfig      config.Config
		expectedString   string
		expectedMapLen   int
		expectedSliceLen int
	}

	testCases := []testCase{
		{
			name: "filters sensitive string fields",
			inputConfig: config.Config{
				ServerPort:     "8000",
				TLSCertFile:    "/path/to/cert",
				AdminUserEmail: "admin@test.com",
			},
			expectedString: "***",
		},
		{
			name: "filters sensitive map fields",
			inputConfig: config.Config{
				ObjectStorePluginData:   map[string]string{"key": "value"},
				SecretManagerPluginData: map[string]string{"secret": "data"},
			},
			expectedMapLen: 0,
		},
		{
			name: "filters sensitive slice fields",
			inputConfig: config.Config{
				FederatedRegistryTrustPolicies: []config.FederatedRegistryTrustPolicy{
					{IssuerURL: "https://test.com"},
				},
				InternalRunners: []config.RunnerConfig{
					{Name: "test-runner"},
				},
			},
			expectedSliceLen: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filterSensitiveFields(tc.inputConfig)

			assert.NotNil(t, result)

			// Check string fields are masked
			if tc.expectedString != "" {
				assert.Equal(t, tc.expectedString, result.TLSCertFile)
				assert.Equal(t, tc.expectedString, result.AdminUserEmail)
			}

			// Check map fields are empty
			if tc.expectedMapLen == 0 {
				assert.Len(t, result.ObjectStorePluginData, 0)
				assert.Len(t, result.SecretManagerPluginData, 0)
			}

			// Check slice fields are empty
			if tc.expectedSliceLen == 0 {
				assert.Len(t, result.FederatedRegistryTrustPolicies, 0)
				assert.Len(t, result.InternalRunners, 0)
			}

			// Non-sensitive fields should remain unchanged
			assert.Equal(t, tc.inputConfig.ServerPort, result.ServerPort)
		})
	}
}
