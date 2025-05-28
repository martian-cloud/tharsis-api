// Package models package
package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*ActivityEvent)(nil)

// ActivityEventAction represents the type of action.
type ActivityEventAction string

// ActivityEventAction Types
const (
	ActionAdd                 ActivityEventAction = "ADD"
	ActionAddMember           ActivityEventAction = "ADD_MEMBER"
	ActionCreateMembership    ActivityEventAction = "CREATE_MEMBERSHIP"
	ActionApply               ActivityEventAction = "APPLY"
	ActionCancel              ActivityEventAction = "CANCEL"
	ActionCreate              ActivityEventAction = "CREATE"
	ActionDeleteChildResource ActivityEventAction = "DELETE_CHILD_RESOURCE"
	ActionLock                ActivityEventAction = "LOCK"
	ActionMigrate             ActivityEventAction = "MIGRATE"
	ActionRemove              ActivityEventAction = "REMOVE"
	ActionRemoveMember        ActivityEventAction = "REMOVE_MEMBER"
	ActionRemoveMembership    ActivityEventAction = "REMOVE_MEMBERSHIP"
	ActionSetVariables        ActivityEventAction = "SET_VARIABLES"
	ActionUnlock              ActivityEventAction = "UNLOCK"
	ActionUpdate              ActivityEventAction = "UPDATE"
	ActionUpdateMember        ActivityEventAction = "UPDATE_MEMBER"
)

// ActivityEventTargetType represents the type of the target of the action.
type ActivityEventTargetType string

// ActivityEventTargetType Values
const (
	TargetGPGKey                         ActivityEventTargetType = "GPG_KEY"
	TargetGroup                          ActivityEventTargetType = "GROUP"
	TargetManagedIdentity                ActivityEventTargetType = "MANAGED_IDENTITY"
	TargetManagedIdentityAccessRule      ActivityEventTargetType = "MANAGED_IDENTITY_ACCESS_RULE"
	TargetNamespaceMembership            ActivityEventTargetType = "NAMESPACE_MEMBERSHIP"
	TargetRun                            ActivityEventTargetType = "RUN"
	TargetRunner                         ActivityEventTargetType = "RUNNER"
	TargetServiceAccount                 ActivityEventTargetType = "SERVICE_ACCOUNT"
	TargetStateVersion                   ActivityEventTargetType = "STATE_VERSION"
	TargetTeam                           ActivityEventTargetType = "TEAM"
	TargetTerraformProvider              ActivityEventTargetType = "TERRAFORM_PROVIDER"
	TargetTerraformProviderVersion       ActivityEventTargetType = "TERRAFORM_PROVIDER_VERSION"
	TargetTerraformProviderVersionMirror ActivityEventTargetType = "TERRAFORM_PROVIDER_VERSION_MIRROR"
	TargetTerraformModule                ActivityEventTargetType = "TERRAFORM_MODULE"
	TargetTerraformModuleVersion         ActivityEventTargetType = "TERRAFORM_MODULE_VERSION"
	TargetVariable                       ActivityEventTargetType = "VARIABLE"
	TargetVCSProvider                    ActivityEventTargetType = "VCS_PROVIDER"
	TargetWorkspace                      ActivityEventTargetType = "WORKSPACE"
	TargetRole                           ActivityEventTargetType = "ROLE"
	TargetFederatedRegistry              ActivityEventTargetType = "FEDERATED_REGISTRY"
)

// ActivityEventCreateNamespaceMembershipPayload helps with custom
// payloads for activity events for namespace memberships.
type ActivityEventCreateNamespaceMembershipPayload struct {
	UserID           *string `json:"userId"`
	ServiceAccountID *string `json:"serviceAccountId"`
	TeamID           *string `json:"teamId"`
	Role             string  `json:"role"`
}

// ActivityEventUpdateNamespaceMembershipPayload helps with custom
// payloads for activity events for namespace memberships.
type ActivityEventUpdateNamespaceMembershipPayload struct {
	PrevRole string `json:"prevRole"`
	NewRole  string `json:"newRole"`
}

// ActivityEventRemoveNamespaceMembershipPayload helps with custom
// payloads for activity events for namespace memberships.
type ActivityEventRemoveNamespaceMembershipPayload struct {
	UserID           *string `json:"userId"`
	ServiceAccountID *string `json:"serviceAccountId"`
	TeamID           *string `json:"teamId"`
}

// ActivityEventDeleteChildResourcePayload holds information about the resource that was deleted.
// The target ID and target type of the associated activity event will be that of the group
// (or workspace for a workspace-level variable) the resource was in.
type ActivityEventDeleteChildResourcePayload struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ActivityEventAddTeamMemberPayload is the custom payload for adding a user to a team
type ActivityEventAddTeamMemberPayload struct {
	UserID     *string `json:"userId"`
	Maintainer bool    `json:"maintainer"`
}

// ActivityEventRemoveTeamMemberPayload is the custom payload for removing a user from a team
type ActivityEventRemoveTeamMemberPayload struct {
	UserID *string `json:"userId"`
}

// ActivityEventUpdateTeamMemberPayload is the custom payload for updating a member of a team
type ActivityEventUpdateTeamMemberPayload struct {
	UserID     *string `json:"userId"`
	Maintainer bool    `json:"maintainer"`
}

// ActivityEventMigrateGroupPayload is the custom payload for migrating a group.
type ActivityEventMigrateGroupPayload struct {
	PreviousGroupPath string `json:"previousGroupPath"`
}

// ActivityEventMigrateWorkspacePayload is the custom payload for migrating a workspace.
type ActivityEventMigrateWorkspacePayload struct {
	PreviousGroupPath string `json:"previousGroupPath"`
}

// ActivityEventMoveManagedIdentityPayload is the custom payload for moving a managed identity to another group.
type ActivityEventMoveManagedIdentityPayload struct {
	PreviousGroupPath string `json:"previousGroupPath"`
}

// ActivityEvent resource
type ActivityEvent struct {
	UserID           *string
	ServiceAccountID *string
	NamespacePath    *string
	Payload          []byte
	TargetID         string
	Action           ActivityEventAction
	TargetType       ActivityEventTargetType
	Metadata         ResourceMetadata
}

// GetID returns the Metadata ID.
func (a *ActivityEvent) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *ActivityEvent) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the Model's type
func (a *ActivityEvent) GetModelType() types.ModelType {
	return types.ActivityEventModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (a *ActivityEvent) ResolveMetadata(key string) (*string, error) {
	val, err := a.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "user_id":
			return a.UserID, nil
		case "service_account_id":
			return a.ServiceAccountID, nil
		case "namespace_path":
			return a.NamespacePath, nil
		case "action":
			action := string(a.Action)
			return &action, nil
		case "target_type":
			targetType := string(a.TargetType)
			return &targetType, nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate validates the resource.
func (a *ActivityEvent) Validate() error {
	return nil
}
