package resolver

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* ManagedIdentity Query Resolvers */

// ManagedIdentityConnectionQueryArgs are used to query a managedIdentity connection
type ManagedIdentityConnectionQueryArgs struct {
	ConnectionQueryArgs
	IncludeInherited *bool
	Search           *string
}

// ManagedIdentityQueryArgs are used to query a single managedIdentity
// DEPRECATED: use node query instead
type ManagedIdentityQueryArgs struct {
	ID   *string
	Path *string
}

// ManagedIdentityCredentials represents the credentials for a managed identity
type ManagedIdentityCredentials struct {
	Data []byte
}

// ManagedIdentityEdgeResolver resolves managedIdentity edges
type ManagedIdentityEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *ManagedIdentityEdgeResolver) Cursor() (string, error) {
	managedIdentity, ok := r.edge.Node.(models.ManagedIdentity)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&managedIdentity)
	return *cursor, err
}

// Node returns a managedIdentity node
func (r *ManagedIdentityEdgeResolver) Node() (*ManagedIdentityResolver, error) {
	managedIdentity, ok := r.edge.Node.(models.ManagedIdentity)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &ManagedIdentityResolver{managedIdentity: &managedIdentity}, nil
}

// ManagedIdentityConnectionResolver resolves a managedIdentity connection
type ManagedIdentityConnectionResolver struct {
	connection Connection
}

// NewManagedIdentityConnectionResolver creates a new ManagedIdentityConnectionResolver
func NewManagedIdentityConnectionResolver(ctx context.Context, input *managedidentity.GetManagedIdentitiesInput) (*ManagedIdentityConnectionResolver, error) {
	managedIdentityService := getServiceCatalog(ctx).ManagedIdentityService

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
	return graphql.ID(r.rule.GetGlobalID())
}

// Metadata resolver
func (r *ManagedIdentityAccessRuleResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.rule.Metadata}
}

// Type resolver
func (r *ManagedIdentityAccessRuleResolver) Type() string {
	return string(r.rule.Type)
}

// RunStage resolver
func (r *ManagedIdentityAccessRuleResolver) RunStage() string {
	return string(r.rule.RunStage)
}

// ModuleAttestationPolicies resolver
func (r *ManagedIdentityAccessRuleResolver) ModuleAttestationPolicies() *[]models.ManagedIdentityAccessRuleModuleAttestationPolicy {
	if r.rule.ModuleAttestationPolicies == nil {
		return nil
	}
	return &r.rule.ModuleAttestationPolicies
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
func (r *ManagedIdentityAccessRuleResolver) AllowedUsers(ctx context.Context) (*[]*UserResolver, error) {
	resolvers := []*UserResolver{}

	for _, userID := range r.rule.AllowedUserIDs {
		user, err := loadUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &UserResolver{user: user})
	}

	return &resolvers, nil
}

// AllowedServiceAccounts resolver
func (r *ManagedIdentityAccessRuleResolver) AllowedServiceAccounts(ctx context.Context) (*[]*ServiceAccountResolver, error) {
	resolvers := []*ServiceAccountResolver{}

	for _, serviceAccountID := range r.rule.AllowedServiceAccountIDs {
		sa, err := loadServiceAccount(ctx, serviceAccountID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &ServiceAccountResolver{serviceAccount: sa})
	}

	return &resolvers, nil
}

// AllowedTeams resolver
func (r *ManagedIdentityAccessRuleResolver) AllowedTeams(ctx context.Context) (*[]*TeamResolver, error) {
	resolvers := []*TeamResolver{}

	for _, teamID := range r.rule.AllowedTeamIDs {
		team, err := loadTeam(ctx, teamID)
		if err != nil {
			return nil, err
		}
		resolvers = append(resolvers, &TeamResolver{team: team})
	}

	return &resolvers, nil
}

// VerifyStateLineage resolver
func (r *ManagedIdentityAccessRuleResolver) VerifyStateLineage() bool {
	return r.rule.VerifyStateLineage
}

// ManagedIdentityResolver resolves a managedIdentity resource
type ManagedIdentityResolver struct {
	managedIdentity *models.ManagedIdentity
}

// ID resolver
func (r *ManagedIdentityResolver) ID() graphql.ID {
	return graphql.ID(r.managedIdentity.GetGlobalID())
}

// GroupPath resolver
func (r *ManagedIdentityResolver) GroupPath() string {
	return r.managedIdentity.GetGroupPath()
}

// ResourcePath resolver
func (r *ManagedIdentityResolver) ResourcePath() string {
	return r.managedIdentity.GetResourcePath()
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
func (r *ManagedIdentityResolver) Type() string {
	return string(r.managedIdentity.Type)
}

// Data resolver
func (r *ManagedIdentityResolver) Data() string {
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

	rules, err := getServiceCatalog(ctx).ManagedIdentityService.GetManagedIdentityAccessRules(ctx, r.managedIdentity)
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

// AliasSourceID resolver
func (r *ManagedIdentityResolver) AliasSourceID() *string {
	if r.managedIdentity.AliasSourceID == nil {
		return nil
	}

	aliasID := gid.ToGlobalID(types.ManagedIdentityModelType, *r.managedIdentity.AliasSourceID)
	return &aliasID
}

// AliasSource resolver
func (r *ManagedIdentityResolver) AliasSource(ctx context.Context) (*ManagedIdentityResolver, error) {
	if r.managedIdentity.AliasSourceID == nil {
		return nil, nil
	}

	identity, err := loadManagedIdentity(ctx, *r.managedIdentity.AliasSourceID)
	if err != nil {
		return nil, err
	}

	return &ManagedIdentityResolver{managedIdentity: identity}, nil
}

// Aliases resolver
func (r *ManagedIdentityResolver) Aliases(ctx context.Context, args *ManagedIdentityConnectionQueryArgs) (*ManagedIdentityConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := managedidentity.GetManagedIdentitiesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		AliasSourceID:     &r.managedIdentity.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.ManagedIdentitySortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewManagedIdentityConnectionResolver(ctx, &input)
}

// Workspaces resolver
func (r *ManagedIdentityResolver) Workspaces(ctx context.Context, args *WorkspaceConnectionQueryArgs) (*WorkspaceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := workspace.GetWorkspacesInput{
		PaginationOptions:         &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		AssignedManagedIdentityID: &r.managedIdentity.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.WorkspaceSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewWorkspaceConnectionResolver(ctx, &input)
}

// IsAlias resolver
func (r *ManagedIdentityResolver) IsAlias() bool {
	return r.managedIdentity.IsAlias()
}

// ManagedIdentityCredentialsResolver resolves managed identity credentials
type ManagedIdentityCredentialsResolver struct {
	managedIdentityCredentials *ManagedIdentityCredentials
}

// Data resolver
func (r *ManagedIdentityCredentialsResolver) Data() string {
	return string(r.managedIdentityCredentials.Data)
}

// DEPRECATED: use node query instead since it supports both TRN and GID
func managedIdentityQuery(ctx context.Context, args *ManagedIdentityQueryArgs) (*ManagedIdentityResolver, error) {
	var valueToResolve string
	switch {
	case args.ID != nil && args.Path != nil:
		return nil, errors.New("cannot specify both id and path", errors.WithErrorCode(errors.EInvalid))
	case args.ID != nil:
		valueToResolve = *args.ID
	case args.Path != nil:
		valueToResolve = types.ManagedIdentityModelType.BuildTRN(*args.Path)
	default:
		return nil, errors.New("must specify either id or path", errors.WithErrorCode(errors.EInvalid))
	}

	model, err := getServiceCatalog(ctx).FetchModel(ctx, valueToResolve)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	managedIdentity, ok := model.(*models.ManagedIdentity)
	if !ok {
		return nil, errors.New("expected managed identity type, got %T", model)
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
func (r *ManagedIdentityAccessRuleMutationPayloadResolver) AccessRule() *ManagedIdentityAccessRuleResolver {
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

// ManagedIdentityMutationPayloadResolver resolves a ManagedIdentityMutationPayload
type ManagedIdentityMutationPayloadResolver struct {
	ManagedIdentityMutationPayload
}

// ManagedIdentity field resolver
func (r *ManagedIdentityMutationPayloadResolver) ManagedIdentity() *ManagedIdentityResolver {
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
func (r *AssignManagedIdentityMutationPayloadResolver) Workspace() *WorkspaceResolver {
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
func (r *ManagedIdentityCredentialsMutationPayloadResolver) ManagedIdentityCredentials() *ManagedIdentityCredentialsResolver {
	if r.ManagedIdentityCredentialsMutationPayload.ManagedIdentityCredentials == nil {
		return nil
	}
	return &ManagedIdentityCredentialsResolver{managedIdentityCredentials: r.ManagedIdentityCredentialsMutationPayload.ManagedIdentityCredentials}
}

// CreateManagedIdentityAccessRuleInput is the input for creating a new access rule
type CreateManagedIdentityAccessRuleInput struct {
	ClientMutationID          *string
	AllowedUserIDs            *[]string
	AllowedServiceAccountIDs  *[]string
	AllowedTeamIDs            *[]string
	AllowedTeams              *[]string // DEPRECATED: use AllowedTeamIDs instead
	ModuleAttestationPolicies *[]models.ManagedIdentityAccessRuleModuleAttestationPolicy
	AllowedUsers              *[]string // DEPRECATED: use AllowedUserIDs instead
	AllowedServiceAccounts    *[]string // DEPRECATED: use AllowServiceAccountIDs instead
	VerifyStateLineage        *bool
	Type                      models.ManagedIdentityAccessRuleType
	RunStage                  models.JobType
	ManagedIdentityID         string
}

// UpdateManagedIdentityAccessRuleInput is the input for updating an existing access rule
type UpdateManagedIdentityAccessRuleInput struct {
	ClientMutationID          *string
	ModuleAttestationPolicies *[]models.ManagedIdentityAccessRuleModuleAttestationPolicy
	AllowedUserIDs            *[]string
	AllowedServiceAccountIDs  *[]string
	AllowedTeamIDs            *[]string
	AllowedUsers              *[]string // DEPRECATED: use AllowedUserIDs instead
	AllowedServiceAccounts    *[]string // DEPRECATED: use AllowedServiceAccountIDs instead
	AllowedTeams              *[]string // DEPRECATED: use AllowedTeamIDs instead
	VerifyStateLineage        *bool
	ID                        string
	RunStage                  models.JobType
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
		ModuleAttestationPolicies *[]models.ManagedIdentityAccessRuleModuleAttestationPolicy
		AllowedUserIDs            *[]string
		AllowedServiceAccountIDs  *[]string
		AllowedTeamIDs            *[]string
		AllowedUsers              *[]string // DEPRECATED: use AllowedUserIDs instead
		AllowedServiceAccounts    *[]string // DEPRECATED: use AllowedServiceAccountIDs instead
		AllowedTeams              *[]string // DEPRECATED: use AllowedTeamIDs instead
		VerifyStateLineage        *bool
		Type                      models.ManagedIdentityAccessRuleType
		RunStage                  models.JobType
	}
	Type        string
	Name        string
	Description string
	GroupPath   *string // DEPRECATED: use GroupID instead with a TRN
	GroupID     *string
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
	ManagedIdentityPath *string // DEPRECATED: use ManagedIdentityID instead with a TRN
	WorkspaceID         *string
	WorkspacePath       *string // DEPRECATED: use WorkspaceID instead with a TRN
}

// CreateManagedIdentityCredentialsInput is for creating credentials for a managed identity.
type CreateManagedIdentityCredentialsInput struct {
	ClientMutationID *string
	ID               string
}

// CreateManagedIdentityAliasInput is the input for creating a managed identity alias.
type CreateManagedIdentityAliasInput struct {
	ClientMutationID *string
	Name             string
	AliasSourceID    *string
	AliasSourcePath  *string // DEPRECATED: use AliasSourceID instead with a TRN
	GroupID          *string
	GroupPath        *string // DEPRECATED: use GroupID instead with a TRN
}

// MoveManagedIdentityInput is the input for moving a managed identity to another parent group.
type MoveManagedIdentityInput struct {
	ClientMutationID  *string
	ManagedIdentityID string
	NewParentPath     *string // DEPRECATED: use NewParentID instead with a TRN
	NewParentID       *string
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
	var allowedUserIDs, allowedServiceAccountIDs, allowedTeamIDs []string
	var moduleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
	var err error

	switch input.Type {
	case models.ManagedIdentityAccessRuleEligiblePrincipals:
		if input.AllowedUserIDs != nil && input.AllowedUsers != nil {
			return nil, errors.New("cannot specify both allowedUserIDs and allowedUsers", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedUserIDs != nil {
			allowedUserIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedUserIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedUsers != nil {
			allowedUserIDs, err = getManagedIdentityAllowedUserIDs(ctx, *input.AllowedUsers)
			if err != nil {
				return nil, err
			}
		} else {
			allowedUserIDs = []string{}
		}

		if input.AllowedServiceAccountIDs != nil && input.AllowedServiceAccounts != nil {
			return nil, errors.New("cannot specify both allowedServiceAccountIDs and allowedServiceAccounts", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedServiceAccountIDs != nil {
			allowedServiceAccountIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedServiceAccountIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedServiceAccounts != nil {
			allowedServiceAccountIDs, err = getManagedIdentityAllowedServiceAccountIDs(ctx, *input.AllowedServiceAccounts)
			if err != nil {
				return nil, err
			}
		} else {
			allowedServiceAccountIDs = []string{}
		}

		if input.AllowedTeamIDs != nil && input.AllowedTeams != nil {
			return nil, errors.New("cannot specify both allowedTeamIDs and allowedTeams", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedTeamIDs != nil {
			allowedTeamIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedTeamIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedTeams != nil {
			allowedTeamIDs, err = getManagedIdentityAllowedTeamIDs(ctx, *input.AllowedTeams)
			if err != nil {
				return nil, err
			}
		} else {
			allowedTeamIDs = []string{}
		}
	case models.ManagedIdentityAccessRuleModuleAttestation:
		if input.ModuleAttestationPolicies != nil {
			moduleAttestationPolicies = *input.ModuleAttestationPolicies
		}
	default:
		return nil, errors.New("invalid managed identity rule type: %s", input.Type, errors.WithErrorCode(errors.EInvalid))
	}

	serviceCatalog := getServiceCatalog(ctx)

	managedIdentityID, err := serviceCatalog.FetchModelID(ctx, input.ManagedIdentityID)
	if err != nil {
		return nil, err
	}

	var verifyStateLineage bool
	if input.VerifyStateLineage != nil {
		verifyStateLineage = *input.VerifyStateLineage
	}

	rule := models.ManagedIdentityAccessRule{
		ManagedIdentityID:         managedIdentityID,
		Type:                      input.Type,
		RunStage:                  input.RunStage,
		ModuleAttestationPolicies: moduleAttestationPolicies,
		AllowedUserIDs:            allowedUserIDs,
		AllowedServiceAccountIDs:  allowedServiceAccountIDs,
		AllowedTeamIDs:            allowedTeamIDs,
		VerifyStateLineage:        verifyStateLineage,
	}

	createdRule, err := serviceCatalog.ManagedIdentityService.CreateManagedIdentityAccessRule(ctx, &rule)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: createdRule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func updateManagedIdentityAccessRuleMutation(ctx context.Context, input *UpdateManagedIdentityAccessRuleInput) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	var allowedUserIDs, allowedServiceAccountIDs, allowedTeamIDs []string
	var moduleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy

	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	rule, err := serviceCatalog.ManagedIdentityService.GetManagedIdentityAccessRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	switch rule.Type {
	case models.ManagedIdentityAccessRuleEligiblePrincipals:
		if input.AllowedUserIDs != nil && input.AllowedUsers != nil {
			return nil, errors.New("cannot specify both allowedUserIDs and allowedUsers", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedUserIDs != nil {
			allowedUserIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedUserIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedUsers != nil {
			allowedUserIDs, err = getManagedIdentityAllowedUserIDs(ctx, *input.AllowedUsers)
			if err != nil {
				return nil, err
			}
		} else {
			allowedUserIDs = []string{}
		}

		if input.AllowedServiceAccountIDs != nil && input.AllowedServiceAccounts != nil {
			return nil, errors.New("cannot specify both allowedServiceAccountIDs and allowedServiceAccounts", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedServiceAccountIDs != nil {
			allowedServiceAccountIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedServiceAccountIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedServiceAccounts != nil {
			allowedServiceAccountIDs, err = getManagedIdentityAllowedServiceAccountIDs(ctx, *input.AllowedServiceAccounts)
			if err != nil {
				return nil, err
			}
		} else {
			allowedServiceAccountIDs = []string{}
		}

		if input.AllowedTeamIDs != nil && input.AllowedTeams != nil {
			return nil, errors.New("cannot specify both allowedTeamIDs and allowedTeams", errors.WithErrorCode(errors.EInvalid))
		}
		if input.AllowedTeamIDs != nil {
			allowedTeamIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *input.AllowedTeamIDs)
			if err != nil {
				return nil, err
			}
		} else if input.AllowedTeams != nil {
			allowedTeamIDs, err = getManagedIdentityAllowedTeamIDs(ctx, *input.AllowedTeams)
			if err != nil {
				return nil, err
			}
		} else {
			allowedTeamIDs = []string{}
		}
	case models.ManagedIdentityAccessRuleModuleAttestation:
		if input.ModuleAttestationPolicies != nil {
			moduleAttestationPolicies = *input.ModuleAttestationPolicies
		}
	default:
		return nil, fmt.Errorf("unexpected managed identity rule type: %s", rule.Type)
	}

	rule.RunStage = input.RunStage
	rule.ModuleAttestationPolicies = moduleAttestationPolicies
	rule.AllowedUserIDs = allowedUserIDs
	rule.AllowedServiceAccountIDs = allowedServiceAccountIDs
	rule.AllowedTeamIDs = allowedTeamIDs

	verifyStateLineage := rule.VerifyStateLineage
	if input.VerifyStateLineage != nil {
		verifyStateLineage = *input.VerifyStateLineage
	}
	rule.VerifyStateLineage = verifyStateLineage

	updatedRule, err := serviceCatalog.ManagedIdentityService.UpdateManagedIdentityAccessRule(ctx, rule)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: updatedRule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func deleteManagedIdentityAccessRuleMutation(ctx context.Context, input *DeleteManagedIdentityAccessRuleInput) (*ManagedIdentityAccessRuleMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	rule, err := serviceCatalog.ManagedIdentityService.GetManagedIdentityAccessRuleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := serviceCatalog.ManagedIdentityService.DeleteManagedIdentityAccessRule(ctx, rule); err != nil {
		return nil, err
	}

	payload := ManagedIdentityAccessRuleMutationPayload{ClientMutationID: input.ClientMutationID, AccessRule: rule, Problems: []Problem{}}
	return &ManagedIdentityAccessRuleMutationPayloadResolver{ManagedIdentityAccessRuleMutationPayload: payload}, nil
}

func createManagedIdentityAliasMutation(ctx context.Context, input *CreateManagedIdentityAliasInput) (*ManagedIdentityMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	aliasSourceID, err := toModelID(ctx, input.AliasSourcePath, input.AliasSourceID, types.ManagedIdentityModelType)
	if err != nil {
		return nil, err
	}

	createOptions := &managedidentity.CreateManagedIdentityAliasInput{
		Name:          input.Name,
		GroupID:       groupID,
		AliasSourceID: aliasSourceID,
	}

	createdManagedIdentity, err := getServiceCatalog(ctx).ManagedIdentityService.CreateManagedIdentityAlias(ctx, createOptions)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: createdManagedIdentity, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func deleteManagedIdentityAliasMutation(ctx context.Context, input *DeleteManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	mi, err := serviceCatalog.ManagedIdentityService.GetManagedIdentityByID(ctx, id)
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

	if err := serviceCatalog.ManagedIdentityService.DeleteManagedIdentityAlias(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: mi, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func createManagedIdentityMutation(ctx context.Context, input *CreateManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	managedIdentityCreateOptions := managedidentity.CreateManagedIdentityInput{
		Type:        models.ManagedIdentityType(input.Type),
		Name:        input.Name,
		Description: input.Description,
		GroupID:     groupID,
		Data:        []byte(input.Data),
		AccessRules: []struct {
			Type                      models.ManagedIdentityAccessRuleType
			RunStage                  models.JobType
			ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
			AllowedUserIDs            []string
			AllowedServiceAccountIDs  []string
			AllowedTeamIDs            []string
			VerifyStateLineage        bool
		}{},
	}

	if input.AccessRules != nil {
		for _, r := range *input.AccessRules {
			var allowedUserIDs, allowedServiceAccountIDs, allowedTeamIDs []string
			var moduleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy

			switch r.Type {
			case models.ManagedIdentityAccessRuleEligiblePrincipals:
				if r.AllowedUserIDs != nil && r.AllowedUsers != nil {
					return nil, errors.New("cannot specify both allowedUserIDs and allowedUsers", errors.WithErrorCode(errors.EInvalid))
				}
				if r.AllowedUserIDs != nil {
					allowedUserIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *r.AllowedUserIDs)
					if err != nil {
						return nil, err
					}
				} else if r.AllowedUsers != nil {
					allowedUserIDs, err = getManagedIdentityAllowedUserIDs(ctx, *r.AllowedUsers)
					if err != nil {
						return nil, err
					}
				} else {
					allowedUserIDs = []string{}
				}

				if r.AllowedServiceAccountIDs != nil && r.AllowedServiceAccounts != nil {
					return nil, errors.New("cannot specify both allowedServiceAccountIDs and allowedServiceAccounts", errors.WithErrorCode(errors.EInvalid))
				}
				if r.AllowedServiceAccountIDs != nil {
					allowedServiceAccountIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *r.AllowedServiceAccountIDs)
					if err != nil {
						return nil, err
					}
				} else if r.AllowedServiceAccounts != nil {
					allowedServiceAccountIDs, err = getManagedIdentityAllowedServiceAccountIDs(ctx, *r.AllowedServiceAccounts)
					if err != nil {
						return nil, err
					}
				} else {
					allowedServiceAccountIDs = []string{}
				}

				if r.AllowedTeamIDs != nil && r.AllowedTeams != nil {
					return nil, errors.New("cannot specify both allowedTeamIDs and allowedTeams", errors.WithErrorCode(errors.EInvalid))
				}
				if r.AllowedTeamIDs != nil {
					allowedTeamIDs, err = resolveManagedIdentityAllowedPrincipalIDs(ctx, *r.AllowedTeamIDs)
					if err != nil {
						return nil, err
					}
				} else if r.AllowedTeams != nil {
					allowedTeamIDs, err = getManagedIdentityAllowedTeamIDs(ctx, *r.AllowedTeams)
					if err != nil {
						return nil, err
					}
				} else {
					allowedTeamIDs = []string{}
				}
			case models.ManagedIdentityAccessRuleModuleAttestation:
				if r.ModuleAttestationPolicies != nil {
					moduleAttestationPolicies = *r.ModuleAttestationPolicies
				}
			default:
				return nil, errors.New("invalid managed identity rule type: %s", input.Type, errors.WithErrorCode(errors.EInvalid))
			}

			var verifyStateLineage bool
			if r.VerifyStateLineage != nil {
				verifyStateLineage = *r.VerifyStateLineage
			}

			managedIdentityCreateOptions.AccessRules = append(
				managedIdentityCreateOptions.AccessRules,
				struct {
					Type                      models.ManagedIdentityAccessRuleType
					RunStage                  models.JobType
					ModuleAttestationPolicies []models.ManagedIdentityAccessRuleModuleAttestationPolicy
					AllowedUserIDs            []string
					AllowedServiceAccountIDs  []string
					AllowedTeamIDs            []string
					VerifyStateLineage        bool
				}{
					Type:                      r.Type,
					RunStage:                  r.RunStage,
					ModuleAttestationPolicies: moduleAttestationPolicies,
					AllowedUserIDs:            allowedUserIDs,
					AllowedServiceAccountIDs:  allowedServiceAccountIDs,
					AllowedTeamIDs:            allowedTeamIDs,
					VerifyStateLineage:        verifyStateLineage,
				})
		}
	}

	createdManagedIdentity, err := getServiceCatalog(ctx).ManagedIdentityService.CreateManagedIdentity(ctx, &managedIdentityCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: createdManagedIdentity, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func updateManagedIdentityMutation(ctx context.Context, input *UpdateManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := serviceCatalog.ManagedIdentityService.UpdateManagedIdentity(ctx, &managedidentity.UpdateManagedIdentityInput{
		ID:          id,
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
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	mi, err := serviceCatalog.ManagedIdentityService.GetManagedIdentityByID(ctx, id)
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

	if err := serviceCatalog.ManagedIdentityService.DeleteManagedIdentity(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, ManagedIdentity: mi, Problems: []Problem{}}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

func assignManagedIdentityMutation(ctx context.Context, input *AssignManagedIdentityInput) (*AssignManagedIdentityMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	identityID, err := toModelID(ctx, input.ManagedIdentityPath, input.ManagedIdentityID, types.ManagedIdentityModelType)
	if err != nil {
		return nil, err
	}

	serviceCatalog := getServiceCatalog(ctx)

	if err = serviceCatalog.ManagedIdentityService.AddManagedIdentityToWorkspace(ctx, identityID, workspaceID); err != nil {
		return nil, err
	}

	workspace, err := serviceCatalog.WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	payload := AssignManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: workspace, Problems: []Problem{}}
	return &AssignManagedIdentityMutationPayloadResolver{AssignManagedIdentityMutationPayload: payload}, nil
}

func unassignManagedIdentityMutation(ctx context.Context, input *AssignManagedIdentityInput) (*AssignManagedIdentityMutationPayloadResolver, error) {
	workspaceID, err := toModelID(ctx, input.WorkspacePath, input.WorkspaceID, types.WorkspaceModelType)
	if err != nil {
		return nil, err
	}

	identityID, err := toModelID(ctx, input.ManagedIdentityPath, input.ManagedIdentityID, types.ManagedIdentityModelType)
	if err != nil {
		return nil, err
	}

	serviceCatalog := getServiceCatalog(ctx)

	if err = serviceCatalog.ManagedIdentityService.RemoveManagedIdentityFromWorkspace(ctx, identityID, workspaceID); err != nil {
		return nil, err
	}

	workspace, err := serviceCatalog.WorkspaceService.GetWorkspaceByID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	payload := AssignManagedIdentityMutationPayload{ClientMutationID: input.ClientMutationID, Workspace: workspace, Problems: []Problem{}}
	return &AssignManagedIdentityMutationPayloadResolver{AssignManagedIdentityMutationPayload: payload}, nil
}

func createManagedIdentityCredentialsMutation(ctx context.Context,
	input *CreateManagedIdentityCredentialsInput,
) (*ManagedIdentityCredentialsMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := serviceCatalog.ManagedIdentityService.GetManagedIdentityByID(ctx, id)
	if err != nil {
		return nil, err
	}

	credentials, err := serviceCatalog.ManagedIdentityService.CreateCredentials(ctx, managedIdentity)
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

func moveManagedIdentityMutation(ctx context.Context, input *MoveManagedIdentityInput) (*ManagedIdentityMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	managedIdentityID, err := serviceCatalog.FetchModelID(ctx, input.ManagedIdentityID)
	if err != nil {
		return nil, err
	}

	// Get the new parent group's ID.
	newParentID, err := toModelID(ctx, input.NewParentPath, input.NewParentID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	managedIdentity, err := serviceCatalog.ManagedIdentityService.MoveManagedIdentity(ctx, &managedidentity.MoveManagedIdentityInput{
		ManagedIdentityID: managedIdentityID,
		NewGroupID:        newParentID,
	})
	if err != nil {
		return nil, err
	}

	payload := ManagedIdentityMutationPayload{
		ClientMutationID: input.ClientMutationID,
		ManagedIdentity:  managedIdentity,
		Problems:         []Problem{},
	}
	return &ManagedIdentityMutationPayloadResolver{ManagedIdentityMutationPayload: payload}, nil
}

// DEPRECATED: replaced by resolveManagedIdentityAllowedPrincipalIDs
func getManagedIdentityAllowedUserIDs(ctx context.Context, usernames []string) ([]string, error) {
	userService := getServiceCatalog(ctx).UserService
	response := []string{}

	for _, username := range usernames {
		user, err := userService.GetUserByTRN(ctx, types.UserModelType.BuildTRN(username))
		if err != nil {
			return nil, err
		}
		response = append(response, user.Metadata.ID)
	}

	return response, nil
}

// DEPRECATED: replaced by resolveManagedIdentityAllowedPrincipalIDs
func getManagedIdentityAllowedServiceAccountIDs(ctx context.Context, serviceAccountPaths []string) ([]string, error) {
	saService := getServiceCatalog(ctx).ServiceAccountService
	response := []string{}

	for _, path := range serviceAccountPaths {
		sa, err := saService.GetServiceAccountByTRN(ctx, types.ServiceAccountModelType.BuildTRN(path))
		if err != nil {
			return nil, err
		}
		response = append(response, sa.Metadata.ID)
	}

	return response, nil
}

// DEPRECATED: replaced by resolveManagedIdentityAllowedPrincipalIDs
func getManagedIdentityAllowedTeamIDs(ctx context.Context, teamNames []string) ([]string, error) {
	teamService := getServiceCatalog(ctx).TeamService
	response := []string{}

	for _, teamName := range teamNames {
		team, err := teamService.GetTeamByTRN(ctx, types.TeamModelType.BuildTRN(teamName))
		if err != nil {
			return nil, err
		}
		response = append(response, team.Metadata.ID)
	}

	return response, nil
}

func resolveManagedIdentityAllowedPrincipalIDs(ctx context.Context, ids []string) ([]string, error) {
	serviceCatalog := getServiceCatalog(ctx)
	response := []string{}

	for _, id := range ids {
		resolvedID, err := serviceCatalog.FetchModelID(ctx, id)
		if err != nil {
			return nil, err
		}
		response = append(response, resolvedID)
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
		return nil, errors.New("Wrong type")
	}

	return &managedIdentity, nil
}

func managedIdentityBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	managedIdentities, err := getServiceCatalog(ctx).ManagedIdentityService.GetManagedIdentitiesByIDs(ctx, ids)
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
		return nil, errors.New("Wrong type")
	}

	return &managedIdentityAccessRule, nil
}

func managedIdentityAccessRuleBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	rules, err := getServiceCatalog(ctx).ManagedIdentityService.GetManagedIdentityAccessRulesByIDs(ctx, ids)
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
