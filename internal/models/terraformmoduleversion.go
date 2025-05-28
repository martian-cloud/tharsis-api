package models

import (
	"encoding/hex"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*TerraformModuleVersion)(nil)

// TerraformModuleVersionStatus is the status of the module version upload
type TerraformModuleVersionStatus string

// TerraformModuleVersionStatus constants
const (
	TerraformModuleVersionStatusPending          TerraformModuleVersionStatus = "pending"
	TerraformModuleVersionStatusUploadInProgress TerraformModuleVersionStatus = "upload_in_progress"
	TerraformModuleVersionStatusErrored          TerraformModuleVersionStatus = "errored"
	TerraformModuleVersionStatusUploaded         TerraformModuleVersionStatus = "uploaded"
)

// TerraformModuleVersion represents a terraform module version
type TerraformModuleVersion struct {
	CreatedBy              string
	ModuleID               string
	SemanticVersion        string
	Status                 TerraformModuleVersionStatus
	Error                  string
	Diagnostics            string
	UploadStartedTimestamp *time.Time
	Metadata               ResourceMetadata
	SHASum                 []byte
	Submodules             []string
	Examples               []string
	Latest                 bool
}

// GetID returns the Metadata ID.
func (t *TerraformModuleVersion) GetID() string {
	return t.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (t *TerraformModuleVersion) GetGlobalID() string {
	return gid.ToGlobalID(t.GetModelType(), t.Metadata.ID)
}

// GetModelType returns the type of the model.
func (t *TerraformModuleVersion) GetModelType() types.ModelType {
	return types.TerraformModuleVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (t *TerraformModuleVersion) ResolveMetadata(key string) (*string, error) {
	return t.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (t *TerraformModuleVersion) Validate() error {
	return nil
}

// GetSHASumHex returns the SHA checksum as a HEX string
func (t *TerraformModuleVersion) GetSHASumHex() string {
	return hex.EncodeToString(t.SHASum)
}
