package jobexecutor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	cliConfigFilename      = "cli-configuration.tfrc"
	cliConfigPermissions   = os.FileMode(0o600)
	tfCliConfigFileEnvName = "TF_CLI_CONFIG_FILE"

	credentialHelperBlockFmt = "credentials_helper %q {}\n"

	// network_mirror proxies providers through Tharsis for caching.
	// direct allows Tharsis-hosted providers to bypass the mirror (no caching needed).
	providerInstallationBlockFmt = `provider_installation {
  network_mirror {
    url = %q
    exclude = [%s]
  }
  direct {
    include = [%s]
  }
}
`
)

type providerMirrorConfig struct {
	url          string
	excludeHosts []string
}

type cliConfig struct {
	credentialHelperName *string
	providerMirror       *providerMirrorConfig
}

func (c cliConfig) String() string {
	var sb strings.Builder

	if c.credentialHelperName != nil {
		fmt.Fprintf(&sb, credentialHelperBlockFmt, *c.credentialHelperName)
	}

	if c.providerMirror != nil {
		patterns := make([]string, len(c.providerMirror.excludeHosts))
		for i, host := range c.providerMirror.excludeHosts {
			patterns[i] = fmt.Sprintf("%q", host+"/*/*")
		}
		// Same patterns used for mirror exclude and direct include
		patternStr := strings.Join(patterns, ", ")
		fmt.Fprintf(&sb, providerInstallationBlockFmt, c.providerMirror.url, patternStr, patternStr)
	}

	return sb.String()
}

func (t *terraformWorkspace) setupCliConfiguration(cfg cliConfig) error {
	tempDir, err := os.MkdirTemp("", "cli-configuration-*")
	if err != nil {
		return err
	}
	t.deletePathWhenJobCompletes(tempDir)

	configPath := filepath.Join(tempDir, cliConfigFilename)
	if err := os.WriteFile(configPath, []byte(cfg.String()), cliConfigPermissions); err != nil {
		return fmt.Errorf("failed to write terraform cli configuration file: %w", err)
	}

	t.fullEnv[tfCliConfigFileEnvName] = configPath
	return nil
}
