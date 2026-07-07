// Package terraform contains core Terraform-domain logic, including resolving the supported
// Terraform CLI versions.
package terraform

import (
	"context"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// CLIVersions represents the supported Terraform CLI versions.
type CLIVersions []string

// Latest returns the latest version from the slice i.e. the last element.
func (v CLIVersions) Latest() string {
	return v[len(v)-1]
}

// Supported returns a Tharsis error if the supplied version is not supported.
func (v CLIVersions) Supported(wantVersion string) error {
	for _, supportedVersion := range v {
		if wantVersion == supportedVersion {
			return nil
		}
	}

	return errors.New("Unsupported Terraform version", errors.WithErrorCode(errors.EInvalid))
}

// GetCLIVersions returns the Terraform CLI versions that satisfy versionConstraint, fetched from the
// HashiCorp releases API. It is a pure function: it performs no authorization and no tracing, so
// callers that have already authorized their operation can invoke it directly.
func GetCLIVersions(ctx context.Context, versionConstraint string) (CLIVersions, error) {
	// Returned versions should adhere to the supplied constraint.
	constraints, err := version.NewConstraint(versionConstraint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate a Terraform CLI version constraint")
	}

	versions := &releases.Versions{
		Product:     product.Terraform,
		Constraints: constraints,
	}

	// List all the versions that meet the constraints above.
	versionSources, err := versions.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list the versions that meet the specified constraints")
	}

	// If the length here is zero, then the retrieval failed.
	if len(versionSources) == 0 {
		return nil, errors.New(
			"failed to get a list of Terraform CLI versions",
			errors.WithErrorCode(errors.EInternal))
	}

	var stringVersions CLIVersions

	// Convert version sources to their raw string version.
	for _, src := range versionSources {
		source := src.(*releases.ExactVersion)
		stringVersions = append(stringVersions, source.Version.String())
	}

	return stringVersions, nil
}
