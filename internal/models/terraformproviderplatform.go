package models

import (
	"regexp"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*TerraformProviderPlatform)(nil)

// platformComponentRegex matches a Terraform OS or architecture name (e.g. linux, darwin, amd64, arm64, 386).
// The values are joined with "_" into object keys and provider filenames, so they can't contain "_" or
// anything that could add a path segment.
var platformComponentRegex = regexp.MustCompile("^[a-z0-9]+$")

// IsValidPlatformComponent reports whether s is a valid Terraform OS or architecture name.
func IsValidPlatformComponent(s string) bool {
	return platformComponentRegex.MatchString(s)
}

// TerraformProviderPlatform represents a supported platform for a terraform provider version
type TerraformProviderPlatform struct {
	ProviderVersionID    string
	OperatingSystem      string
	Architecture         string
	SHASum               string
	Filename             string
	BinaryObjectStoreKey *string
	CreatedBy            Actor
	Metadata             ResourceMetadata
	BinaryUploaded       bool
}

// GetID returns the Metadata ID.
func (t *TerraformProviderPlatform) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformProviderPlatform) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformProviderPlatform) GetModelType() types.ModelType {
	return types.TerraformProviderPlatformModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformProviderPlatform) ResolveMetadata(key string) (*string, error) {
	return t.Metadata.resolveFieldValue(key)
}

// Validate validates the TerraformProviderPlatform model.
func (t *TerraformProviderPlatform) Validate() error {
	if !IsValidPlatformComponent(t.OperatingSystem) {
		return errors.New("invalid provider platform operating system %q", t.OperatingSystem, errors.WithErrorCode(errors.EInvalid))
	}
	if !IsValidPlatformComponent(t.Architecture) {
		return errors.New("invalid provider platform architecture %q", t.Architecture, errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}
