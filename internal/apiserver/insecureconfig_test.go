package apiserver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
)

func TestInsecureConfigWarnings(t *testing.T) {
	secureConfig := func() *config.Config {
		return &config.Config{
			SecretManagerPluginType: "awskms",
			JWSProviderPluginType:   "awskms",
			AdminUserEmail:          "admin@example.com",
			InternalRunners: []config.RunnerConfig{
				{Name: "runner", JobDispatcherType: "kubernetes"},
			},
		}
	}

	testCases := []struct {
		name          string
		modify        func(cfg *config.Config)
		expectedCount int
	}{
		{
			name:          "secure config has no warnings",
			modify:        func(_ *config.Config) {},
			expectedCount: 0,
		},
		{
			name:          "noop secret manager",
			modify:        func(cfg *config.Config) { cfg.SecretManagerPluginType = "" },
			expectedCount: 1,
		},
		{
			name: "noop secret manager with sensitive variables disabled",
			modify: func(cfg *config.Config) {
				cfg.SecretManagerPluginType = ""
				cfg.DisableSensitiveVariableFeature = true
			},
			expectedCount: 0,
		},
		{
			name:          "memory jws provider",
			modify:        func(cfg *config.Config) { cfg.JWSProviderPluginType = "memory" },
			expectedCount: 1,
		},
		{
			name:          "admin password set",
			modify:        func(cfg *config.Config) { cfg.AdminUserPassword = "s3cret" },
			expectedCount: 1,
		},
		{
			name:          "default admin password",
			modify:        func(cfg *config.Config) { cfg.AdminUserPassword = defaultAdminUserPassword },
			expectedCount: 2,
		},
		{
			name: "admin password without admin email is ignored",
			modify: func(cfg *config.Config) {
				cfg.AdminUserEmail = ""
				cfg.AdminUserPassword = defaultAdminUserPassword
			},
			expectedCount: 0,
		},
		{
			name: "email linking enabled",
			modify: func(cfg *config.Config) {
				cfg.OauthProviders = []config.IdpConfig{
					{IssuerURL: "https://idp-1.example.com", EmailLinking: true},
					{IssuerURL: "https://idp-2.example.com"},
				}
			},
			expectedCount: 1,
		},
		{
			name: "local job dispatcher",
			modify: func(cfg *config.Config) {
				cfg.InternalRunners = append(cfg.InternalRunners, config.RunnerConfig{Name: "local-runner", JobDispatcherType: "local"})
			},
			expectedCount: 1,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := secureConfig()
			test.modify(cfg)
			assert.Len(t, insecureConfigWarnings(cfg), test.expectedCount)
		})
	}
}
