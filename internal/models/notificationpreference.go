package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// NotificationPreferenceScope represents the scope for determining which notifications will be sent
type NotificationPreferenceScope string

// NotificationPreferenceScope Types
const (
	// NotificationPreferenceScopeAll is used to get all notifications
	NotificationPreferenceScopeAll         NotificationPreferenceScope = "ALL"
	// NotificationPreferenceScopeParticipate is used to get notifications only for events the user participates in
	NotificationPreferenceScopeParticipate NotificationPreferenceScope = "PARTICIPATE"
	// NotificationPreferenceScopeCustom is used to get notifications for a custom list of events
	NotificationPreferenceScopeCustom      NotificationPreferenceScope = "CUSTOM"
	// NotificationPreferenceScopeNone is used to not get any notifications
	NotificationPreferenceScopeNone        NotificationPreferenceScope = "NONE"
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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (m *NotificationPreference) ResolveMetadata(key string) (string, error) {
	return m.Metadata.resolveFieldValue(key)
}

// IsGlobal returns true if the notification preference is global.
func (m *NotificationPreference) IsGlobal() bool {
	return m.NamespacePath == nil
}

// Validate returns an error if the model is not valid
func (m *NotificationPreference) Validate() error {
	// Verify scope satisfies constraints
	if !m.Scope.Valid() {
		return errors.New("scope is invalid", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify custom events are set if scope is custom
	if m.Scope == NotificationPreferenceScopeCustom && m.CustomEvents == nil {
		return errors.New("custom events must be set if scope is custom", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify custom events is nil if scope is not custom
	if m.Scope != NotificationPreferenceScopeCustom && m.CustomEvents != nil {
		return errors.New("custom events must be nil if scope is not custom", errors.WithErrorCode(errors.EInvalid))
	}
	// Verify namespace path is not empty
	if m.NamespacePath != nil && *m.NamespacePath == "" {
		return errors.New("namespace path cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
