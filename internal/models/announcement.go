package models

import (
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	maxAnnouncementMessageLength = 500
)

var _ Model = (*Announcement)(nil)

// AnnouncementType represents the type/severity level of an announcement
type AnnouncementType string

const (
	// AnnouncementTypeInfo represents informational announcements
	AnnouncementTypeInfo AnnouncementType = "INFO"
	// AnnouncementTypeError represents error/critical announcements
	AnnouncementTypeError AnnouncementType = "ERROR"
	// AnnouncementTypeWarning represents warning announcements
	AnnouncementTypeWarning AnnouncementType = "WARNING"
	// AnnouncementTypeSuccess represents success announcements
	AnnouncementTypeSuccess AnnouncementType = "SUCCESS"
)

// Announcement represents a scheduled announcement message
type Announcement struct {
	Metadata    ResourceMetadata
	Message     string
	StartTime   time.Time
	EndTime     *time.Time
	CreatedBy   string
	Type        AnnouncementType
	Dismissible bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *Announcement) ResolveMetadata(key string) (*string, error) {
	val, err := a.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "start_time":
			return ptr.String(a.StartTime.String()), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// GetID returns the Metadata ID.
func (a *Announcement) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *Announcement) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the type of the model.
func (a *Announcement) GetModelType() types.ModelType {
	return types.AnnouncementModelType
}

// IsActive returns true if the announcement is currently active (within its time range)
func (a *Announcement) IsActive() bool {
	now := time.Now().UTC()
	isAfterOrEqualStart := !now.Before(a.StartTime)

	if a.EndTime == nil {
		return isAfterOrEqualStart
	}

	return isAfterOrEqualStart && !now.After(*a.EndTime)
}

// IsExpired returns true if the announcement has expired (past its end time)
func (a *Announcement) IsExpired() bool {
	if a.EndTime == nil {
		return false
	}

	return time.Now().UTC().After(*a.EndTime)
}

// Validate validates the announcement fields
func (a *Announcement) Validate() error {
	if len(a.Message) == 0 {
		return errors.New("message is required", errors.WithErrorCode(errors.EInvalid))
	}

	if len(a.Message) > maxAnnouncementMessageLength {
		return errors.New("message cannot be greater than %d characters", maxAnnouncementMessageLength, errors.WithErrorCode(errors.EInvalid))
	}

	if a.StartTime.IsZero() {
		return errors.New("start time is required", errors.WithErrorCode(errors.EInvalid))
	}

	if a.EndTime != nil && a.StartTime.After(*a.EndTime) {
		return errors.New("start time must be before end time", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
