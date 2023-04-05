package resolver

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Namespace Membership Query Resolvers */

// NamespaceMembershipConnectionQueryArgs are used to query a namespace membership connection
type NamespaceMembershipConnectionQueryArgs struct {
	ConnectionQueryArgs
	NamespacePath string
}

// NamespaceMembershipEdgeResolver resolves namespace membership edges
type NamespaceMembershipEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *NamespaceMembershipEdgeResolver) Cursor() (string, error) {
	namespaceMembership, ok := r.edge.Node.(models.NamespaceMembership)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&namespaceMembership)
	return *cursor, err
}

// Node returns a namespace membership node
func (r *NamespaceMembershipEdgeResolver) Node() (*NamespaceMembershipResolver, error) {
	namespaceMembership, ok := r.edge.Node.(models.NamespaceMembership)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &NamespaceMembershipResolver{namespaceMembership: &namespaceMembership}, nil
}

// NamespaceMembershipConnectionResolver resolves a namespace membership connection
type NamespaceMembershipConnectionResolver struct {
	connection Connection
}

// NewNamespaceMembershipConnectionResolver creates a new NamespaceMembershipConnectionResolver
func NewNamespaceMembershipConnectionResolver(ctx context.Context,
	input *namespacemembership.GetNamespaceMembershipsForSubjectInput,
) (*NamespaceMembershipConnectionResolver, error) {
	service := getNamespaceMembershipService(ctx)

	result, err := service.GetNamespaceMembershipsForSubject(ctx, input)
	if err != nil {
		return nil, err
	}

	namespaceMemberships := result.NamespaceMemberships

	// Create edges
	edges := make([]Edge, len(namespaceMemberships))
	for i, namespaceMembership := range namespaceMemberships {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: namespaceMembership}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(namespaceMemberships) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&namespaceMemberships[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&namespaceMemberships[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &NamespaceMembershipConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *NamespaceMembershipConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *NamespaceMembershipConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *NamespaceMembershipConnectionResolver) Edges() *[]*NamespaceMembershipEdgeResolver {
	resolvers := make([]*NamespaceMembershipEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &NamespaceMembershipEdgeResolver{edge: edge}
	}
	return &resolvers
}

// MemberResolver results the Member union type
type MemberResolver struct {
	result interface{}
}

// ToUser resolves user member types
func (r *MemberResolver) ToUser() (*UserResolver, bool) {
	res, ok := r.result.(*UserResolver)
	return res, ok
}

// ToServiceAccount resolves service account member types
func (r *MemberResolver) ToServiceAccount() (*ServiceAccountResolver, bool) {
	res, ok := r.result.(*ServiceAccountResolver)
	return res, ok
}

// ToTeam resolves team member types
func (r *MemberResolver) ToTeam() (*TeamResolver, bool) {
	res, ok := r.result.(*TeamResolver)
	return res, ok
}

// NamespaceMembershipResolver resolves a namespace membership resource
type NamespaceMembershipResolver struct {
	namespaceMembership *models.NamespaceMembership
}

// ID resolver
func (r *NamespaceMembershipResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.NamespaceMembershipType, r.namespaceMembership.Metadata.ID))
}

// ResourcePath resolver
func (r *NamespaceMembershipResolver) ResourcePath() string {
	return fmt.Sprintf("%s/%s", r.namespaceMembership.Namespace.Path, r.namespaceMembership.Metadata.ID)
}

// Member resolver
func (r *NamespaceMembershipResolver) Member(ctx context.Context) (*MemberResolver, error) {
	// Query for member based on type
	return makeMemberResolver(ctx, r.namespaceMembership.UserID,
		r.namespaceMembership.ServiceAccountID, r.namespaceMembership.TeamID)
}

// makeMemberResolver is also called by the activity event resolver module.
func makeMemberResolver(ctx context.Context, userID, serviceAccountID, teamID *string) (*MemberResolver, error) {
	if userID != nil {
		// Use resource loader to get user
		user, err := loadUser(ctx, *userID)
		if err != nil {
			return nil, err
		}
		return &MemberResolver{result: &UserResolver{user: user}}, nil
	}

	if serviceAccountID != nil {
		// Use resource loader to get service account
		serviceAccount, err := loadServiceAccount(ctx, *serviceAccountID)
		if err != nil {
			return nil, err
		}
		return &MemberResolver{result: &ServiceAccountResolver{serviceAccount: serviceAccount}}, nil
	}

	if teamID != nil {
		// Use resource loader to get team
		team, err := loadTeam(ctx, *teamID)
		if err != nil {
			return nil, err
		}
		return &MemberResolver{result: &TeamResolver{team: team}}, nil
	}

	return nil, errors.NewError(errors.EInvalid, "UserID, ServiceAccountID, or TeamID must be specified")
}

// Namespace resolver
func (r *NamespaceMembershipResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	// Query for member based on type
	if r.namespaceMembership.Namespace.GroupID != nil {
		group, err := loadGroup(ctx, *r.namespaceMembership.Namespace.GroupID)
		if err != nil {
			return nil, err
		}
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := loadWorkspace(ctx, *r.namespaceMembership.Namespace.WorkspaceID)
	if err != nil {
		return nil, err
	}
	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
}

// Role resolver
func (r *NamespaceMembershipResolver) Role(ctx context.Context) (*RoleResolver, error) {
	role, err := loadRole(ctx, r.namespaceMembership.RoleID)
	if err != nil {
		return nil, err
	}

	return &RoleResolver{role: role}, nil
}

// Metadata resolver
func (r *NamespaceMembershipResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.namespaceMembership.Metadata}
}

/* Namespace Membership Mutation Resolvers */

// NamespaceMembershipMutationPayload is the response payload for a namespace membership mutation
type NamespaceMembershipMutationPayload struct {
	ClientMutationID    *string
	NamespaceMembership *models.NamespaceMembership
	Problems            []Problem
}

// NamespaceMembershipMutationPayloadResolver resolves a NamespaceMembershipMutationPayload
type NamespaceMembershipMutationPayloadResolver struct {
	NamespaceMembershipMutationPayload
}

// Namespace field resolver
func (r *NamespaceMembershipMutationPayloadResolver) Namespace(ctx context.Context) (*NamespaceResolver, error) {
	if r.NamespaceMembershipMutationPayload.NamespaceMembership == nil {
		return nil, nil
	}
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, r.NamespaceMembership.Namespace.Path)
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}
	if group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, r.NamespaceMembership.Namespace.Path)
	if err != nil {
		return nil, err
	}
	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
}

// CreateNamespaceMembershipInput is the input for creating a new namespace membership
type CreateNamespaceMembershipInput struct {
	ClientMutationID *string
	Username         *string
	ServiceAccountID *string
	TeamName         *string
	Role             string
	NamespacePath    string
}

// UpdateNamespaceMembershipInput is the input for updating a namespace membership
type UpdateNamespaceMembershipInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
	Role             string
}

// DeleteNamespaceMembershipInput is the input for deleting a namespace membership
type DeleteNamespaceMembershipInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
}

func handleNamespaceMembershipMutationProblem(e error,
	clientMutationID *string,
) (*NamespaceMembershipMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := NamespaceMembershipMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &NamespaceMembershipMutationPayloadResolver{NamespaceMembershipMutationPayload: payload}, nil
}

func createNamespaceMembershipMutation(ctx context.Context,
	input *CreateNamespaceMembershipInput,
) (*NamespaceMembershipMutationPayloadResolver, error) {
	role, err := getRoleService(ctx).GetRoleByName(ctx, input.Role)
	if err != nil {
		return nil, err
	}

	createOptions := namespacemembership.CreateNamespaceMembershipInput{
		NamespacePath: input.NamespacePath,
		RoleID:        role.Metadata.ID,
	}

	if input.Username != nil {
		user, uErr := getUserService(ctx).GetUserByUsername(ctx, *input.Username)
		if uErr != nil {
			return nil, uErr
		}
		createOptions.User = user
	}

	if input.ServiceAccountID != nil {
		serviceAccount, sErr := getSAService(ctx).GetServiceAccountByID(ctx, gid.FromGlobalID(*input.ServiceAccountID))
		if sErr != nil {
			return nil, sErr
		}
		createOptions.ServiceAccount = serviceAccount
	}

	if input.TeamName != nil {
		team, tErr := getTeamService(ctx).GetTeamByName(ctx, *input.TeamName)
		if tErr != nil {
			return nil, tErr
		}
		createOptions.Team = team
	}

	namespaceMembership, nErr := getNamespaceMembershipService(ctx).CreateNamespaceMembership(ctx, &createOptions)
	if nErr != nil {
		return nil, nErr
	}

	payload := NamespaceMembershipMutationPayload{
		ClientMutationID:    input.ClientMutationID,
		NamespaceMembership: namespaceMembership,
		Problems:            []Problem{},
	}
	return &NamespaceMembershipMutationPayloadResolver{NamespaceMembershipMutationPayload: payload}, nil
}

func updateNamespaceMembershipMutation(ctx context.Context,
	input *UpdateNamespaceMembershipInput,
) (*NamespaceMembershipMutationPayloadResolver, error) {
	service := getNamespaceMembershipService(ctx)

	namespaceMembership, err := service.GetNamespaceMembershipByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		namespaceMembership.Metadata.Version = v
	}

	role, err := getRoleService(ctx).GetRoleByName(ctx, input.Role)
	if err != nil {
		return nil, err
	}

	namespaceMembership.RoleID = role.Metadata.ID

	namespaceMembership, err = service.UpdateNamespaceMembership(ctx, namespaceMembership)
	if err != nil {
		return nil, err
	}

	payload := NamespaceMembershipMutationPayload{
		ClientMutationID:    input.ClientMutationID,
		NamespaceMembership: namespaceMembership,
		Problems:            []Problem{},
	}
	return &NamespaceMembershipMutationPayloadResolver{NamespaceMembershipMutationPayload: payload}, nil
}

func deleteNamespaceMembershipMutation(ctx context.Context,
	input *DeleteNamespaceMembershipInput,
) (*NamespaceMembershipMutationPayloadResolver, error) {
	service := getNamespaceMembershipService(ctx)

	namespaceMembership, err := service.GetNamespaceMembershipByID(ctx, gid.FromGlobalID(input.ID))
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		namespaceMembership.Metadata.Version = v
	}

	if err = service.DeleteNamespaceMembership(ctx, namespaceMembership); err != nil {
		return nil, err
	}

	payload := NamespaceMembershipMutationPayload{
		ClientMutationID:    input.ClientMutationID,
		NamespaceMembership: namespaceMembership,
		Problems:            []Problem{},
	}
	return &NamespaceMembershipMutationPayloadResolver{NamespaceMembershipMutationPayload: payload}, nil
}

/* NamespaceMembership loader */

const namespaceMembershipLoaderKey = "namespaceMembership"

// RegisterNamespaceMembershipLoader registers a namespaceMembership loader function
func RegisterNamespaceMembershipLoader(collection *loader.Collection) {
	collection.Register(namespaceMembershipLoaderKey, namespaceMembershipBatchFunc)
}

func loadNamespaceMembership(ctx context.Context, id string) (*models.NamespaceMembership, error) {
	ldr, err := loader.Extract(ctx, namespaceMembershipLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	namespaceMembership, ok := data.(models.NamespaceMembership)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &namespaceMembership, nil
}

func namespaceMembershipBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	namespaceMemberships, err := getNamespaceMembershipService(ctx).GetNamespaceMembershipsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range namespaceMemberships {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
