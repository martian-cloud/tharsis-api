package resolver

import (
	"context"
	"sort"
	"strconv"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/role"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// RolesConnectionQueryArgs are used to query a role connection
type RolesConnectionQueryArgs struct {
	ConnectionQueryArgs
	Search *string
}

// RoleQueryArgs are used to query a single role
// DEPRECATED: use node query instead with a TRN
type RoleQueryArgs struct {
	Name string
}

// RoleEdgeResolver resolves role edges
type RoleEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *RoleEdgeResolver) Cursor() (string, error) {
	role, ok := r.edge.Node.(models.Role)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&role)
	return *cursor, err
}

// Node returns a role node
func (r *RoleEdgeResolver) Node() (*RoleResolver, error) {
	role, ok := r.edge.Node.(models.Role)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &RoleResolver{role: &role}, nil
}

// RoleConnectionResolver resolves a role connection
type RoleConnectionResolver struct {
	connection Connection
}

// NewRoleConnectionResolver creates a new RoleConnectionResolver
func NewRoleConnectionResolver(ctx context.Context, input *role.GetRolesInput) (*RoleConnectionResolver, error) {
	roleService := getServiceCatalog(ctx).RoleService

	result, err := roleService.GetRoles(ctx, input)
	if err != nil {
		return nil, err
	}

	roles := result.Roles

	// Create edges
	edges := make([]Edge, len(roles))
	for i, role := range roles {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: role}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(roles) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&roles[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&roles[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &RoleConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *RoleConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *RoleConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *RoleConnectionResolver) Edges() *[]*RoleEdgeResolver {
	resolvers := make([]*RoleEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &RoleEdgeResolver{edge: edge}
	}
	return &resolvers
}

// RoleResolver resolves a role resource
type RoleResolver struct {
	role *models.Role
}

// ID resolver
func (r *RoleResolver) ID() graphql.ID {
	return graphql.ID(r.role.GetGlobalID())
}

// Name resolver
func (r *RoleResolver) Name() string {
	return r.role.Name
}

// Description resolver
func (r *RoleResolver) Description() string {
	return r.role.Description
}

// Metadata resolver
func (r *RoleResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.role.Metadata}
}

// CreatedBy resolver
func (r *RoleResolver) CreatedBy() string {
	return r.role.CreatedBy
}

// Permissions resolver
func (r *RoleResolver) Permissions() []string {
	permSlice := []string{}
	for _, perm := range r.role.GetPermissions() {
		permSlice = append(permSlice, perm.String())
	}

	sort.Strings(permSlice)
	return permSlice
}

func availableRolePermissionsQuery(ctx context.Context) ([]string, error) {
	return getServiceCatalog(ctx).RoleService.GetAvailablePermissions(ctx)
}

// DEPRECATED: use node query instead with a TRN
func roleQuery(ctx context.Context, args *RoleQueryArgs) (*RoleResolver, error) {
	role, err := getServiceCatalog(ctx).RoleService.GetRoleByTRN(ctx, types.RoleModelType.BuildTRN(args.Name))
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}

		return nil, err
	}

	return &RoleResolver{role: role}, nil
}

func rolesQuery(ctx context.Context, args *RolesConnectionQueryArgs) (*RoleConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := role.GetRolesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
	}

	if args.Sort != nil {
		sort := db.RoleSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewRoleConnectionResolver(ctx, &input)
}

/* Role Mutation Resolvers */

// RoleMutationPayload is the response payload for a role mutation
type RoleMutationPayload struct {
	ClientMutationID *string
	Role             *models.Role
	Problems         []Problem
}

// RoleMutationPayloadResolver resolves a RoleMutationPayload
type RoleMutationPayloadResolver struct {
	RoleMutationPayload
}

// Role field resolver
func (r *RoleMutationPayloadResolver) Role() *RoleResolver {
	if r.RoleMutationPayload.Role == nil {
		return nil
	}
	return &RoleResolver{role: r.RoleMutationPayload.Role}
}

// CreateRoleInput contains the input for creating a new role
type CreateRoleInput struct {
	ClientMutationID *string
	Name             string
	Description      string
	Permissions      []string
}

// UpdateRoleInput contains the input for updating a role
type UpdateRoleInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	ID               string
	Description      *string
	Permissions      []string
}

// DeleteRoleInput contains the input for deleting a role
type DeleteRoleInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	ID               string
}

func handleRoleMutationProblem(e error, clientMutationID *string) (*RoleMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}

	payload := RoleMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &RoleMutationPayloadResolver{RoleMutationPayload: payload}, nil
}

func createRoleMutation(ctx context.Context, input *CreateRoleInput) (*RoleMutationPayloadResolver, error) {
	perms, err := models.ParsePermissions(input.Permissions)
	if err != nil {
		return nil, err
	}

	roleCreateOptions := &role.CreateRoleInput{
		Name:        input.Name,
		Description: input.Description,
		Permissions: perms,
	}

	createdRole, err := getServiceCatalog(ctx).RoleService.CreateRole(ctx, roleCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := RoleMutationPayload{ClientMutationID: input.ClientMutationID, Role: createdRole, Problems: []Problem{}}
	return &RoleMutationPayloadResolver{RoleMutationPayload: payload}, nil
}

func updateRoleMutation(ctx context.Context, input *UpdateRoleInput) (*RoleMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	gotRole, err := serviceCatalog.RoleService.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		gotRole.Metadata.Version = v
	}

	if input.Description != nil {
		gotRole.Description = *input.Description
	}

	if len(input.Permissions) > 0 {
		perms, pErr := models.ParsePermissions(input.Permissions)
		if pErr != nil {
			return nil, pErr
		}

		gotRole.SetPermissions(perms)
	}

	updatedRole, err := serviceCatalog.RoleService.UpdateRole(ctx, &role.UpdateRoleInput{Role: gotRole})
	if err != nil {
		return nil, err
	}

	payload := RoleMutationPayload{ClientMutationID: input.ClientMutationID, Role: updatedRole, Problems: []Problem{}}
	return &RoleMutationPayloadResolver{RoleMutationPayload: payload}, nil
}

func deleteRoleMutation(ctx context.Context, input *DeleteRoleInput) (*RoleMutationPayloadResolver, error) {
	serviceCatalog := getServiceCatalog(ctx)

	id, err := serviceCatalog.FetchModelID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	gotRole, err := serviceCatalog.RoleService.GetRoleByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		gotRole.Metadata.Version = v
	}

	deleteOptions := role.DeleteRoleInput{
		Role: gotRole,
	}

	if input.Force != nil {
		deleteOptions.Force = *input.Force
	}

	if err := serviceCatalog.RoleService.DeleteRole(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := RoleMutationPayload{ClientMutationID: input.ClientMutationID, Role: gotRole, Problems: []Problem{}}
	return &RoleMutationPayloadResolver{RoleMutationPayload: payload}, nil
}

/* Role loader */

const roleLoaderKey = "role"

// RegisterRoleLoader registers a role loader function
func RegisterRoleLoader(collection *loader.Collection) {
	collection.Register(roleLoaderKey, roleBatchFunc)
}

func loadRole(ctx context.Context, id string) (*models.Role, error) {
	ldr, err := loader.Extract(ctx, roleLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	vp, ok := data.(models.Role)
	if !ok {
		return nil, errors.New("Wrong type")
	}

	return &vp, nil
}

func roleBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	roles, err := getServiceCatalog(ctx).RoleService.GetRolesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range roles {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
