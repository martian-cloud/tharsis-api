package models

import (
	"encoding/hex"
	"time"

	"github.com/hashicorp/go-version"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*PackageVersion)(nil)

// PackageVersionStatus is the status of the policy set version upload
type PackageVersionStatus string

// PackageVersionStatus constants
const (
	PackageVersionStatusPending          PackageVersionStatus = "pending"
	PackageVersionStatusUploadInProgress PackageVersionStatus = "upload_in_progress"
	PackageVersionStatusErrored          PackageVersionStatus = "errored"
	PackageVersionStatusUploaded         PackageVersionStatus = "uploaded"
)

// PackageVersion represents a version of a policy set
type PackageVersion struct {
	CreatedBy              string
	PackageID              string
	SemanticVersion        string
	Status                 PackageVersionStatus
	Error                  *string
	ObjectStoreKey         *string
	UploadStartedTimestamp *time.Time
	Metadata               ResourceMetadata
	SHASum                 []byte
	// Size is the byte size of the uploaded package. It is 0 until the upload completes, and also for
	// versions uploaded before the field existed, so callers must treat 0 as unknown.
	Size   int
	Latest bool
}

// GetID returns the Metadata ID.
func (p *PackageVersion) GetID() string {
	return p.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (p *PackageVersion) GetGlobalID() string {
	return gid.ToGlobalID(p.GetModelType(), p.Metadata.ID)
}

// GetModelType returns the type of the model.
func (p *PackageVersion) GetModelType() types.ModelType {
	return types.PackageVersionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (p *PackageVersion) ResolveMetadata(key string) (*string, error) {
	return p.Metadata.resolveFieldValue(key)
}

// Validate validates the model: a version is always owned by a package, carries a parseable
// semantic version, and has a recognized status.
func (p *PackageVersion) Validate() error {
	if p.PackageID == "" {
		return errors.New("package version must belong to a package", errors.WithErrorCode(errors.EInvalid))
	}

	if p.SemanticVersion == "" {
		return errors.New("package version semantic version is required", errors.WithErrorCode(errors.EInvalid))
	}
	if _, err := version.NewSemver(p.SemanticVersion); err != nil {
		return errors.New("package version %q is not a valid semantic version", p.SemanticVersion,
			errors.WithErrorCode(errors.EInvalid))
	}

	switch p.Status {
	case PackageVersionStatusPending, PackageVersionStatusUploadInProgress,
		PackageVersionStatusErrored, PackageVersionStatusUploaded:
	default:
		return errors.New("package version status %s is not supported", p.Status, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// GetSHASumHex returns the SHA checksum as a HEX string
func (p *PackageVersion) GetSHASumHex() string {
	return hex.EncodeToString(p.SHASum)
}
