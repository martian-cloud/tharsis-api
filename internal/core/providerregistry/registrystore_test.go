package providerregistry

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func FuzzProviderObjectKeys(f *testing.F) {
	f.Add("linux", "amd64", "aws", "1.0.0")
	f.Add("darwin", "arm64", "my-provider", "2.0.0-beta.1")
	f.Add("", "", "", "")
	f.Add("..", "amd64", "aws", "1.0.0")
	f.Add("../../x", "amd64", "aws", "1.0.0")

	provider := &models.TerraformProvider{Metadata: models.ResourceMetadata{ID: "8b1d3c4e-1f2a-4b3c-9d8e-7f6a5b4c3d2e"}}
	providerVersion := &models.TerraformProviderVersion{Metadata: models.ResourceMetadata{ID: "2e3d4c5b-6a7f-4e8d-9c3b-4a2f1e3d2c1b"}}
	versionPrefix := fmt.Sprintf("registry/providers/%s/%s/", provider.Metadata.ID, providerVersion.Metadata.ID)

	f.Fuzz(func(t *testing.T, operatingSystem, architecture, name, semanticVersion string) {
		p := *provider
		p.Name = name
		pv := *providerVersion
		pv.SemanticVersion = semanticVersion
		platform := &models.TerraformProviderPlatform{OperatingSystem: operatingSystem, Architecture: architecture}

		keys := []string{
			providerVersionReadmeObjectKey(&pv, &p),
			providerVersionSHASumsObjectKey(&pv, &p),
			providerVersionSHASumsSignatureObjectKey(&pv, &p),
			providerPlatformObjectKey(platform, &pv, &p),
		}

		// A version's objects never share a key, whatever the platform and version strings are.
		seen := map[string]bool{}
		for _, key := range keys {
			assert.True(t, strings.HasPrefix(key, versionPrefix), key)
			assert.False(t, seen[key], "duplicate key %q", key)
			seen[key] = true
		}
		assert.True(t, strings.HasSuffix(keys[3], ".zip"))

		// Everything that reaches a key is validated before it is stored: the platform's OS and
		// architecture, the provider name, and the version (stored in its normalized semver form).
		// For any such input the keys stay inside this version's prefix once the filesystem object
		// store cleans them, and the platform folder names exactly this OS and architecture.
		if platform.Validate() != nil || p.Validate() != nil {
			return
		}
		v, err := version.NewSemver(semanticVersion)
		if err != nil {
			return
		}
		pv.SemanticVersion = v.String()
		platformKey := providerPlatformObjectKey(platform, &pv, &p)
		for _, key := range append(keys[:3], platformKey) {
			assert.Equal(t, "/"+key, path.Clean("/"+key), key)
			assert.True(t, strings.HasPrefix(key, versionPrefix), key)
		}
		platformDir, _, _ := strings.Cut(strings.TrimPrefix(platformKey, versionPrefix+"platforms/"), "/")
		assert.Equal(t, operatingSystem+"_"+architecture, platformDir)
		gotOS, gotArch, _ := strings.Cut(platformDir, "_")
		assert.Equal(t, operatingSystem, gotOS)
		assert.Equal(t, architecture, gotArch)
	})
}
