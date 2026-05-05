package token

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLeveledLogger struct {
	errorCount int
	warnCount  int
}

func (m *mockLeveledLogger) Error(_ string, _ ...interface{}) { m.errorCount++ }
func (m *mockLeveledLogger) Info(_ string, _ ...interface{})  {}
func (m *mockLeveledLogger) Debug(_ string, _ ...interface{}) {}
func (m *mockLeveledLogger) Warn(_ string, _ ...interface{})  { m.warnCount++ }

func TestServiceAccountTokenResolver(t *testing.T) {
	tests := []struct {
		name          string
		setup         func() *serviceAccountTokenResolver
		expectedToken string
		expectError   string
		expectLog     bool
	}{
		{
			name: "returns cached token when renewAt is in the future",
			setup: func() *serviceAccountTokenResolver {
				renewAt := time.Now().Add(5 * time.Minute)
				return &serviceAccountTokenResolver{
					token:   "cached-token",
					renewAt: &renewAt,
				}
			},
			expectedToken: "cached-token",
		},
		{
			name: "triggers renewal when renewAt is nil",
			setup: func() *serviceAccountTokenResolver {
				return &serviceAccountTokenResolver{
					oidcTokenFunc: func() ([]byte, error) {
						return nil, fmt.Errorf("cred helper failed")
					},
				}
			},
			expectError: "failed to get OIDC token",
			expectLog:   true,
		},
		{
			name: "triggers renewal when renewAt is in the past",
			setup: func() *serviceAccountTokenResolver {
				renewAt := time.Now().Add(-1 * time.Minute)
				return &serviceAccountTokenResolver{
					token:   "stale-token",
					renewAt: &renewAt,
					oidcTokenFunc: func() ([]byte, error) {
						return nil, fmt.Errorf("expired")
					},
				}
			},
			expectError: "failed to get OIDC token",
			expectLog:   true,
		},
		{
			name: "double-check prevents redundant renewal",
			setup: func() *serviceAccountTokenResolver {
				renewAt := time.Now().Add(5 * time.Minute)
				return &serviceAccountTokenResolver{
					token:   "fresh-token",
					renewAt: &renewAt,
					oidcTokenFunc: func() ([]byte, error) {
						return nil, fmt.Errorf("should not be called")
					},
				}
			},
			expectedToken: "fresh-token",
		},
		{
			name: "does not panic with nil logger on failure",
			setup: func() *serviceAccountTokenResolver {
				return &serviceAccountTokenResolver{
					oidcTokenFunc: func() ([]byte, error) {
						return nil, fmt.Errorf("aws sso expired")
					},
				}
			},
			expectError: "aws sso expired",
		},
		{
			name: "wraps oidc token error",
			setup: func() *serviceAccountTokenResolver {
				return &serviceAccountTokenResolver{
					oidcTokenFunc: func() ([]byte, error) {
						return nil, fmt.Errorf("aws sso expired")
					},
				}
			},
			expectError: "failed to get OIDC token: aws sso expired",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := test.setup()

			var mockLog *mockLeveledLogger
			if test.expectLog {
				mockLog = &mockLeveledLogger{}
				resolver.logger = mockLog
			}

			token, err := resolver.Token(t.Context())

			if test.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.expectError)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expectedToken, token)
			}

			if test.expectLog {
				assert.Equal(t, 1, mockLog.errorCount)
			}
		})
	}
}

func TestServiceAccountTokenResolver_Close(_ *testing.T) {
	var _ interface{ Close() error } = &serviceAccountTokenResolver{}
}
