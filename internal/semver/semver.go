package semver

import "github.com/hashicorp/go-version"

// IsSemverGreaterThan returns true if version v1 is greater than v2. A non pre-release version
// will take precedense over a pre-release version.
func IsSemverGreaterThan(v1 *version.Version, v2 *version.Version) bool {
	// A non pre-release version will always take precedence over a latest pre-release version
	return (v1.Prerelease() == "" && v2.Prerelease() != "") ||
		((v1.Prerelease() == "" || v2.Prerelease() != "") && v1.GreaterThan(v2))
}
