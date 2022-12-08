package models

import (
	"encoding/hex"
	"time"
)

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

// GetSHASumHex returns the SHA checksum as a HEX string
func (t *TerraformModuleVersion) GetSHASumHex() string {
	return hex.EncodeToString(t.SHASum)
}
