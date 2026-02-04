package jobexecutor

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliConfig_String(t *testing.T) {
	t.Run("credential helper only", func(t *testing.T) {
		helperName := "helper-name"
		cfg := cliConfig{credentialHelperName: &helperName}

		assert.Equal(t, "credentials_helper \"helper-name\" {}\n", cfg.String())
	})

	t.Run("mirror only", func(t *testing.T) {
		cfg := cliConfig{
			providerMirror: &providerMirrorConfig{url: "https://127.0.0.1:12345"},
		}

		expected := `provider_installation {
  network_mirror {
    url = "https://127.0.0.1:12345"
    exclude = []
  }
  direct {
    include = []
  }
}
`
		assert.Equal(t, expected, cfg.String())
	})

	t.Run("mirror with exclude hosts", func(t *testing.T) {
		cfg := cliConfig{
			providerMirror: &providerMirrorConfig{
				url:          "https://127.0.0.1:12345",
				excludeHosts: []string{"local.tharsis.example.com", "other.example.com"},
			},
		}

		expected := `provider_installation {
  network_mirror {
    url = "https://127.0.0.1:12345"
    exclude = ["local.tharsis.example.com/*/*", "other.example.com/*/*"]
  }
  direct {
    include = ["local.tharsis.example.com/*/*", "other.example.com/*/*"]
  }
}
`
		assert.Equal(t, expected, cfg.String())
	})

	t.Run("both credential helper and mirror", func(t *testing.T) {
		helperName := "helper-name"
		cfg := cliConfig{
			credentialHelperName: &helperName,
			providerMirror: &providerMirrorConfig{
				url:          "https://127.0.0.1:12345",
				excludeHosts: []string{"local.tharsis.example.com"},
			},
		}

		expected := `credentials_helper "helper-name" {}
provider_installation {
  network_mirror {
    url = "https://127.0.0.1:12345"
    exclude = ["local.tharsis.example.com/*/*"]
  }
  direct {
    include = ["local.tharsis.example.com/*/*"]
  }
}
`
		assert.Equal(t, expected, cfg.String())
	})
}

func TestSetupCliConfiguration(t *testing.T) {
	workspace := &terraformWorkspace{fullEnv: make(map[string]string)}
	helperName := "helper-name"

	err := workspace.setupCliConfiguration(cliConfig{credentialHelperName: &helperName})
	require.NoError(t, err)

	configPath := workspace.fullEnv[tfCliConfigFileEnvName]
	assert.NotEmpty(t, configPath)

	contents, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "credentials_helper \"helper-name\" {}\n", string(contents))

	// Verify cleanup is scheduled
	hasCleanup := slices.ContainsFunc(workspace.pathsToRemove, func(path string) bool {
		return strings.Contains(configPath, path)
	})
	assert.True(t, hasCleanup, "should schedule temp dir for cleanup")
}
