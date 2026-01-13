package resolver

import (
	"context"
)

// RootResolver is the entry point for all top-level operations.
type RootResolver struct{}

// NewRootResolver creates a root resolver
func NewRootResolver() *RootResolver {
	return &RootResolver{}
}

/* Me query to get current authenticated subject */

// Me query returns the authenticated subject
func (r RootResolver) Me(ctx context.Context) (*MeResponseResolver, error) {
	return meQuery(ctx)
}

/* User Preferences Queries and Mutations */

// UserPreferences query returns the user preferences for the authenticated subject
func (r RootResolver) UserPreferences() (*UserPreferencesResolver, error) {
	return userPreferencesQuery()
}

// SetUserNotificationPreference sets the notification preference for the authenticated subject
func (r RootResolver) SetUserNotificationPreference(ctx context.Context, args *struct {
	Input *SetUserNotificationPreferenceInput
},
) (*UserNotificationPreferenceMutationPayloadResolver, error) {
	response, err := setUserNotificationPreferenceMutation(ctx, args.Input)
	if err != nil {
		return handleUserNotificationPreferenceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Node query */

// Node query returns a node by ID
func (r RootResolver) Node(ctx context.Context, args *struct{ ID string }) (*NodeResolver, error) {
	return node(ctx, args.ID)
}

/* Namespace Queries and Mutations */

// Namespace query returns a namespace by full path
func (r RootResolver) Namespace(ctx context.Context, args *NamespaceQueryArgs) (*NamespaceResolver, error) {
	return namespaceQuery(ctx, args)
}

/* Workspace Queries and Mutations */

// Workspace query returns a workspace by full path
func (r RootResolver) Workspace(ctx context.Context, args *WorkspaceQueryArgs) (*WorkspaceResolver, error) {
	return workspaceQuery(ctx, args)
}

// Workspaces query returns a workspace connection
func (r RootResolver) Workspaces(ctx context.Context, args *WorkspaceConnectionQueryArgs) (*WorkspaceConnectionResolver, error) {
	return workspacesQuery(ctx, args)
}

// CreateWorkspace creates a new workspace
func (r RootResolver) CreateWorkspace(ctx context.Context, args *struct{ Input *CreateWorkspaceInput }) (*WorkspaceMutationPayloadResolver, error) {
	response, err := createWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateWorkspace updates an existing workspace
func (r RootResolver) UpdateWorkspace(ctx context.Context, args *struct{ Input *UpdateWorkspaceInput }) (*WorkspaceMutationPayloadResolver, error) {
	response, err := updateWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteWorkspace deletes a workspace
func (r RootResolver) DeleteWorkspace(ctx context.Context, args *struct{ Input *DeleteWorkspaceInput }) (*WorkspaceMutationPayloadResolver, error) {
	response, err := deleteWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// WorkspaceEvents subscribes to events for a particular workspace
func (r RootResolver) WorkspaceEvents(ctx context.Context, args *struct{ Input *WorkspaceSubscriptionInput }) (<-chan *WorkspaceEventResolver, error) {
	return r.workspaceEventsSubscription(ctx, args.Input)
}

// LockWorkspace mutation locks a workspace
func (r RootResolver) LockWorkspace(ctx context.Context, args *struct{ Input *LockWorkspaceInput }) (*WorkspaceMutationPayloadResolver, error) {
	response, err := lockWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UnlockWorkspace mutation unlocks a workspace
func (r RootResolver) UnlockWorkspace(ctx context.Context, args *struct{ Input *UnlockWorkspaceInput }) (*WorkspaceMutationPayloadResolver, error) {
	response, err := unlockWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// MigrateWorkspace migrates an existing workspace
func (r RootResolver) MigrateWorkspace(ctx context.Context,
	args *struct{ Input *MigrateWorkspaceInput },
) (*WorkspaceMutationPayloadResolver, error) {
	response, err := migrateWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DestroyWorkspace creates a destroy run for an existing workspace
func (r RootResolver) DestroyWorkspace(ctx context.Context,
	args *struct{ Input *DestroyWorkspaceInput },
) (*RunMutationPayloadResolver, error) {
	response, err := destroyWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// AssessWorkspace creates an assessment run for an existing workspace
func (r RootResolver) AssessWorkspace(ctx context.Context,
	args *struct{ Input *AssessWorkspaceInput },
) (*RunMutationPayloadResolver, error) {
	response, err := assessWorkspaceMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* State Version Queries and Mutations */

// CreateStateVersion creates a new state version
func (r RootResolver) CreateStateVersion(ctx context.Context, args *struct{ Input *CreateStateVersionInput }) (*StateVersionMutationPayloadResolver, error) {
	response, err := createStateVersionMutation(ctx, args.Input)
	if err != nil {
		return handleStateVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Groups Queries and Mutations */

// Group query returns a group by full path
func (r RootResolver) Group(ctx context.Context, args *GroupQueryArgs) (*GroupResolver, error) {
	return groupQuery(ctx, args)
}

// Groups query returns a groups connection
func (r RootResolver) Groups(ctx context.Context, args *GroupConnectionQueryArgs) (*GroupConnectionResolver, error) {
	return groupsQuery(ctx, args)
}

// CreateGroup creates a new group
func (r RootResolver) CreateGroup(ctx context.Context, args *struct{ Input *CreateGroupInput }) (*GroupMutationPayloadResolver, error) {
	response, err := createGroupMutation(ctx, args.Input)
	if err != nil {
		return handleGroupMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateGroup updates an existing group
func (r RootResolver) UpdateGroup(ctx context.Context, args *struct{ Input *UpdateGroupInput }) (*GroupMutationPayloadResolver, error) {
	response, err := updateGroupMutation(ctx, args.Input)
	if err != nil {
		return handleGroupMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteGroup deletes a group
func (r RootResolver) DeleteGroup(ctx context.Context, args *struct{ Input *DeleteGroupInput }) (*GroupMutationPayloadResolver, error) {
	response, err := deleteGroupMutation(ctx, args.Input)
	if err != nil {
		return handleGroupMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// MigrateGroup migrates an existing group
func (r RootResolver) MigrateGroup(ctx context.Context,
	args *struct{ Input *MigrateGroupInput },
) (*GroupMutationPayloadResolver, error) {
	response, err := migrateGroupMutation(ctx, args.Input)
	if err != nil {
		return handleGroupMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Run Queries and Mutations */

// Run query returns a run by ID
func (r RootResolver) Run(ctx context.Context, args *RunQueryArgs) (*RunResolver, error) {
	return runQuery(ctx, args)
}

// Runs query returns a run connection
func (r RootResolver) Runs(ctx context.Context, args *RunConnectionQueryArgs) (*RunConnectionResolver, error) {
	return runsQuery(ctx, args)
}

// CreateRun mutation creates a new run
func (r RootResolver) CreateRun(ctx context.Context, args *struct{ Input *CreateRunInput }) (*RunMutationPayloadResolver, error) {
	response, err := createRunMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// ApplyRun mutation starts the apply stage for a run
func (r RootResolver) ApplyRun(ctx context.Context, args *struct{ Input *ApplyRunInput }) (*RunMutationPayloadResolver, error) {
	response, err := applyRunMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CancelRun mutation cancels a run
func (r RootResolver) CancelRun(ctx context.Context, args *struct{ Input *CancelRunInput }) (*RunMutationPayloadResolver, error) {
	response, err := cancelRunMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Plan Queries and Mutations */

// UpdatePlan updates an existing plan
func (r RootResolver) UpdatePlan(ctx context.Context, args *struct{ Input *UpdatePlanInput }) (*PlanMutationPayloadResolver, error) {
	response, err := updatePlanMutation(ctx, args.Input)
	if err != nil {
		return handlePlanMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Apply Queries and Mutations */

// UpdateApply updates an existing apply
func (r RootResolver) UpdateApply(ctx context.Context, args *struct{ Input *UpdateApplyInput }) (*ApplyMutationPayloadResolver, error) {
	response, err := updateApplyMutation(ctx, args.Input)
	if err != nil {
		return handleApplyMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// SetVariablesIncludedInTFConfig sets the variables that are included in the Terraform config.
func (r RootResolver) SetVariablesIncludedInTFConfig(ctx context.Context, args *struct {
	Input *SetVariablesIncludedInTFConfigInput
},
) (*RunMutationPayloadResolver, error) {
	response, err := setVariablesIncludedInTFConfigMutation(ctx, args.Input)
	if err != nil {
		return handleRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Managed Identity / Credentials Queries and Mutations */

// ManagedIdentity query returns a managed identity
func (r RootResolver) ManagedIdentity(ctx context.Context, args *ManagedIdentityQueryArgs) (*ManagedIdentityResolver, error) {
	return managedIdentityQuery(ctx, args)
}

// CreateManagedIdentityAccessRule creates a new managed identity access rule
func (r RootResolver) CreateManagedIdentityAccessRule(ctx context.Context, args *struct {
	Input *CreateManagedIdentityAccessRuleInput
},
) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	response, err := createManagedIdentityAccessRuleMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityAccessRuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateManagedIdentityAccessRule updates an existing managed identity access rule
func (r RootResolver) UpdateManagedIdentityAccessRule(ctx context.Context, args *struct {
	Input *UpdateManagedIdentityAccessRuleInput
},
) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	response, err := updateManagedIdentityAccessRuleMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityAccessRuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteManagedIdentityAccessRule deletes an existing managed identity access rule
func (r RootResolver) DeleteManagedIdentityAccessRule(ctx context.Context, args *struct {
	Input *DeleteManagedIdentityAccessRuleInput
},
) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	response, err := deleteManagedIdentityAccessRuleMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityAccessRuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateManagedIdentityAlias creates a managed identity alias
func (r RootResolver) CreateManagedIdentityAlias(ctx context.Context, args *struct {
	Input *CreateManagedIdentityAliasInput
},
) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := createManagedIdentityAliasMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteManagedIdentityAlias deletes a managed identity alias
func (r RootResolver) DeleteManagedIdentityAlias(ctx context.Context, args *struct{ Input *DeleteManagedIdentityInput }) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := deleteManagedIdentityAliasMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateManagedIdentity creates a new managed identity
func (r RootResolver) CreateManagedIdentity(ctx context.Context, args *struct{ Input *CreateManagedIdentityInput }) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := createManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateManagedIdentity updates an existing managed identity
func (r RootResolver) UpdateManagedIdentity(ctx context.Context, args *struct{ Input *UpdateManagedIdentityInput }) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := updateManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteManagedIdentity deletes a managed identity
func (r RootResolver) DeleteManagedIdentity(ctx context.Context, args *struct{ Input *DeleteManagedIdentityInput }) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := deleteManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// AssignManagedIdentity assigns a managed identity to a workspace
func (r RootResolver) AssignManagedIdentity(ctx context.Context, args *struct{ Input *AssignManagedIdentityInput }) (*AssignManagedIdentityMutationPayloadResolver, error) {
	response, err := assignManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleAssignManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UnassignManagedIdentity un-assigns a managed identity from a workspace
func (r RootResolver) UnassignManagedIdentity(ctx context.Context, args *struct{ Input *AssignManagedIdentityInput }) (*AssignManagedIdentityMutationPayloadResolver, error) {
	response, err := unassignManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleAssignManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateManagedIdentityCredentials creates credentials for a managed identity
func (r RootResolver) CreateManagedIdentityCredentials(ctx context.Context, args *struct {
	Input *CreateManagedIdentityCredentialsInput
},
) (*ManagedIdentityCredentialsMutationPayloadResolver, error) {
	response, err := createManagedIdentityCredentialsMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityCredentialsMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// MoveManagedIdentity moves a managed identity to another group
func (r RootResolver) MoveManagedIdentity(ctx context.Context, args *struct {
	Input *MoveManagedIdentityInput
},
) (*ManagedIdentityMutationPayloadResolver, error) {
	response, err := moveManagedIdentityMutation(ctx, args.Input)
	if err != nil {
		return handleManagedIdentityMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Service Account Queries and Mutations */

// ServiceAccount query returns a service account
func (r RootResolver) ServiceAccount(ctx context.Context, args *ServiceAccountQueryArgs) (*ServiceAccountResolver, error) {
	return serviceAccountQuery(ctx, args)
}

// CreateServiceAccount creates a new service account
func (r RootResolver) CreateServiceAccount(ctx context.Context, args *struct{ Input *CreateServiceAccountInput }) (*ServiceAccountMutationPayloadResolver, error) {
	response, err := createServiceAccountMutation(ctx, args.Input)
	if err != nil {
		return handleServiceAccountMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateServiceAccount updates an existing service account
func (r RootResolver) UpdateServiceAccount(ctx context.Context, args *struct{ Input *UpdateServiceAccountInput }) (*ServiceAccountMutationPayloadResolver, error) {
	response, err := updateServiceAccountMutation(ctx, args.Input)
	if err != nil {
		return handleServiceAccountMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteServiceAccount deletes a service account
func (r RootResolver) DeleteServiceAccount(ctx context.Context, args *struct{ Input *DeleteServiceAccountInput }) (*ServiceAccountMutationPayloadResolver, error) {
	response, err := deleteServiceAccountMutation(ctx, args.Input)
	if err != nil {
		return handleServiceAccountMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// ServiceAccountCreateToken creates a token for a service account
func (r RootResolver) ServiceAccountCreateToken(ctx context.Context, args *struct {
	Input *ServiceAccountCreateTokenInput
},
) (*ServiceAccountCreateTokenPayload, error) {
	response, err := serviceAccountCreateTokenMutation(ctx, args.Input)
	if err != nil {
		return handleServiceAccountCreateTokenProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// WorkspaceRunEvents subscribes to run events for a particular workspace
func (r RootResolver) WorkspaceRunEvents(ctx context.Context, args *struct{ Input *RunSubscriptionInput }) (<-chan *RunEventResolver, error) {
	return r.workspaceRunEventsSubscription(ctx, args.Input)
}

/* Namespace Membership queries and Mutations */

// CreateNamespaceMembership creates a new namespace membership in a namespace
func (r RootResolver) CreateNamespaceMembership(ctx context.Context,
	args *struct {
		Input *CreateNamespaceMembershipInput
	},
) (*NamespaceMembershipMutationPayloadResolver, error) {
	response, err := createNamespaceMembershipMutation(ctx, args.Input)
	if err != nil {
		return handleNamespaceMembershipMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateNamespaceMembership updates an existing namespace membership
func (r RootResolver) UpdateNamespaceMembership(ctx context.Context,
	args *struct {
		Input *UpdateNamespaceMembershipInput
	},
) (*NamespaceMembershipMutationPayloadResolver, error) {
	response, err := updateNamespaceMembershipMutation(ctx, args.Input)
	if err != nil {
		return handleNamespaceMembershipMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteNamespaceMembership updates an existing namespace membership
func (r RootResolver) DeleteNamespaceMembership(ctx context.Context,
	args *struct {
		Input *DeleteNamespaceMembershipInput
	},
) (*NamespaceMembershipMutationPayloadResolver, error) {
	response, err := deleteNamespaceMembershipMutation(ctx, args.Input)
	if err != nil {
		return handleNamespaceMembershipMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Job queries and Mutations */

// Job query returns a single job
func (r RootResolver) Job(ctx context.Context, args *JobQueryArgs) (*JobResolver, error) {
	return jobQuery(ctx, args)
}

// Jobs query returns a job connection
func (r RootResolver) Jobs(ctx context.Context, args *JobConnectionQueryArgs) (*JobConnectionResolver, error) {
	return jobsQuery(ctx, args)
}

// JobEvents subscribes to job events
func (r RootResolver) JobEvents(ctx context.Context, args *struct {
	Input *JobEventSubscriptionInput
},
) (<-chan *JobEventResolver, error) {
	return r.jobEventsSubscription(ctx, args.Input)
}

// JobLogStreamEvents sets up a subscription for job log events
func (r RootResolver) JobLogStreamEvents(ctx context.Context,
	args *struct {
		Input *JobLogStreamSubscriptionInput
	},
) (<-chan *JobLogStreamEventResolver, error) {
	return r.jobLogStreamEventsSubscription(ctx, args.Input)
}

// JobCancellationEvent sets up a subscription for job cancellation event
func (r RootResolver) JobCancellationEvent(ctx context.Context, args *struct {
	Input *JobCancellationEventSubscriptionInput
},
) (<-chan *JobCancellationEventResolver, error) {
	return r.jobCancellationEventSubscription(ctx, args.Input)
}

// SaveJobLogs saves job logs
func (r RootResolver) SaveJobLogs(ctx context.Context, args *struct{ Input *SaveJobLogsInput }) (*SaveJobLogsPayload, error) {
	response, err := saveJobLogsMutation(ctx, args.Input)
	if err != nil {
		return handleSaveJobLogsMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// ClaimJob attempts to claim the next available job, it'll block if no jobs are available to be claimed
func (r RootResolver) ClaimJob(ctx context.Context, args *struct{ Input *ClaimJobInput }) (*ClaimJobMutationPayload, error) {
	response, err := claimJobMutation(ctx, args.Input)
	if err != nil {
		return handleClaimJobMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Configuration Version Queries and Mutations */

// ConfigurationVersion query returns a configuration version by ID
func (r RootResolver) ConfigurationVersion(ctx context.Context, args *ConfigurationVersionQueryArgs) (*ConfigurationVersionResolver, error) {
	return configurationVersionQuery(ctx, args)
}

// CreateConfigurationVersion creates a new configuration version
func (r RootResolver) CreateConfigurationVersion(ctx context.Context, args *struct {
	Input *CreateConfigurationVersionInput
},
) (*ConfigurationVersionMutationPayloadResolver, error) {
	response, err := createConfigurationVersionMutation(ctx, args.Input)
	if err != nil {
		return handleConfigurationVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Variable Mutations */

// SetNamespaceVariables mutation replaces all the variables for the specified namespace
func (r RootResolver) SetNamespaceVariables(ctx context.Context, args *struct{ Input *SetNamespaceVariablesInput }) (*VariableMutationPayloadResolver, error) {
	response, err := setNamespaceVariablesMutation(ctx, args.Input)
	if err != nil {
		return handleVariableMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateNamespaceVariable mutation creates a new variable
func (r RootResolver) CreateNamespaceVariable(ctx context.Context, args *struct{ Input *CreateNamespaceVariableInput }) (*VariableMutationPayloadResolver, error) {
	response, err := createNamespaceVariableMutation(ctx, args.Input)
	if err != nil {
		return handleVariableMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateNamespaceVariable mutation updates an existing variable
func (r RootResolver) UpdateNamespaceVariable(ctx context.Context, args *struct{ Input *UpdateNamespaceVariableInput }) (*VariableMutationPayloadResolver, error) {
	response, err := updateNamespaceVariableMutation(ctx, args.Input)
	if err != nil {
		return handleVariableMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteNamespaceVariable mutation deletes an existing variable
func (r RootResolver) DeleteNamespaceVariable(ctx context.Context, args *struct{ Input *DeleteNamespaceVariableInput }) (*VariableMutationPayloadResolver, error) {
	response, err := deleteNamespaceVariableMutation(ctx, args.Input)
	if err != nil {
		return handleVariableMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* User Queries and Mutations */

// Users query returns a user connection
func (r RootResolver) Users(ctx context.Context, args *UserConnectionQueryArgs) (*UserConnectionResolver, error) {
	return usersQuery(ctx, args)
}

// UpdateUserAdminStatus updates the admin status of a user.
func (r RootResolver) UpdateUserAdminStatus(ctx context.Context, args *struct{ Input *UpdateUserAdminStatusInput }) (*UserMutationPayloadResolver, error) {
	response, err := updateUserAdminStatusMutation(ctx, args.Input)
	if err != nil {
		return handleUserMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// RevokeUserSession revokes a user session.
func (r RootResolver) RevokeUserSession(ctx context.Context, args *struct{ Input *RevokeUserSessionInput }) (*RevokeUserSessionPayloadResolver, error) {
	response, err := revokeUserSessionMutation(ctx, args.Input)
	if err != nil {
		return handleRevokeUserSessionProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateUser creates a new user.
func (r RootResolver) CreateUser(ctx context.Context, args *struct{ Input *CreateUserInput }) (*UserMutationPayloadResolver, error) {
	response, err := createUserMutation(ctx, args.Input)
	if err != nil {
		return handleUserMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteUser deletes a user.
func (r RootResolver) DeleteUser(ctx context.Context, args *struct{ Input *DeleteUserInput }) (*UserMutationPayloadResolver, error) {
	response, err := deleteUserMutation(ctx, args.Input)
	if err != nil {
		return handleUserMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// SetUserPassword sets a user's password.
func (r RootResolver) SetUserPassword(ctx context.Context, args *struct{ Input *SetUserPasswordInput }) (*UserMutationPayloadResolver, error) {
	response, err := setUserPasswordMutation(ctx, args.Input)
	if err != nil {
		return handleUserMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Teams Queries and Mutations */

// Team query returns a team by name
func (r RootResolver) Team(ctx context.Context, args *TeamQueryArgs) (*TeamResolver, error) {
	return teamQuery(ctx, args)
}

// Teams query returns a teams connection
func (r RootResolver) Teams(ctx context.Context, args *TeamConnectionQueryArgs) (*TeamConnectionResolver, error) {
	return teamsQuery(ctx, args)
}

// CreateTeam creates a new team
func (r RootResolver) CreateTeam(ctx context.Context, args *struct{ Input *CreateTeamInput }) (*TeamMutationPayloadResolver, error) {
	response, err := createTeamMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateTeam updates an existing team
func (r RootResolver) UpdateTeam(ctx context.Context, args *struct{ Input *UpdateTeamInput }) (*TeamMutationPayloadResolver, error) {
	response, err := updateTeamMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTeam deletes a team
func (r RootResolver) DeleteTeam(ctx context.Context, args *struct{ Input *DeleteTeamInput }) (*TeamMutationPayloadResolver, error) {
	response, err := deleteTeamMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* TeamMember queries and Mutations */

// AddUserToTeam adds a user to a team.
func (r RootResolver) AddUserToTeam(ctx context.Context, args *struct{ Input *AddUserToTeamInput }) (*TeamMemberMutationPayloadResolver, error) {
	response, err := addUserToTeamMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMemberMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateTeamMember updates an existing team member
func (r RootResolver) UpdateTeamMember(ctx context.Context, args *struct{ Input *UpdateTeamMemberInput }) (*TeamMemberMutationPayloadResolver, error) {
	response, err := updateTeamMemberMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMemberMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// RemoveUserFromTeam removes a user from a team.
func (r RootResolver) RemoveUserFromTeam(ctx context.Context, args *struct{ Input *RemoveUserFromTeamInput }) (*TeamMemberMutationPayloadResolver, error) {
	response, err := removeUserFromTeamMutation(ctx, args.Input)
	if err != nil {
		return handleTeamMemberMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Provider Queries and Mutations */

// TerraformProviders query returns a terraform provider connection
func (r RootResolver) TerraformProviders(ctx context.Context, args *TerraformProviderConnectionQueryArgs) (*TerraformProviderConnectionResolver, error) {
	return terraformProvidersQuery(ctx, args)
}

// TerraformProvider query returns a terraform provider by address
func (r RootResolver) TerraformProvider(ctx context.Context, args *TerraformProviderQueryArgs) (*TerraformProviderResolver, error) {
	return terraformProviderQuery(ctx, args)
}

// CreateTerraformProvider creates a new terraform provider
func (r RootResolver) CreateTerraformProvider(ctx context.Context, args *struct{ Input *CreateTerraformProviderInput }) (*TerraformProviderMutationPayloadResolver, error) {
	response, err := createTerraformProviderMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateTerraformProvider updates a terraform provider
func (r RootResolver) UpdateTerraformProvider(ctx context.Context, args *struct{ Input *UpdateTerraformProviderInput }) (*TerraformProviderMutationPayloadResolver, error) {
	response, err := updateTerraformProviderMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformProvider deletes a terraform provider
func (r RootResolver) DeleteTerraformProvider(ctx context.Context, args *struct{ Input *DeleteTerraformProviderInput }) (*TerraformProviderMutationPayloadResolver, error) {
	response, err := deleteTerraformProviderMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Provider Version Queries and Mutations */

// TerraformProviderVersion query returns a terraform provider version
func (r RootResolver) TerraformProviderVersion(ctx context.Context, args *TerraformProviderVersionQueryArgs) (*TerraformProviderVersionResolver, error) {
	return terraformProviderVersionQuery(ctx, args)
}

// CreateTerraformProviderVersion creates a new terraform provider version
func (r RootResolver) CreateTerraformProviderVersion(ctx context.Context, args *struct {
	Input *CreateTerraformProviderVersionInput
},
) (*TerraformProviderVersionMutationPayloadResolver, error) {
	response, err := createTerraformProviderVersionMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformProviderVersion deletes a terraform provider version
func (r RootResolver) DeleteTerraformProviderVersion(ctx context.Context, args *struct {
	Input *DeleteTerraformProviderVersionInput
},
) (*TerraformProviderVersionMutationPayloadResolver, error) {
	response, err := deleteTerraformProviderVersionMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Provider Platform Queries and Mutations */

// CreateTerraformProviderPlatform creates a new terraform provider platform
func (r RootResolver) CreateTerraformProviderPlatform(ctx context.Context, args *struct {
	Input *CreateTerraformProviderPlatformInput
},
) (*TerraformProviderPlatformMutationPayloadResolver, error) {
	response, err := createTerraformProviderPlatformMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderPlatformMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformProviderPlatform deletes a terraform provider platform
func (r RootResolver) DeleteTerraformProviderPlatform(ctx context.Context, args *struct {
	Input *DeleteTerraformProviderPlatformInput
},
) (*TerraformProviderPlatformMutationPayloadResolver, error) {
	response, err := deleteTerraformProviderPlatformMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderPlatformMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Module Queries and Mutations */

// TerraformModules query returns a terraform module connection
func (r RootResolver) TerraformModules(ctx context.Context, args *TerraformModuleConnectionQueryArgs) (*TerraformModuleConnectionResolver, error) {
	return terraformModulesQuery(ctx, args)
}

// TerraformModule query returns a terraform module by address
func (r RootResolver) TerraformModule(ctx context.Context, args *TerraformModuleQueryArgs) (*TerraformModuleResolver, error) {
	return terraformModuleQuery(ctx, args)
}

// CreateTerraformModule creates a new terraform module
func (r RootResolver) CreateTerraformModule(ctx context.Context, args *struct{ Input *CreateTerraformModuleInput }) (*TerraformModuleMutationPayloadResolver, error) {
	response, err := createTerraformModuleMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateTerraformModule updates a terraform module
func (r RootResolver) UpdateTerraformModule(ctx context.Context, args *struct{ Input *UpdateTerraformModuleInput }) (*TerraformModuleMutationPayloadResolver, error) {
	response, err := updateTerraformModuleMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformModule deletes a terraform module
func (r RootResolver) DeleteTerraformModule(ctx context.Context, args *struct{ Input *DeleteTerraformModuleInput }) (*TerraformModuleMutationPayloadResolver, error) {
	response, err := deleteTerraformModuleMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Module Version Queries and Mutations */

// TerraformModuleVersion query returns a terraform module version
func (r RootResolver) TerraformModuleVersion(ctx context.Context, args *TerraformModuleVersionQueryArgs) (*TerraformModuleVersionResolver, error) {
	return terraformModuleVersionQuery(ctx, args)
}

// CreateTerraformModuleVersion creates a new terraform module version
func (r RootResolver) CreateTerraformModuleVersion(ctx context.Context, args *struct {
	Input *CreateTerraformModuleVersionInput
},
) (*TerraformModuleVersionMutationPayloadResolver, error) {
	response, err := createTerraformModuleVersionMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformModuleVersion deletes a terraform module version
func (r RootResolver) DeleteTerraformModuleVersion(ctx context.Context, args *struct {
	Input *DeleteTerraformModuleVersionInput
},
) (*TerraformModuleVersionMutationPayloadResolver, error) {
	response, err := deleteTerraformModuleVersionMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleVersionMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Terraform Module Attestation Mutations */

// CreateTerraformModuleAttestation creates a new terraform module attestation
func (r RootResolver) CreateTerraformModuleAttestation(ctx context.Context, args *struct {
	Input *CreateTerraformModuleAttestationInput
},
) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	response, err := createTerraformModuleAttestationMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleAttestationMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateTerraformModuleAttestation updates an existing terraform module attestation
func (r RootResolver) UpdateTerraformModuleAttestation(ctx context.Context, args *struct {
	Input *UpdateTerraformModuleAttestationInput
},
) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	response, err := updateTerraformModuleAttestationMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleAttestationMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteTerraformModuleAttestation deletes a terraform module attestation
func (r RootResolver) DeleteTerraformModuleAttestation(ctx context.Context, args *struct {
	Input *DeleteTerraformModuleAttestationInput
},
) (*TerraformModuleAttestationMutationPayloadResolver, error) {
	response, err := deleteTerraformModuleAttestationMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformModuleAttestationMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* GPG Key Queries and Mutations */

// CreateGPGKey creates a new gpg key
func (r RootResolver) CreateGPGKey(ctx context.Context, args *struct{ Input *CreateGPGKeyInput }) (*GPGKeyMutationPayloadResolver, error) {
	response, err := createGPGKeyMutation(ctx, args.Input)
	if err != nil {
		return handleGPGKeyMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// DeleteGPGKey deletes a gpg key
func (r RootResolver) DeleteGPGKey(ctx context.Context, args *struct{ Input *DeleteGPGKeyInput }) (*GPGKeyMutationPayloadResolver, error) {
	response, err := deleteGPGKeyMutation(ctx, args.Input)
	if err != nil {
		return handleGPGKeyMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* TerraformCLIVersions queries and mutations */

// TerraformCLIVersions queries for available TerraformCLIVersions.
func (r RootResolver) TerraformCLIVersions(ctx context.Context) (*TerraformCLIVersionsResolver, error) {
	return terraformCLIVersionsQuery(ctx)
}

// CreateTerraformCLIDownloadURL create a download URL for a Terraform CLI binary.
func (r RootResolver) CreateTerraformCLIDownloadURL(ctx context.Context, args *struct {
	Input *CreateTerraformCLIDownloadURLInput
},
) (*TerraformCLIMutationPayload, error) {
	response, err := createTerraformCLIDownloadURLMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformCLIMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* SCIM queries and mutations */

// CreateSCIMToken generates a token specifically for provisioning SCIM resources.
func (r RootResolver) CreateSCIMToken(ctx context.Context) (*SCIMTokenPayload, error) {
	response, err := createSCIMTokenMutation(ctx)
	if err != nil {
		return handleSCIMMutationProblem(err, nil)
	}

	return response, nil
}

/* ActivityEvents Query */

// ActivityEvents query returns an activity event connection
func (r RootResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	return activityEventsQuery(ctx, args)
}

/* VCSProvider queries and mutations */

// ResetVCSProviderOAuthToken returns a new OAuth authorization code URL that can
// be used to reset an OAuth token.
func (r RootResolver) ResetVCSProviderOAuthToken(ctx context.Context, args *struct {
	Input *ResetVCSProviderOAuthTokenInput
},
) (*ResetVCSProviderOAuthTokenMutationPayloadResolver, error) {
	response, err := resetVCSProviderOAuthTokenMutation(ctx, args.Input)
	if err != nil {
		return handleResetVCSProviderOAuthTokenMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateVCSProvider creates a new vcs provider
func (r RootResolver) CreateVCSProvider(ctx context.Context,
	args *struct{ Input *CreateVCSProviderInput },
) (*VCSProviderMutationPayloadResolver, error) {
	response, err := createVCSProviderMutation(ctx, args.Input)
	if err != nil {
		return handleVCSProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateVCSProvider updates a vcs provider
func (r RootResolver) UpdateVCSProvider(ctx context.Context,
	args *struct{ Input *UpdateVCSProviderInput },
) (*VCSProviderMutationPayloadResolver, error) {
	response, err := updateVCSProviderMutation(ctx, args.Input)
	if err != nil {
		return handleVCSProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteVCSProvider deletes a vcs provider
func (r RootResolver) DeleteVCSProvider(ctx context.Context,
	args *struct{ Input *DeleteVCSProviderInput },
) (*VCSProviderMutationPayloadResolver, error) {
	response, err := deleteVCSProviderMutation(ctx, args.Input)
	if err != nil {
		return handleVCSProviderMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* VCSProviderLink queries and mutations */

// CreateWorkspaceVCSProviderLink creates a new vcs provider link
func (r RootResolver) CreateWorkspaceVCSProviderLink(ctx context.Context,
	args *struct {
		Input *CreateWorkspaceVCSProviderLinkInput
	},
) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	response, err := createWorkspaceVCSProviderLinkMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceVCSProviderLinkMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateWorkspaceVCSProviderLink updates a vcs provider link
func (r RootResolver) UpdateWorkspaceVCSProviderLink(ctx context.Context,
	args *struct {
		Input *UpdateWorkspaceVCSProviderLinkInput
	},
) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	response, err := updateWorkspaceVCSProviderLinkMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceVCSProviderLinkMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteWorkspaceVCSProviderLink deletes a vcs provider link
func (r RootResolver) DeleteWorkspaceVCSProviderLink(ctx context.Context,
	args *struct {
		Input *DeleteWorkspaceVCSProviderLinkInput
	},
) (*WorkspaceVCSProviderLinkMutationPayloadResolver, error) {
	response, err := deleteWorkspaceVCSProviderLinkMutation(ctx, args.Input)
	if err != nil {
		return handleWorkspaceVCSProviderLinkMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// CreateVCSRun creates a vcs run
func (r RootResolver) CreateVCSRun(ctx context.Context,
	args *struct {
		Input *CreateVCSRunInput
	},
) (*CreateVCSRunMutationPayload, error) {
	response, err := createVCSRunMutation(ctx, args.Input)
	if err != nil {
		return handleVCSRunMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Role queries and mutations */

// AvailableRolePermissions returns a list of available role permissions.
func (r RootResolver) AvailableRolePermissions(ctx context.Context) ([]string, error) {
	return availableRolePermissionsQuery(ctx)
}

// Role query returns a role by name
func (r RootResolver) Role(ctx context.Context, args *RoleQueryArgs) (*RoleResolver, error) {
	return roleQuery(ctx, args)
}

// Roles query returns a roles connection
func (r RootResolver) Roles(ctx context.Context, args *RolesConnectionQueryArgs) (*RoleConnectionResolver, error) {
	return rolesQuery(ctx, args)
}

// CreateRole creates a role
func (r RootResolver) CreateRole(ctx context.Context, args *struct{ Input *CreateRoleInput }) (*RoleMutationPayloadResolver, error) {
	response, err := createRoleMutation(ctx, args.Input)
	if err != nil {
		return handleRoleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateRole updates a role
func (r RootResolver) UpdateRole(ctx context.Context, args *struct{ Input *UpdateRoleInput }) (*RoleMutationPayloadResolver, error) {
	response, err := updateRoleMutation(ctx, args.Input)
	if err != nil {
		return handleRoleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteRole updates a role
func (r RootResolver) DeleteRole(ctx context.Context, args *struct{ Input *DeleteRoleInput }) (*RoleMutationPayloadResolver, error) {
	response, err := deleteRoleMutation(ctx, args.Input)
	if err != nil {
		return handleRoleMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Auth Settings Query */

// AuthSettings returns the configured auth settings
func (r RootResolver) AuthSettings(ctx context.Context) *AuthSettingsResolver {
	return authSettingsQuery(ctx)
}

/* Runner Queries and Mutations */

// SharedRunners query returns a shared runners connection
func (r RootResolver) SharedRunners(ctx context.Context, args *ConnectionQueryArgs) (*RunnerConnectionResolver, error) {
	return sharedRunnersQuery(ctx, args)
}

// CreateRunner creates a new runner
func (r RootResolver) CreateRunner(ctx context.Context, args *struct{ Input *CreateRunnerInput }) (*RunnerMutationPayloadResolver, error) {
	response, err := createRunnerMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// UpdateRunner updates an existing runner
func (r RootResolver) UpdateRunner(ctx context.Context, args *struct{ Input *UpdateRunnerInput }) (*RunnerMutationPayloadResolver, error) {
	response, err := updateRunnerMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// DeleteRunner deletes a runner
func (r RootResolver) DeleteRunner(ctx context.Context, args *struct{ Input *DeleteRunnerInput }) (*RunnerMutationPayloadResolver, error) {
	response, err := deleteRunnerMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// AssignServiceAccountToRunner assigns a service account to a runner
func (r RootResolver) AssignServiceAccountToRunner(ctx context.Context, args *struct {
	Input *AssignServiceAccountToRunnerInput
},
) (*RunnerMutationPayloadResolver, error) {
	response, err := assignServiceAccountToRunnerMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// UnassignServiceAccountFromRunner unassigns a service account from a runner
func (r RootResolver) UnassignServiceAccountFromRunner(ctx context.Context, args *struct {
	Input *AssignServiceAccountToRunnerInput
},
) (*RunnerMutationPayloadResolver, error) {
	response, err := unassignServiceAccountFromRunnerMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* Runner Session Subscriptions */

// RunnerSessionEvents subscribes to runner session events
func (r RootResolver) RunnerSessionEvents(ctx context.Context, args *struct {
	Input *RunnerSessionEventSubscriptionInput
},
) (<-chan *RunnerSessionEventResolver, error) {
	return r.runnerSessionEventsSubscription(ctx, args.Input)
}

// RunnerSessionErrorLogEvents subscribes to runner session error log events
func (r RootResolver) RunnerSessionErrorLogEvents(ctx context.Context, args *struct {
	Input *RunnerSessionErrorLogSubscriptionInput
},
) (<-chan *RunnerSessionErrorLogEventResolver, error) {
	return r.runnerSessionErrorLogSubscription(ctx, args.Input)
}

// CreateRunnerSession creates a new runner session.
func (r RootResolver) CreateRunnerSession(ctx context.Context, args *struct {
	Input *CreateRunnerSessionInput
},
) (*CreateRunnerSessionMutationPayloadResolver, error) {
	response, err := createRunnerSessionMutation(ctx, args.Input)
	if err != nil {
		return handleCreateRunnerSessionMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// RunnerSessionHeartbeat sends a runner session heartbeat to the runner session.
func (r RootResolver) RunnerSessionHeartbeat(ctx context.Context, args *struct {
	Input *RunnerSessionHeartbeatInput
},
) (*RunnerSessionHeartbeatErrorMutationPayloadResolver, error) {
	response, err := runnerSessionHeartbeatMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerSessionHeartbeatErrorMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// CreateRunnerSessionError sends a runner session error to the runner session.
func (r RootResolver) CreateRunnerSessionError(ctx context.Context, args *struct {
	Input *CreateRunnerSessionErrorInput
},
) (*RunnerSessionHeartbeatErrorMutationPayloadResolver, error) {
	response, err := createRunnerSessionErrorMutation(ctx, args.Input)
	if err != nil {
		return handleRunnerSessionHeartbeatErrorMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* Resource Limits Query and Mutation */

// ResourceLimits returns the current resource limits
func (r RootResolver) ResourceLimits(ctx context.Context) ([]*ResourceLimitResolver, error) {
	return resourceLimitsQuery(ctx)
}

// UpdateResourceLimit creates or updates a resource limit
func (r RootResolver) UpdateResourceLimit(ctx context.Context,
	args *struct{ Input *UpdateResourceLimitInput },
) (*ResourceLimitMutationPayloadResolver, error) {
	response, err := updateResourceLimitMutation(ctx, args.Input)
	if err != nil {
		return handleResourceLimitMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* TerraformProviderVersionMirror Query and Mutations */

// TerraformProviderVersionMirror query returns a terraform provider version mirror by address.
func (r RootResolver) TerraformProviderVersionMirror(ctx context.Context, args *TerraformProviderVersionMirrorQueryArgs) (*TerraformProviderVersionMirrorResolver, error) {
	return terraformProviderVersionMirrorQuery(ctx, args)
}

// CreateTerraformProviderVersionMirror creates a TerraformProviderVersionMirror.
func (r RootResolver) CreateTerraformProviderVersionMirror(ctx context.Context, args *struct {
	Input *CreateTerraformProviderVersionMirrorInput
},
) (*TerraformProviderVersionMirrorMutationPayloadResolver, error) {
	response, err := createTerraformProviderVersionMirrorMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderVersionMirrorMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// DeleteTerraformProviderVersionMirror deletes a TerraformProviderVersionMirror.
func (r RootResolver) DeleteTerraformProviderVersionMirror(ctx context.Context, args *struct {
	Input *DeleteTerraformProviderVersionMirrorInput
},
) (*TerraformProviderVersionMirrorMutationPayloadResolver, error) {
	response, err := deleteTerraformProviderVersionMirrorMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderVersionMirrorMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* TerraformProviderPlatformMirror Mutations */

// DeleteTerraformProviderPlatformMirror deletes a TerraformProviderPlatformMirror.
func (r RootResolver) DeleteTerraformProviderPlatformMirror(ctx context.Context, args *struct {
	Input *DeleteTerraformProviderPlatformMirrorInput
},
) (*TerraformProviderPlatformMirrorMutationPayloadResolver, error) {
	response, err := deleteTerraformProviderPlatformMirrorMutation(ctx, args.Input)
	if err != nil {
		return handleTerraformProviderPlatformMirrorMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* MaintenanceMode Queries and Mutations */

// MaintenanceMode returns the current maintenance mode
func (r RootResolver) MaintenanceMode(ctx context.Context) (*MaintenanceModeResolver, error) {
	return maintenanceModeQuery(ctx)
}

// EnableMaintenanceMode enables maintenance mode
func (r RootResolver) EnableMaintenanceMode(ctx context.Context,
	args *struct{ Input *EnableMaintenanceModeInput },
) (*MaintenanceModeMutationPayloadResolver, error) {
	response, err := enableMaintenanceModeMutation(ctx, args.Input)
	if err != nil {
		return handleMaintenanceModeMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// DisableMaintenanceMode disables maintenance mode
func (r RootResolver) DisableMaintenanceMode(ctx context.Context,
	args *struct{ Input *DisableMaintenanceModeInput },
) (*MaintenanceModeMutationPayloadResolver, error) {
	response, err := disableMaintenanceModeMutation(ctx, args.Input)
	if err != nil {
		return handleMaintenanceModeMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* Announcement Queries and Mutations */

// Announcements returns a list of announcements
func (r RootResolver) Announcements(ctx context.Context, args *AnnouncementConnectionQueryArgs) (*AnnouncementConnectionResolver, error) {
	return announcementsQuery(ctx, args)
}

// CreateAnnouncement creates a new announcement
func (r RootResolver) CreateAnnouncement(ctx context.Context,
	args *struct{ Input *CreateAnnouncementInput },
) (*AnnouncementMutationPayloadResolver, error) {
	response, err := createAnnouncementMutation(ctx, args.Input)
	if err != nil {
		return handleAnnouncementMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// UpdateAnnouncement updates an existing announcement
func (r RootResolver) UpdateAnnouncement(ctx context.Context,
	args *struct{ Input *UpdateAnnouncementInput },
) (*AnnouncementMutationPayloadResolver, error) {
	response, err := updateAnnouncementMutation(ctx, args.Input)
	if err != nil {
		return handleAnnouncementMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// DeleteAnnouncement deletes an existing announcement
func (r RootResolver) DeleteAnnouncement(ctx context.Context,
	args *struct{ Input *DeleteAnnouncementInput },
) (*AnnouncementMutationPayloadResolver, error) {
	response, err := deleteAnnouncementMutation(ctx, args.Input)
	if err != nil {
		return handleAnnouncementMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

/* NamespaceFavorite Mutations */

// FavoriteNamespace favorites a namespace
func (r RootResolver) FavoriteNamespace(ctx context.Context, args *struct{ Input *NamespaceFavoriteInput }) (*NamespaceFavoriteMutationPayloadResolver, error) {
	response, err := favoriteNamespaceMutation(ctx, args.Input)
	if err != nil {
		return handleNamespaceFavoriteMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// UnfavoriteNamespace unfavorites a namespace
func (r RootResolver) UnfavoriteNamespace(ctx context.Context, args *struct{ Input *NamespaceFavoriteInput }) (*NamespaceUnfavoriteMutationPayloadResolver, error) {
	response, err := unfavoriteNamespaceMutation(ctx, args.Input)
	if err != nil {
		return handleNamespaceUnfavoriteMutationProblem(err, args.Input.ClientMutationID)
	}
	return response, nil
}

// Version returns the version of the API and its components
func (r RootResolver) Version(ctx context.Context) (*VersionResolver, error) {
	return versionQuery(ctx)
}

// Config returns the API configuration
func (r RootResolver) Config(ctx context.Context) (*ConfigResolver, error) {
	return configQuery(ctx)
}

/* Namespace Variable Version Queries */

// NamespaceVariableVersion query returns a namespace variable version by ID
func (r RootResolver) NamespaceVariableVersion(ctx context.Context, args *NamespaceVariableVersionQueryArgs) (*NamespaceVariableVersionResolver, error) {
	return namespaceVariableVersionQuery(ctx, args)
}

/* FederatedRegistry mutations */

// CreateFederatedRegistry creates a new federated registry
func (r RootResolver) CreateFederatedRegistry(ctx context.Context,
	args *struct {
		Input *CreateFederatedRegistryInput
	},
) (*FederatedRegistryMutationPayloadResolver, error) {
	response, err := createFederatedRegistryMutation(ctx, args.Input)
	if err != nil {
		return handleFederatedRegistryMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// UpdateFederatedRegistry updates an existing federated registry
func (r RootResolver) UpdateFederatedRegistry(ctx context.Context,
	args *struct {
		Input *UpdateFederatedRegistryInput
	},
) (*FederatedRegistryMutationPayloadResolver, error) {
	response, err := updateFederatedRegistryMutation(ctx, args.Input)
	if err != nil {
		return handleFederatedRegistryMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

// DeleteFederatedRegistry deletes a federated registry
func (r RootResolver) DeleteFederatedRegistry(ctx context.Context,
	args *struct {
		Input *DeleteFederatedRegistryInput
	},
) (*FederatedRegistryMutationPayloadResolver, error) {
	response, err := deleteFederatedRegistryMutation(ctx, args.Input)
	if err != nil {
		return handleFederatedRegistryMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}

/* Federated Registry Mutation */

// CreateFederatedRegistryTokens creates a list of federated registry tokens.
func (r RootResolver) CreateFederatedRegistryTokens(ctx context.Context,
	args *struct {
		Input *CreateFederatedRegistryTokensInput
	},
) (*FederatedRegistryTokensMutationPayloadResolver, error) {
	response, err := createFederatedRegistryTokensMutation(ctx, args.Input)
	if err != nil {
		return handleCreateFederatedRegistryTokensMutationProblem(err, args.Input.ClientMutationID)
	}

	return response, nil
}
