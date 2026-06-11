package builder

import "encoding/json"

// MembershipChangeAction represents the type of namespace membership change
type MembershipChangeAction string

// MembershipChangeAction constants
const (
	MembershipChangeActionCreated     MembershipChangeAction = "created"
	MembershipChangeActionRoleChanged MembershipChangeAction = "role_changed"
	MembershipChangeActionRemoved     MembershipChangeAction = "removed"
)

// MembershipChangeEmail is the email builder for namespace membership change notifications.
type MembershipChangeEmail struct {
	Action                  MembershipChangeAction
	NamespacePath           string
	RoleName                string
	PrevRoleName            string // non-empty only for role updates
	PerformedBy             string
	TeamName                string // non-empty when membership was granted via a team
	ServiceAccountPath      string // non-empty when notifying namespace members about a service account membership change
	ServiceAccountID        string // global ID of the service account, non-empty when ServiceAccountPath is set
	ServiceAccountGroupPath string // parent group path of the service account, non-empty when ServiceAccountPath is set
	IsCustomRole            bool   // true when the current/new role is not a built-in role
	IsPrevCustomRole        bool   // true when the previous role is not a built-in role (role updates only)
	IsWorkspace             bool   // true when the namespace is a workspace rather than a group
}

// ShowCTA returns true when a "View Group/Workspace" button should be rendered in the email.
func (e *MembershipChangeEmail) ShowCTA() bool {
	// Owners notified of SA membership changes always retain access to the namespace.
	if e.ServiceAccountPath != "" {
		return true
	}

	return e.Action != MembershipChangeActionRemoved
}

func (e *MembershipChangeEmail) namespaceType() string {
	if e.IsWorkspace {
		return "Workspace"
	}

	return "Group"
}

// Subject returns the email subject line matching the action and namespace type.
func (e *MembershipChangeEmail) Subject() string {
	ns := e.namespaceType()

	if e.ServiceAccountPath != "" {
		switch e.Action {
		case MembershipChangeActionCreated:
			return "Service account added to " + ns
		case MembershipChangeActionRoleChanged:
			return "Service account role updated in " + ns
		default:
			return "Service account removed from " + ns
		}
	}

	switch e.Action {
	case MembershipChangeActionCreated:
		return ns + " access granted"
	case MembershipChangeActionRoleChanged:
		return ns + " role updated"
	default:
		return ns + " access revoked"
	}
}

// Type returns the type of email builder
func (e *MembershipChangeEmail) Type() EmailType {
	return MembershipChangeEmailType
}

// Build returns the email html
func (e *MembershipChangeEmail) Build(templateCtx *TemplateContext) (string, error) {
	html, err := templateCtx.ExecuteTemplate(e.Type().TemplateFilename(), e)
	if err != nil {
		return "", err
	}

	return templateCtx.WrapInBaseTemplate(html)
}

// InitFromData creates the builder from raw data
func (e *MembershipChangeEmail) InitFromData(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, e)
}
