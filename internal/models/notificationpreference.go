package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*NotificationPreference)(nil)

// NotificationPreferenceScope represents the scope for determining which notifications will be sent
type NotificationPreferenceScope string

// NotificationPreferenceScope Types
const (
	// NotificationPreferenceScopeAll is used to get all notifications
	NotificationPreferenceScopeAll NotificationPreferenceScope = "ALL"
	// NotificationPreferenceScopeParticipate is used to get notifications only for events the user participates in
	NotificationPreferenceScopeParticipate NotificationPreferenceScope = "PARTICIPATE"
	// NotificationPreferenceScopeCustom is used to get notifications for a custom list of events
	NotificationPreferenceScopeCustom NotificationPreferenceScope = "CUSTOM"
	// NotificationPreferenceScopeNone is used to not get any notifications
	NotificationPreferenceScopeNone NotificationPreferenceScope = "NONE"
)

// Valid checks if the NotificationPreferenceScope is valid
func (s NotificationPreferenceScope) Valid() bool {
	switch s {
	case NotificationPreferenceScopeAll,
		NotificationPreferenceScopeParticipate,
		NotificationPreferenceScopeCustom,
		NotificationPreferenceScopeNone:
		return true
	default:
		return false
	}
}

// NotificationPreferenceCustomEvents represents the custom events for notification preferences
type NotificationPreferenceCustomEvents struct {
	FailedRun bool `json:"failed_run"`
}

// NotificationPreference is used to control which notifications are sent to a user
type NotificationPreference struct {
	Scope         NotificationPreferenceScope
	NamespacePath *string
	CustomEvents  *NotificationPreferenceCustomEvents
	UserID        string
	Metadata      ResourceMetadata
}

// GetID returns the Metadata ID.
func (n *NotificationPreference) GetID() string {
	return n.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (n *NotificationPreference) GetGlobalID() string {
	return gid.ToGlobalID(n.GetModelType(), n.Metadata.ID)
}

// GetModelType returns the type of the model.
func (n *NotificationPreference) GetModelType() types.ModelType {
	return types.NotificationPreferenceModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (n *NotificationPreference) ResolveMetadata(key string) (string, error) {
	return n.Metadata.resolveFieldValue(key)
}

// IsGlobal returns true if the notification preference is global.
func (n *NotificationPreference) IsGlobal() bool {
	return n.NamespacePath == nil
}

// Validate returns an error if the model is not valid
func (n *NotificationPreference) Validate() error {
	// Verify scope satisfies constraints
	if !n.Scope.Valid() {
		return errors.New("scope is invalid", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify custom events are set if scope is custom
	if n.Scope == NotificationPreferenceScopeCustom && n.CustomEvents == nil {
		return errors.New("custom events must be set if scope is custom", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify custom events is nil if scope is not custom
	if n.Scope != NotificationPreferenceScopeCustom && n.CustomEvents != nil {
		return errors.New("custom events must be nil if scope is not custom", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify namespace path is not empty
	if n.NamespacePath != nil && *n.NamespacePath == "" {
		return errors.New("namespace path cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
