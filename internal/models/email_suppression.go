package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*EmailSuppression)(nil)

// EmailSuppressionCause represents the event that added an address to the suppression list.
type EmailSuppressionCause string

// EmailSuppressionCause constants.
const (
	// EmailSuppressionCauseHardBounce means the address hard-bounced.
	EmailSuppressionCauseHardBounce EmailSuppressionCause = "hard_bounce"
	// EmailSuppressionCauseComplaint means the recipient reported the message as spam.
	EmailSuppressionCauseComplaint EmailSuppressionCause = "complaint"
)

// EmailSuppression represents a globally suppressed email address. Checked at
// enqueue time so suppressed addresses never get an email_recipients row.
type EmailSuppression struct {
	Metadata ResourceMetadata
	Address  string
	Cause    EmailSuppressionCause
}

// GetID returns the Metadata ID.
func (e *EmailSuppression) GetID() string {
	return e.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (e *EmailSuppression) GetGlobalID() string {
	return gid.ToGlobalID(e.GetModelType(), e.Metadata.ID)
}

// GetModelType returns the type of the model.
func (e *EmailSuppression) GetModelType() types.ModelType {
	return types.EmailSuppressionModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (e *EmailSuppression) ResolveMetadata(key string) (*string, error) {
	return e.Metadata.resolveFieldValue(key)
}

// Validate validates the email suppression fields.
func (e *EmailSuppression) Validate() error {
	return nil
}
