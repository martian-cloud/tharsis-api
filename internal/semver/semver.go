// Package semver package
package semver

import "github.com/hashicorp/go-version"

// IsSemverGreaterThan returns true if version v1 is greater than v2. A non pre-release version
// will take precedense over a pre-release version.
func IsSemverGreaterThan(v1 *version.Version, v2 *version.Version) bool {
	// A non pre-release version will always take precedence over a latest pre-release version
	return (v1.Prerelease() == "" && v2.Prerelease() != "") ||
		((v1.Prerelease() == "" || v2.Prerelease() != "") && v1.GreaterThan(v2))
}

// ExactVersion reports whether the constraint is a bare semantic version (e.g. "1.0.0" or "1.2")
// rather than a range or operator constraint (e.g. "^1.0", ">=1.0.0", "1.x"). When it is, the
// normalized version string is returned so callers can query for that exact version directly.
func ExactVersion(constraint string) (string, bool) {
	v, err := version.NewSemver(constraint)
	if err != nil {
		return "", false
	}
	return v.String(), true
}

// HighestMatching returns the highest of the given semantic versions that satisfies the constraint,
// and whether a match was found. An empty constraint matches any version (the plain highest). Values
// that do not parse as semver are ignored. Ordering uses IsSemverGreaterThan so a release outranks a
// pre-release.
func HighestMatching(versions []string, constraint string) (string, bool) {
	var constraints version.Constraints
	if constraint != "" {
		parsed, err := version.NewConstraint(constraint)
		if err != nil {
			return "", false
		}
		constraints = parsed
	}

	var best *version.Version
	var bestRaw string
	for _, raw := range versions {
		v, err := version.NewSemver(raw)
		if err != nil {
			continue
		}
		if constraints != nil && !constraints.Check(v) {
			continue
		}
		if best == nil || IsSemverGreaterThan(v, best) {
			best = v
			bestRaw = raw
		}
	}

	return bestRaw, best != nil
}
