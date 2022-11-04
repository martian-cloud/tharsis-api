package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	managedidentitytypes "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/types"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* ManagedIdentity Query Resolvers */

// ManagedIdentityConnectionQueryArgs are used to query a managedIdentity connection
type ManagedIdentityConnectionQueryArgs struct {
	ConnectionQueryArgs
	GroupPath        *string
	IncludeInherited *bool
	Search           *string
}

// ManagedIdentityQueryArgs are used to query a single managedIdentity
type ManagedIdentityQueryArgs struct {
	ID string
}

// ManagedIdentityEdgeResolver resolves managedIdentity edges
type ManagedIdentityEdgeResolver struct {
	edge Edge
}

// ManagedIdentityCredentials represents the credentials for a managed identity
type ManagedIdentityCredentials struct {
	Data []byte
}

// Cursor returns an opaque cursor
func (r *ManagedIdentityEdgeResolver) Cursor() (string, error) {
	managedIdentity, ok := r.edge.Node.(models.ManagedIdentity)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&managedIdentity)
	return *cursor, err
}

// Node returns a managedIdentity node
func (r *ManagedIdentityEdgeResolver) Node(ctx context.Context) (*ManagedIdentityResolver, error) {
	managedIdentity, ok := r.edge.Node.(models.ManagedIdentity)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &ManagedIdentityResolver{managedIdentity: &managedIdentity}, nil
}

// ManagedIdentityConnectionResolver resolves a managedIdentity connection
type ManagedIdentityConnectionResolver struct {
	connection Connection
}

// NewManagedIdentityConnectionResolver creates a new ManagedIdentityConnectionResolver
func NewManagedIdentityConnectionResolver(ctx context.Context, input *managedidentity.GetManagedIdentitiesInput) (*ManagedIdentityConnectionResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)

	result, err := managedIdentityService.GetManagedIdentities(ctx, input)
	if err != nil {
		return nil, err
	}

	managedIdentities := result.ManagedIdentities

	// Create edges
	edges := make([]Edge, len(managedIdentities))
	for i, managedIdentity := range managedIdentities {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: managedIdentity}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(managedIdentities) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&managedIdentities[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&managedIdentities[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &ManagedIdentityConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *ManagedIdentityConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *ManagedIdentityConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *ManagedIdentityConnectionResolver) Edges() *[]*ManagedIdentityEdgeResolver {
	resolvers := make([]*ManagedIdentityEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &ManagedIdentityEdgeResolver{edge: edge}
	}
	return &resolvers
}

// ManagedIdentityAccessRuleResolver resolves a managed identity access rule
type ManagedIdentityAccessRuleResolver struct {
	rule *models.ManagedIdentityAccessRule
}

// ID resolver
func (r *ManagedIdentityAccessRuleResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.ManagedIdentityAccessRuleType, r.rule.Metadata.ID))
}

// Metadata resolver
func (r *ManagedIdentityAccessRuleResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.rule.Metadata}
}

// RunStage resolver
func (r *ManagedIdentityAccessRuleResolver) RunStage() string {
	return string(r.rule.RunStage)
}

// ManagedIdentity resolver
func (r *ManagedIdentityAccessRuleResolver) ManagedIdentity(ctx context.Context) (*ManagedIdentityResolver, error) {
	mi, err := loadManagedIdentity(ctx, r.rule.ManagedIdentityID)
	if err != nil {
		return nil, err
	}
	return &ManagedIdentityResolver{managedIdentity: mi}, nil
}

// AllowedUsers resolver
func (r *ManagedIdentityAccessRuleResolver) AllowedUsers(ctx context.Context) ([]*UserResolver, error) {
	resolvers := []*UserResolver{}

	for _, userID := range r.rule.AllowedUserIDs {
		user, err := loadUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &UserResolver{user: user})
	}

	return resolvers, nil
}

// AllowedServiceAccounts resolver
func (r *ManagedIdentityAccessRuleResolver) AllowedServiceAccounts(ctx context.Context) ([]*ServiceAccountResolver, error) {
	resolvers := []*ServiceAccountResolver{}

	for _, serviceAccountID := range r.rule.AllowedServiceAccountIDs {
		sa, err := loadServiceAccount(ctx, serviceAccountID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &ServiceAccountResolver{serviceAccount: sa})
	}

	return resolvers, nil
}

// AllowedTeams resolver
func (r *ManagedIdentityAccessRuleResolver) AllowedTeams(ctx context.Context) ([]*TeamResolver, error) {
	resolvers := []*TeamResolver{}

	for _, teamID := range r.rule.AllowedTeamIDs {
		team, err := loadTeam(ctx, teamID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &TeamResolver{team: team})
	}

	return resolvers, nil
}

// ManagedIdentityResolver resolves a managedIdentity resource
type ManagedIdentityResolver struct {
	managedIdentity *models.ManagedIdentity
}

// ID resolver
func (r *ManagedIdentityResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.ManagedIdentityType, r.managedIdentity.Metadata.ID))
}

// ResourcePath resolver
func (r *ManagedIdentityResolver) ResourcePath() string {
	return r.managedIdentity.ResourcePath
}

// Name resolver
func (r *ManagedIdentityResolver) Name() string {
	return r.managedIdentity.Name
}

// Description resolver
func (r *ManagedIdentityResolver) Description() string {
	return r.managedIdentity.Description
}

// Metadata resolver
func (r *ManagedIdentityResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.managedIdentity.Metadata}
}

// Type resolver
func (r *ManagedIdentityResolver) Type(ctx context.Context) string {
	return string(r.managedIdentity.Type)
}

// Data resolver
func (r *ManagedIdentityResolver) Data(ctx context.Context) string {
	return string(r.managedIdentity.Data)
}

// Group resolver
func (r *ManagedIdentityResolver) Group(ctx context.Context) (*GroupResolver, error) {
	group, err := loadGroup(ctx, r.managedIdentity.GroupID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// AccessRules resolver
func (r *ManagedIdentityResolver) AccessRules(ctx context.Context) ([]*ManagedIdentityAccessRuleResolver, error) {
	resolvers := []*ManagedIdentityAccessRuleResolver{}

	rules, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRules(ctx, r.managedIdentity)
	if err != nil {
		return nil, err
	}

	for _, rule := range rules {
		ruleCopy := rule
		resolvers = append(resolvers, &ManagedIdentityAccessRuleResolver{rule: &ruleCopy})
	}

	return resolvers, nil
}

// CreatedBy resolver
func (r *ManagedIdentityResolver) CreatedBy() string {
	return r.managedIdentity.CreatedBy
}

// ManagedIdentityCredentialsResolver resolves managed identity credentials
type ManagedIdentityCredentialsResolver struct {
	managedIdentityCredentials *ManagedIdentityCredentials
}

// Data resolver
func (r *ManagedIdentityCredentialsResolver) Data() string {
	return string(r.managedIdentityCredentials.Data)
}

func managedIdentityQuery(ctx context.Context, args *ManagedIdentityQueryArgs) (*ManagedIdentityResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)

	managedIdentity, err := managedIdentityService.GetManagedIdentityByID(ctx, gid.FromGlobalID(args.ID))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	if managedIdentity == nil {
		return nil, nil
	}

	return &ManagedIdentityResolver{managedIdentity: managedIdentity}, nil
}

/* ManagedIdentity Mutation Resolvers */

// ManagedIdentityAccessRuleMutationPayload is the response payload for a managed identity access rule mutation
type ManagedIdentityAccessRuleMutationPayload struct {
	ClientMutationID *string
	AccessRule       *models.ManagedIdentityAccessRule
	Problems         []Problem
}

// ManagedIdentityAccessRuleMutationPayloadResolver resolves a ManagedIdentityAccessRuleMutationPayload
type ManagedIdentityAccessRuleMutationPayloadResolver struct {
	ManagedIdentityAccessRuleMutationPayload
}

// AccessRule field resolver
func (r *ManagedIdentityAccessRuleMutationPayloadResolver) AccessRule(ctx context.Context) *ManagedIdentityAccessRuleResolver {
	if r.ManagedIdentityAccessRuleMutationPayload.AccessRule == nil {
		return nil
	}
	return &ManagedIdentityAccessRuleResolver{rule: r.ManagedIdentityAccessRuleMutationPayload.AccessRule}
}

// ManagedIdentityMutationPayload is the response payload for a managedIdentity mutation
type ManagedIdentityMutationPayload struct {
	ClientMutationID *string
	ManagedIdentity  *models.ManagedIdentity
	Problems         []Problem
}

// ManagedIdentityMutationPayloadResolver resolvers a ManagedIdentityMutationPayload
type ManagedIdentityMutationPayloadResolver struct {
	ManagedIdentityMutationPayload
}

// ManagedIdentity field resolver
func (r *ManagedIdentityMutationPayloadResolver) ManagedIdentity(ctx context.Context) *ManagedIdentityResolver {
	if r.ManagedIdentityMutationPayload.ManagedIdentity == nil {
		return nil
	}
	return &ManagedIdentityResolver{managedIdentity: r.ManagedIdentityMutationPayload.ManagedIdentity}
}

// AssignManagedIdentityMutationPayload is the response payload for a managedIdentity mutation
type AssignManagedIdentityMutationPayload struct {
	ClientMutationID *string
	Workspace        *models.Workspace
	Problems         []Problem
}

// AssignManagedIdentityMutationPayloadResolver resolves a ManagedIdentityMutationPayload
type AssignManagedIdentityMutationPayloadResolver struct {
	AssignManagedIdentityMutationPayload
}

// Workspace field resolver
func (r *AssignManagedIdentityMutationPayloadResolver) Workspace(ctx context.Context) *WorkspaceResolver {
	if r.AssignManagedIdentityMutationPayload.Workspace == nil {
		return nil
	}
	return &WorkspaceResolver{workspace: r.AssignManagedIdentityMutationPayload.Workspace}
}

// ManagedIdentityCredentialsMutationPayload is the response payload for a managedIdentity credentials
type ManagedIdentityCredentialsMutationPayload struct {
	ClientMutationID           *string
	ManagedIdentityCredentials *ManagedIdentityCredentials
	Problems                   []Problem
}

// ManagedIdentityCredentialsMutationPayloadResolver resolves managed identity credentials
type ManagedIdentityCredentialsMutationPayloadResolver struct {
	ManagedIdentityCredentialsMutationPayload
}

// ManagedIdentityCredentials field resolver
func (r *ManagedIdentityCredentialsMutationPayloadResolver) ManagedIdentityCredentials(ctx context.Context) *ManagedIdentityCredentialsResolver {
	if r.ManagedIdentityCredentialsMutationPayload.ManagedIdentityCredentials == nil {
		return nil
	}
	return &ManagedIdentityCredentialsResolver{managedIdentityCredentials: r.ManagedIdentityCredentialsMutationPayload.ManagedIdentityCredentials}
}

// CreateManagedIdentityAccessRuleInput is the input for creating a new access rule
type CreateManagedIdentityAccessRuleInput struct {
	ClientMutationID       *string
	ManagedIdentityID      string
	RunStage               models.JobType
	AllowedUsers           []string
	AllowedServiceAccounts []string
	AllowedTeams           []string
}

// UpdateManagedIdentityAccessRuleInput is the input for updating an existing access rule
type UpdateManagedIdentityAccessRuleInput struct {
	ClientMutationID       *string
	ID                     string
	RunStage               models.JobType
	AllowedUsers           []string
	AllowedServiceAccounts []string
	AllowedTeams           []string
}

// DeleteManagedIdentityAccessRuleInput is the input for deleting an access rule
type DeleteManagedIdentityAccessRuleInput struct {
	ClientMutationID *string
	ID               string
}

// CreateManagedIdentityInput contains the input for creating a new managedIdentity
type CreateManagedIdentityInput struct {
	ClientMutationID *string
	AccessRules      *[]struct {
		RunStage               models.JobType
		AllowedUsers           []string
		AllowedServiceAccounts []string
		AllowedTeams           []string
	}
	Type        string
	Name        string
	Description string
	GroupPath   string
	Data        string
}

// UpdateManagedIdentityInput contains the input for updating a managedIdentity
type UpdateManagedIdentityInput struct {
	ClientMutationID *string
	ID               string
	Metadata         *MetadataInput
	Description      string
	Data             string
}

// DeleteManagedIdentityInput contains the input for deleting a managedIdentity
type DeleteManagedIdentityInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	ID               string
}

// AssignManagedIdentityInput is used to assign a managed identity to a workspace
type AssignManagedIdentityInput struct {
	ClientMutationID    *string
	ManagedIdentityID   *string
	ManagedIdentityPath *string
	WorkspacePath       string
}

// CreateManagedIdentityCredentialsInput is for creating credentials for a managed identity.
type CreateManagedIdentityCredentialsInput struct {
	ClientMutationID *string
	ID               string
}

func handleManagedIdentityAccessRuleMutationProblem(e error, clientMutationID *string) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func handleManagedIdentityMutationProblem(e error, clientMutationID *string) (*ManagedIdentityMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ManagedIdentityMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func handleAssignManagedIdentityMutationProblem(e error, clientMutationID *string) (*AssignManagedIdentityMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := AssignManagedIdentityMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &AssignManagedIdentityMutationPayloadResolver{AssignManagedIdentityMutationPayload: payload}, nil
}

func handleManagedIdentityCredentialsMutationProblem(e error, clientMutationID *string) (*ManagedIdentityCredentialsMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := ManagedIdentityCredentialsMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &ManagedIdentityCredentialsMutationPayloadResolver{ManagedIdentityCredentialsMutationPayload: payload}, nil
}

func createManagedIdentityAccessRuleMutation(ctx context.Context, input *CreateManagedIdentityAccessRuleInput) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	allowedUserIDs, err := getManagedIdentityAllowedUserIDs(ctx, input.AllowedUsers)
	if err != nil {
		return nil, err
	}

	allowedServiceAccountIDs, err := getManagedIdentityAllowedServiceAccountIDs(ctx, input.AllowedServiceAccounts)
	if err != nil {
		return nil, err
	}

	allowedTeamIDs, err := getManagedIdentityAllowedTeamIDs(ctx, input.AllowedTeams)
	if err != nil {
		return nil, err
	}

	rule := models.ManagedIdentityAccessRule{
		ManagedIdentityID:        gid.FromGlobalID(input.ManagedIdentityID),
		RunStage:                 input.RunStage,
		AllowedUserIDs:           allowedUserIDs,
		AllowedServiceAccountIDs: allowedServiceAccountIDs,
		AllowedTeamIDs:           allowedTeamIDs,
	}

	createdRule, err := getManagedIdentityService(ctx).CreateManagedIdentityAccessRule(ctx, &rule)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: createdRule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func updateManagedIdentityAccessRuleMutation(ctx context.Context, input *UpdateManagedIdentityAccessRuleInput) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	allowedUserIDs, err := getManagedIdentityAllowedUserIDs(ctx, input.AllowedUsers)
	if err != nil {
		return nil, err
	}

	allowedServiceAccountIDs, err := getManagedIdentityAllowedServiceAccountIDs(ctx, input.AllowedServiceAccounts)
	if err != nil {
		return nil, err
	}

	allowedTeamIDs, err := getManagedIdentityAllowedTeamIDs(ctx, input.AllowedTeams)
	if err != nil {
		return nil, err
	}

	rule, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRule(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	rule.RunStage = input.RunStage
	rule.AllowedUserIDs = allowedUserIDs
	rule.AllowedServiceAccountIDs = allowedServiceAccountIDs
	rule.AllowedTeamIDs = allowedTeamIDs

	updatedRule, err := getManagedIdentityService(ctx).UpdateManagedIdentityAccessRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: updatedRule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func deleteManagedIdentityAccessRuleMutation(ctx context.Context, input *DeleteManagedIdentityAccessRuleInput) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	rule, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRule(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	if err := getManagedIdentityService(ctx).DeleteManagedIdentityAccessRule(ctx, rule); err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: rule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func createManagedIdentityMutation(ctx context.Context, input *CreateManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {

	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}
	groupID := group.Metadata.ID

	managedIdentityCreateOptions := managedidentitytypes.CreateManagedIdentityInput{
		Type:        models.ManagedIdentityType(input.Type),
		Name:        input.Name,
		Description: input.Description,
		GroupID:     groupID,
		Data:        []byte(input.Data),
		AccessRules: []struct {
			RunStage                 models.JobType
			AllowedUserIDs           []string
			AllowedServiceAccountIDs []string
			AllowedTeamIDs           []string
		}{},
	}

	if input.AccessRules != nil {
		for _, r := range *input.AccessRules {
			allowedUsersIDs, miErr := getManagedIdentityAllowedUserIDs(ctx, r.AllowedUsers)
			if miErr != nil {
				return nil, miErr
			}

			allowedServiceAccountIDs, miErr := getManagedIdentityAllowedServiceAccountIDs(ctx, r.AllowedServiceAccounts)
			if miErr != nil {
				return nil, miErr
			}

			allowedTeamIDs, miErr := getManagedIdentityAllowedTeamIDs(ctx, r.AllowedTeams)
			if miErr != nil {
				return nil, miErr
			}

			managedIdentityCreateOptions.AccessRules = append(
				managedIdentityCreateOptions.AccessRules,
				struct {
					RunStage                 models.JobType
					AllowedUserIDs           []string
					AllowedServiceAccountIDs []string
					AllowedTeamIDs           []string
				}{
					RunStage:                 r.RunStage,
					AllowedUserIDs:           allowedUsersIDs,
					AllowedServiceAccountIDs: allowedServiceAccountIDs,
					AllowedTeamIDs:           allowedTeamIDs,
				})
		}
	}

	managedIdentityService := getManagedIdentityService(ctx)

	createdManagedIdentity, err := managedIdentityService.CreateManagedIdentity(ctx, &managedIdentityCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: createdManagedIdentity, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func updateManagedIdentityMutation(ctx context.Context, input *UpdateManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)

	managedIdentity, err := managedIdentityService.UpdateManagedIdentity(ctx, &managedidentitytypes.UpdateManagedIdentityInput{
		ID:          gid.FromGlobalID(input.ID),
		Description: input.Description,
		Data:        []byte(input.Data),
	})
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: managedIdentity, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func deleteManagedIdentityMutation(ctx context.Context, input *DeleteManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)

	mi, err := managedIdentityService.GetManagedIdentityByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		mi.Metadata.Version = v
	}

	deleteOptions := managedidentity.DeleteManagedIdentityInput{
		ManagedIdentity: mi,
	}

	if input.Force != nil {
		deleteOptions.Force = *input.Force
	}

	if err := managedIdentityService.DeleteManagedIdentity(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: mi, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func assignManagedIdentityMutation(ctx context.Context, input *AssignManagedIdentityInput) (*AssignManagedIdentityMutationPayloadResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)
	workspaceService := getWorkspaceService(ctx)

	var identityID string
	workspace, err := workspaceService.GetWorkspaceByFullPath(ctx, input.WorkspacePath)
	if err != nil {
		return nil, err
	}

	if input.ManagedIdentityID != nil && *input.ManagedIdentityID != "" {
		identityID = gid.FromGlobalID(*input.ManagedIdentityID)
	} else if input.ManagedIdentityPath != nil {
		id, err := getManagedIdentityIDByPath(ctx, *input.ManagedIdentityPath)
		if err != nil {
			return nil, err
		}

		identityID = id
	} else {
		return nil, errors.NewError(errors.EInvalid, "Either managedIdentityId or managedIdentityPath is required")
	}

	if err := managedIdentityService.AddManagedIdentityToWorkspace(ctx, identityID, workspace.Metadata.ID); err != nil {
		return nil, err
	}

	payload := AssignManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: workspace, Problems: []Problem{}}
	return &AssignManagedIdentityMutationPayloadResolver{AssignManagedIdentityMutationPayload: payload}, nil
}

func unassignManagedIdentityMutation(ctx context.Context, input *AssignManagedIdentityInput) (*AssignManagedIdentityMutationPayloadResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)
	workspaceService := getWorkspaceService(ctx)

	var identityID string
	workspace, err := workspaceService.GetWorkspaceByFullPath(ctx, input.WorkspacePath)
	if err != nil {
		return nil, err
	}

	if input.ManagedIdentityID != nil && *input.ManagedIdentityID != "" {
		identityID = gid.FromGlobalID(*input.ManagedIdentityID)
	} else if input.ManagedIdentityPath != nil {
		id, err := getManagedIdentityIDByPath(ctx, *input.ManagedIdentityPath)
		if err != nil {
			return nil, err
		}

		identityID = id
	} else {
		return nil, errors.NewError(errors.EInvalid, "Either managedIdentityId or managedIdentityPath is required")
	}

	if err := managedIdentityService.RemoveManagedIdentityFromWorkspace(ctx, identityID, workspace.Metadata.ID); err != nil {
		return nil, err
	}

	payload := AssignManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: workspace, Problems: []Problem{}}
	return &AssignManagedIdentityMutationPayloadResolver{AssignManagedIdentityMutationPayload: payload}, nil
}

func createManagedIdentityCredentialsMutation(ctx context.Context,
	input *CreateManagedIdentityCredentialsInput) (*ManagedIdentityCredentialsMutationPayloadResolver, error) {
	managedIdentityService := getManagedIdentityService(ctx)

	managedIdentity, err := managedIdentityService.GetManagedIdentityByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	credentials, err := managedIdentityService.CreateCredentials(ctx, managedIdentity)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityCredentialsMutationPayload{
		ClientMutationID:           input.ClientMutationID,
		ManagedIdentityCredentials: &ManagedIdentityCredentials{Data: credentials},
		Problems:                   []Problem{},
	}

	return &ManagedIdentityCredentialsMutationPayloadResolver{ManagedIdentityCredentialsMutationPayload: payload}, nil
}

func getManagedIdentityIDByPath(ctx context.Context, path string) (string, error) {
	managedIdentityService := getManagedIdentityService(ctx)
	identity, err := managedIdentityService.GetManagedIdentityByPath(ctx, path)
	if err != nil {
		return "", err
	}

	return identity.Metadata.ID, nil
}

func getManagedIdentityAllowedUserIDs(ctx context.Context, usernames []string) ([]string, error) {
	userService := getUserService(ctx)
	response := []string{}

	for _, username := range usernames {
		user, err := userService.GetUserByUsername(ctx, username)
		if err != nil {
			return nil, err
		}
		response = append(response, user.Metadata.ID)
	}

	return response, nil
}

func getManagedIdentityAllowedServiceAccountIDs(ctx context.Context, serviceAccountPaths []string) ([]string, error) {
	saService := getSAService(ctx)
	response := []string{}

	for _, path := range serviceAccountPaths {
		sa, err := saService.GetServiceAccountByPath(ctx, path)
		if err != nil {
			return nil, err
		}
		response = append(response, sa.Metadata.ID)
	}

	return response, nil
}

func getManagedIdentityAllowedTeamIDs(ctx context.Context, teamNames []string) ([]string, error) {
	teamService := getTeamService(ctx)
	response := []string{}

	for _, teamName := range teamNames {
		team, err := teamService.GetTeamByName(ctx, teamName)
		if err != nil {
			return nil, err
		}
		response = append(response, team.Metadata.ID)
	}

	return response, nil
}

/* ManagedIdentity loader */

const managedIdentityLoaderKey = "managedIdentity"

// RegisterManagedIdentityLoader registers a managedIdentity loader function
func RegisterManagedIdentityLoader(collection *loader.Collection) {
	collection.Register(managedIdentityLoaderKey, managedIdentityBatchFunc)
}

func loadManagedIdentity(ctx context.Context, id string) (*models.ManagedIdentity, error) {
	ldr, err := loader.Extract(ctx, managedIdentityLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	managedIdentity, ok := data.(models.ManagedIdentity)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &managedIdentity, nil
}

func managedIdentityBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {

	managedIdentities, err := getManagedIdentityService(ctx).GetManagedIdentitiesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range managedIdentities {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}

/* ManagedIdentityAccessRule loader */

const managedIdentityAccessRuleLoaderKey = "managedIdentityAccessRule"

// RegisterManagedIdentityAccessRuleLoader registers a managedIdentityAccessRule loader function
func RegisterManagedIdentityAccessRuleLoader(collection *loader.Collection) {
	collection.Register(managedIdentityAccessRuleLoaderKey, managedIdentityAccessRuleBatchFunc)
}

func loadManagedIdentityAccessRule(ctx context.Context, id string) (*models.ManagedIdentityAccessRule, error) {
	ldr, err := loader.Extract(ctx, managedIdentityAccessRuleLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	managedIdentityAccessRule, ok := data.(models.ManagedIdentityAccessRule)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &managedIdentityAccessRule, nil
}

func managedIdentityAccessRuleBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {

	rules, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRulesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range rules {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
