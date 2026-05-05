package token

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Resolve(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		envVars       map[string]string
		staticTokenFn func() (string, error)
		expectError   string
		expectedToken string
		expectLogWarn bool
	}{
		{
			name:          "static token from config",
			config:        Config{StaticToken: "my-token"},
			expectedToken: "my-token",
		},
		{
			name:   "static token uses staticTokenFunc when not overridden by env",
			config: Config{StaticToken: "file-token"},
			staticTokenFn: func() (string, error) {
				return "refreshed-token", nil
			},
			expectedToken: "refreshed-token",
		},
		{
			name:   "static token env override ignores staticTokenFunc",
			config: Config{StaticToken: "file-token"},
			envVars: map[string]string{
				"THARSIS_STATIC_TOKEN": "env-token",
			},
			staticTokenFn: func() (string, error) {
				return "should-not-be-used", nil
			},
			expectedToken: "env-token",
		},
		{
			name:        "service account ID and path both set errors",
			config:      Config{ServiceAccountID: "trn:service_account:g/sa", ServiceAccountPath: "g/sa"},
			expectError: "cannot both be set",
		},
		{
			name:          "service account ID without token falls through to static",
			config:        Config{ServiceAccountID: "trn:service_account:g/sa", StaticToken: "fallback"},
			expectedToken: "fallback",
		},
		{
			name:        "no credentials errors",
			config:      Config{},
			expectError: "missing authentication credentials",
		},
		{
			name:        "service account ID without token and no static errors",
			config:      Config{ServiceAccountID: "trn:service_account:g/sa"},
			expectError: "missing authentication credentials",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for k, v := range test.envVars {
				t.Setenv(k, v)
			}

			mockLog := &mockLeveledLogger{}
			config := test.config

			resolver, err := config.Resolve(
				t.Context(),
				"http://localhost:1234",
				test.staticTokenFn,
				WithLogger(mockLog),
			)

			if test.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectError)
				if test.expectLogWarn {
					assert.Equal(t, 1, mockLog.warnCount)
				}
				return
			}

			require.NoError(t, err)

			if test.expectLogWarn {
				assert.Equal(t, 1, mockLog.warnCount)
			}

			token, err := resolver.Token(t.Context())
			require.NoError(t, err)
			assert.Equal(t, test.expectedToken, token)
		})
	}
}
