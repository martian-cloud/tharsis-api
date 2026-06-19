package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewAdminLogTailStore(t *testing.T) {
	testCases := []struct {
		name       string
		pluginType string
		pluginData map[string]string
		expectErr  bool
	}{
		{name: "noop backend (empty)", pluginType: ""},
		{name: "noop backend (explicit)", pluginType: "noop"},
		{name: "memory backend", pluginType: "memory"},
		{name: "redis backend with endpoint", pluginType: "redis", pluginData: map[string]string{"redis_endpoint": "redis://127.0.0.1:6390"}},
		{name: "redis backend missing endpoint errors", pluginType: "redis", pluginData: map[string]string{}, expectErr: true},
		{name: "unknown type errors", pluginType: "bogus", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// t.Context() stops the redis backend's background writer when the test ends.
			store, err := newAdminLogTailStore(t.Context(), tc.pluginType, tc.pluginData)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, store)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, store)
		})
	}
}

func TestNewRateLimitStore(t *testing.T) {
	lg, _ := logger.NewForTest()

	t.Run("memory backend", func(t *testing.T) {
		store, err := newRateLimitStore(t.Context(), lg, "memory", nil, 100)
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("unknown type errors", func(t *testing.T) {
		store, err := newRateLimitStore(t.Context(), lg, "bogus", nil, 100)
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}

func TestNewObjectStorePlugin(t *testing.T) {
	lg, _ := logger.NewForTest()

	t.Run("filesystem backend", func(t *testing.T) {
		cfg := &config.Config{
			ObjectStorePluginType: "filesystem",
			TharsisAPIURL:         "https://test.local",
			ObjectStorePluginData: map[string]string{"directory": t.TempDir()},
		}
		store, err := newObjectStorePlugin(t.Context(), lg, cfg)
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("filesystem missing directory errors", func(t *testing.T) {
		cfg := &config.Config{ObjectStorePluginType: "filesystem", ObjectStorePluginData: map[string]string{}}
		store, err := newObjectStorePlugin(t.Context(), lg, cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
	})

	t.Run("unknown type errors", func(t *testing.T) {
		cfg := &config.Config{ObjectStorePluginType: "bogus"}
		store, err := newObjectStorePlugin(t.Context(), lg, cfg)
		assert.Error(t, err)
		assert.Nil(t, store)
	})
}

func TestNewJWSProviderPlugin(t *testing.T) {
	lg, _ := logger.NewForTest()

	t.Run("memory backend", func(t *testing.T) {
		cfg := &config.Config{JWSProviderPluginType: "memory"}
		plugin, err := newJWSProviderPlugin(t.Context(), lg, cfg)
		require.NoError(t, err)
		assert.NotNil(t, plugin)
	})

	t.Run("unknown type errors", func(t *testing.T) {
		cfg := &config.Config{JWSProviderPluginType: "bogus"}
		plugin, err := newJWSProviderPlugin(t.Context(), lg, cfg)
		assert.Error(t, err)
		assert.Nil(t, plugin)
	})
}

func TestNewSecretManagerPlugin(t *testing.T) {
	lg, _ := logger.NewForTest()

	t.Run("empty type returns noop manager", func(t *testing.T) {
		cfg := &config.Config{SecretManagerPluginType: ""}
		plugin, err := newSecretManagerPlugin(t.Context(), lg, cfg)
		require.NoError(t, err)
		assert.NotNil(t, plugin)
	})

	t.Run("unknown type errors", func(t *testing.T) {
		cfg := &config.Config{SecretManagerPluginType: "bogus"}
		plugin, err := newSecretManagerPlugin(t.Context(), lg, cfg)
		assert.Error(t, err)
		assert.Nil(t, plugin)
	})
}

func TestNewEmailProvider(t *testing.T) {
	lg, _ := logger.NewForTest()

	testCases := []struct {
		name       string
		pluginType string
		pluginData map[string]string
		expectErr  bool
	}{
		{name: "empty type returns noop", pluginType: ""},
		{
			name:       "smtp with full config",
			pluginType: "smtp",
			pluginData: map[string]string{
				"smtp_host": "mail.local", "smtp_port": "587", "from_address": "a@b.com",
				"smtp_username": "u", "smtp_password": "p",
			},
		},
		{
			name:       "smtp missing required field errors",
			pluginType: "smtp",
			pluginData: map[string]string{"smtp_host": "mail.local"},
			expectErr:  true,
		},
		{
			name:       "smtp non-integer port errors",
			pluginType: "smtp",
			pluginData: map[string]string{
				"smtp_host": "mail.local", "smtp_port": "notnum", "from_address": "a@b.com",
				"smtp_username": "u", "smtp_password": "p",
			},
			expectErr: true,
		},
		{
			name:       "plunk with config",
			pluginType: "plunk",
			pluginData: map[string]string{"endpoint": "https://plunk.local", "api_key": "key"},
		},
		{
			name:       "plunk missing api_key errors",
			pluginType: "plunk",
			pluginData: map[string]string{"endpoint": "https://plunk.local"},
			expectErr:  true,
		},
		{name: "unknown type errors", pluginType: "bogus", expectErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			provider, err := newEmailProvider(t.Context(), lg, tc.pluginType, tc.pluginData)

			if tc.expectErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, provider)
		})
	}
}

func TestNewLLMClientPlugin(t *testing.T) {
	t.Run("AI disabled returns noop client", func(t *testing.T) {
		client, err := newLLMClientPlugin(t.Context(), &config.Config{AIEnabled: false})
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("AI enabled with unknown type errors", func(t *testing.T) {
		client, err := newLLMClientPlugin(t.Context(), &config.Config{AIEnabled: true, LLMClientPluginType: "bogus"})
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}
