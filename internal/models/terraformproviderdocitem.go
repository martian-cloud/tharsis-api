package models

import (
	"regexp"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*TerraformProviderDocItem)(nil)

// maxTerraformProviderDocItemFilenameLength caps a doc item's filename length.
const maxTerraformProviderDocItemFilenameLength = 255

// docItemFilenameExtension is the only accepted documentation file extension; the stored filename keeps its
// extension so other formats can be distinguished if they are supported later.
const docItemFilenameExtension = ".md"

// docItemFilenameRegex matches a doc item filename: letters, numbers, '.', '_', and '-', with no leading
// dot, which also rejects "." and "..". The filename is interpolated into object keys and the UI's item URL.
var docItemFilenameRegex = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9._-]*$`)

// TerraformProviderDocItemCategory represents the documentation category a doc item belongs to.
type TerraformProviderDocItemCategory string

// TerraformProviderDocItemCategory constants
const (
	TerraformProviderDocItemCategoryOverview           TerraformProviderDocItemCategory = "overview"
	TerraformProviderDocItemCategoryResources          TerraformProviderDocItemCategory = "resources"
	TerraformProviderDocItemCategoryDataSources        TerraformProviderDocItemCategory = "data_sources"
	TerraformProviderDocItemCategoryEphemeralResources TerraformProviderDocItemCategory = "ephemeral_resources"
	TerraformProviderDocItemCategoryFunctions          TerraformProviderDocItemCategory = "functions"
	TerraformProviderDocItemCategoryGuides             TerraformProviderDocItemCategory = "guides"
)

// IsValid returns true if the category is one of the known documentation categories.
func (c TerraformProviderDocItemCategory) IsValid() bool {
	switch c {
	case TerraformProviderDocItemCategoryOverview,
		TerraformProviderDocItemCategoryResources,
		TerraformProviderDocItemCategoryDataSources,
		TerraformProviderDocItemCategoryEphemeralResources,
		TerraformProviderDocItemCategoryFunctions,
		TerraformProviderDocItemCategoryGuides:
		return true
	default:
		return false
	}
}

// TerraformProviderDocItem represents a single documentation file belonging to a provider version.
// The file's Markdown content is stored in object storage; this model holds only its metadata.
type TerraformProviderDocItem struct {
	ProviderVersionID string
	Filename          string
	Category          TerraformProviderDocItemCategory
	Subcategory       *string
	Metadata          ResourceMetadata
}

// GetID returns the Metadata ID.
func (d *TerraformProviderDocItem) GetID() string {
	return d.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (d *TerraformProviderDocItem) GetGlobalID() string {
	return gid.ToGlobalID(d.GetModelType(), d.Metadata.ID)
}

// GetModelType returns the Model's type
func (d *TerraformProviderDocItem) GetModelType() types.ModelType {
	return types.TerraformProviderDocItemModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (d *TerraformProviderDocItem) ResolveMetadata(key string) (*string, error) {
	return d.Metadata.resolveFieldValue(key)
}

// Validate validates the resource.
func (d *TerraformProviderDocItem) Validate() error {
	if !d.Category.IsValid() {
		return errors.New("invalid documentation category %q", d.Category, errors.WithErrorCode(errors.EInvalid))
	}

	if d.Filename == "" {
		return errors.New("documentation filename is required", errors.WithErrorCode(errors.EInvalid))
	}

	if len(d.Filename) > maxTerraformProviderDocItemFilenameLength {
		return errors.New(
			"documentation filename cannot be greater than %d characters", maxTerraformProviderDocItemFilenameLength,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	if !docItemFilenameRegex.MatchString(d.Filename) {
		return errors.New(
			"documentation filename %q may only contain the characters A-Z, a-z, 0-9, '.', '_', and '-' and cannot start with '.'", d.Filename,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	if !strings.HasSuffix(d.Filename, docItemFilenameExtension) {
		return errors.New(
			"documentation filename %q must have a %s extension", d.Filename, docItemFilenameExtension,
			errors.WithErrorCode(errors.EInvalid),
		)
	}

	return nil
}
