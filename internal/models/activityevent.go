package models

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
	TargetGPGKey                    ActivityEventTargetType = "GPG_KEY"
	TargetGroup                     ActivityEventTargetType = "GROUP"
	TargetManagedIdentity           ActivityEventTargetType = "MANAGED_IDENTITY"
	TargetManagedIdentityAccessRule ActivityEventTargetType = "MANAGED_IDENTITY_ACCESS_RULE"
	TargetNamespaceMembership       ActivityEventTargetType = "NAMESPACE_MEMBERSHIP"
	TargetRun                       ActivityEventTargetType = "RUN"
	TargetServiceAccount            ActivityEventTargetType = "SERVICE_ACCOUNT"
	TargetStateVersion              ActivityEventTargetType = "STATE_VERSION"
	TargetTeam                      ActivityEventTargetType = "TEAM"
	TargetTerraformProvider         ActivityEventTargetType = "TERRAFORM_PROVIDER"
	TargetTerraformProviderVersion  ActivityEventTargetType = "TERRAFORM_PROVIDER_VERSION"
	TargetVariable                  ActivityEventTargetType = "VARIABLE"
	TargetWorkspace                 ActivityEventTargetType = "WORKSPACE"
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

// The End.
