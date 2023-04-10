package resolver

import (
	"context"
	"fmt"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Group Query Resolvers */

// GroupConnectionQueryArgs are used to query a group connection
type GroupConnectionQueryArgs struct {
	ConnectionQueryArgs
	ParentPath *string
}

// GroupQueryArgs are used to query a single group
type GroupQueryArgs struct {
	FullPath string
}

// GroupEdgeResolver resolves group edges
type GroupEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *GroupEdgeResolver) Cursor() (string, error) {
	group, ok := r.edge.Node.(models.Group)
	if !ok {
		return "", errors.NewError(errors.EInternal, "Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&group)
	return *cursor, err
}

// Node returns a group node
func (r *GroupEdgeResolver) Node() (*GroupResolver, error) {
	group, ok := r.edge.Node.(models.Group)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Failed to convert node type")
	}

	return &GroupResolver{group: &group}, nil
}

// GroupConnectionResolver resolves a group connection
type GroupConnectionResolver struct {
	connection Connection
}

// NewGroupConnectionResolver creates a new GroupConnectionResolver
func NewGroupConnectionResolver(ctx context.Context, input *group.GetGroupsInput) (*GroupConnectionResolver, error) {
	groupService := getGroupService(ctx)

	result, err := groupService.GetGroups(ctx, input)
	if err != nil {
		return nil, err
	}

	groups := result.Groups

	// Create edges
	edges := make([]Edge, len(groups))
	for i, group := range groups {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: group}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(groups) > 0 {
		var err error
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&groups[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&groups[len(edges)-1])
		if err != nil {
			return nil, err
		}
	}

	connection := Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}

	return &GroupConnectionResolver{connection: connection}, nil
}

// TotalCount returns the total result count for the connection
func (r *GroupConnectionResolver) TotalCount() int32 {
	return r.connection.TotalCount
}

// PageInfo returns the connection page information
func (r *GroupConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *GroupConnectionResolver) Edges() *[]*GroupEdgeResolver {
	resolvers := make([]*GroupEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &GroupEdgeResolver{edge: edge}
	}
	return &resolvers
}

// GroupResolver resolves a group resource
type GroupResolver struct {
	group *models.Group
}

// ID resolver
func (r *GroupResolver) ID() graphql.ID {
	return graphql.ID(gid.ToGlobalID(gid.GroupType, r.group.Metadata.ID))
}

// Name resolver
func (r *GroupResolver) Name() string {
	return r.group.Name
}

// Description resolver
func (r *GroupResolver) Description() string {
	return r.group.Description
}

// FullPath resolver
func (r *GroupResolver) FullPath() string {
	return r.group.FullPath
}

// Metadata resolver
func (r *GroupResolver) Metadata() *MetadataResolver {
	return &MetadataResolver{metadata: &r.group.Metadata}
}

// Parent resolver
func (r *GroupResolver) Parent(ctx context.Context) (*GroupResolver, error) {
	if r.group.ParentID == "" {
		return nil, nil
	}

	group, err := loadGroup(ctx, r.group.ParentID)
	if err != nil {
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

// DescendentGroups resolver
func (r *GroupResolver) DescendentGroups(ctx context.Context, args ConnectionQueryArgs) (*GroupConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := group.GetGroupsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		ParentGroup:       r.group,
	}

	if args.Sort != nil {
		sort := db.GroupSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewGroupConnectionResolver(ctx, &input)
}

// Workspaces resolvers
func (r *GroupResolver) Workspaces(ctx context.Context, args *ConnectionQueryArgs) (*WorkspaceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := workspace.GetWorkspacesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Group:             r.group,
	}

	if args.Sort != nil {
		sort := db.WorkspaceSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewWorkspaceConnectionResolver(ctx, &input)
}

// Memberships resolver
// The field is called "memberships", but most everything else is called "namespace memberships".
func (r *GroupResolver) Memberships(ctx context.Context) ([]*NamespaceMembershipResolver, error) {
	resolvers := []*NamespaceMembershipResolver{}

	result, err := getNamespaceMembershipService(ctx).GetNamespaceMembershipsForNamespace(ctx, r.group.FullPath)
	if err != nil {
		return nil, err
	}

	for _, v := range result {
		varCopy := v
		resolvers = append(resolvers, &NamespaceMembershipResolver{namespaceMembership: &varCopy})
	}

	return resolvers, nil
}

// Variables resolver
func (r *GroupResolver) Variables(ctx context.Context) ([]*NamespaceVariableResolver, error) {
	return getVariables(ctx, r.group.FullPath)
}

// GPGKeys resolver
func (r *GroupResolver) GPGKeys(ctx context.Context, args *GPGKeysConnectionQueryArgs) (*GPGKeyConnectionResolver, error) {
	input := &gpgkey.GetGPGKeysInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		NamespacePath: r.group.FullPath,
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewGPGKeyConnectionResolver(ctx, input)
}

// TerraformProviders resolver
func (r *GroupResolver) TerraformProviders(ctx context.Context, args *TerraformProviderConnectionQueryArgs) (*TerraformProviderConnectionResolver, error) {
	input := &providerregistry.GetProvidersInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Group:  r.group,
		Search: args.Search,
	}

	return NewTerraformProviderConnectionResolver(ctx, input)
}

// ServiceAccounts resolver
func (r *GroupResolver) ServiceAccounts(ctx context.Context, args *ServiceAccountsConnectionQueryArgs) (*ServiceAccountConnectionResolver, error) {
	input := &serviceaccount.GetServiceAccountsInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Search:        args.Search,
		NamespacePath: r.group.FullPath,
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewServiceAccountConnectionResolver(ctx, input)
}

// ManagedIdentities resolver
func (r *GroupResolver) ManagedIdentities(ctx context.Context, args *ManagedIdentityConnectionQueryArgs) (*ManagedIdentityConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := managedidentity.GetManagedIdentitiesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
		NamespacePath:     r.group.FullPath,
	}

	if args.Sort != nil {
		sort := db.ManagedIdentitySortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewManagedIdentityConnectionResolver(ctx, &input)
}

// Runners resolver
func (r *GroupResolver) Runners(ctx context.Context, args *RunnersConnectionQueryArgs) (*RunnerConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := runner.GetRunnersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		NamespacePath:     r.group.FullPath,
	}

	if args.Sort != nil {
		sort := db.RunnerSortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewRunnerConnectionResolver(ctx, &input)
}

// CreatedBy resolver
func (r *GroupResolver) CreatedBy() string {
	return r.group.CreatedBy
}

// ActivityEvents resolver
func (r *GroupResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	input, err := getActivityEventsInputFromQueryArgs(ctx, args)
	if err != nil {
		// error is already a Tharsis error
		return nil, err
	}

	// Need to filter to this group/namespace.
	input.NamespacePath = &r.group.FullPath

	return NewActivityEventConnectionResolver(ctx, input)
}

// VCSProviders resolver
func (r *GroupResolver) VCSProviders(ctx context.Context,
	args *VCSProviderConnectionQueryArgs,
) (*VCSProviderConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := vcs.GetVCSProvidersInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
		NamespacePath:     r.group.FullPath,
	}

	if args.Sort != nil {
		sort := db.VCSProviderSortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	return NewVCSProviderConnectionResolver(ctx, &input)
}

func groupQuery(ctx context.Context, args *GroupQueryArgs) (*GroupResolver, error) {
	groupService := getGroupService(ctx)

	group, err := groupService.GetGroupByFullPath(ctx, args.FullPath)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return &GroupResolver{group: group}, nil
}

func groupsQuery(ctx context.Context, args *GroupConnectionQueryArgs) (*GroupConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := group.GetGroupsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
	}

	if args.ParentPath != nil {
		parent, err := getGroupService(ctx).GetGroupByFullPath(ctx, *args.ParentPath)
		if err != nil {
			return nil, err
		}
		input.ParentGroup = parent
	}

	if args.Sort != nil {
		sort := db.GroupSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewGroupConnectionResolver(ctx, &input)
}

/* Group Mutation Resolvers */

// GroupMutationPayload is the response payload for a group mutation
type GroupMutationPayload struct {
	ClientMutationID *string
	Group            *models.Group
	Problems         []Problem
}

// GroupMutationPayloadResolver resolves a GroupMutationPayload
type GroupMutationPayloadResolver struct {
	GroupMutationPayload
}

// Group field resolver
func (r *GroupMutationPayloadResolver) Group() *GroupResolver {
	if r.GroupMutationPayload.Group == nil {
		return nil
	}
	return &GroupResolver{group: r.GroupMutationPayload.Group}
}

// CreateGroupInput contains the input for creating a new group
type CreateGroupInput struct {
	ClientMutationID *string
	Name             string
	ParentPath       *string
	Description      string
}

// UpdateGroupInput contains the input for updating a group
type UpdateGroupInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Description      *string
	GroupPath        *string
	ID               *string
}

// DeleteGroupInput contains the input for deleting a group
type DeleteGroupInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	GroupPath        *string
	ID               *string
}

// MigrateGroupInput contains the input for migrating a group
type MigrateGroupInput struct {
	ClientMutationID *string
	NewParentPath    *string
	GroupPath        string
}

func handleGroupMutationProblem(e error, clientMutationID *string) (*GroupMutationPayloadResolver, error) {
	problem, err := buildProblem(e)
	if err != nil {
		return nil, err
	}
	payload := GroupMutationPayload{ClientMutationID: clientMutationID, Problems: []Problem{*problem}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func createGroupMutation(ctx context.Context, input *CreateGroupInput) (*GroupMutationPayloadResolver, error) {
	groupCreateOptions := models.Group{Name: input.Name, Description: input.Description}
	groupService := getGroupService(ctx)

	if input.ParentPath != nil {
		parent, err := groupService.GetGroupByFullPath(ctx, *input.ParentPath)
		if err != nil {
			return nil, err
		}
		parentID := parent.Metadata.ID
		groupCreateOptions.ParentID = parentID
	}

	createdGroup, err := groupService.CreateGroup(ctx, &groupCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: createdGroup, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func updateGroupMutation(ctx context.Context, input *UpdateGroupInput) (*GroupMutationPayloadResolver, error) {
	groupService := getGroupService(ctx)

	var group *models.Group
	var err error
	switch {
	case input.GroupPath != nil:
		group, err = groupService.GetGroupByFullPath(ctx, *input.GroupPath)
	case input.ID != nil:
		group, err = groupService.GetGroupByID(ctx, gid.FromGlobalID(*input.ID))
	default:
		err = fmt.Errorf("must specify either GroupPath or ID")
	}
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, cErr := strconv.Atoi(input.Metadata.Version)
		if cErr != nil {
			return nil, cErr
		}

		group.Metadata.Version = v
	}

	// Update fields
	if input.Description != nil {
		group.Description = *input.Description
	}

	group, err = groupService.UpdateGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: group, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func deleteGroupMutation(ctx context.Context, input *DeleteGroupInput) (*GroupMutationPayloadResolver, error) {
	groupService := getGroupService(ctx)

	var groupToDelete *models.Group
	var err error
	switch {
	case input.GroupPath != nil:
		groupToDelete, err = groupService.GetGroupByFullPath(ctx, *input.GroupPath)
	case input.ID != nil:
		groupToDelete, err = groupService.GetGroupByID(ctx, gid.FromGlobalID(*input.ID))
	default:
		err = fmt.Errorf("must specify either GroupPath or ID")
	}
	if err != nil {
		return nil, err
	}

	// Check if resource version is specified
	if input.Metadata != nil {
		v, err := strconv.Atoi(input.Metadata.Version)
		if err != nil {
			return nil, err
		}

		groupToDelete.Metadata.Version = v
	}

	deleteOptions := group.DeleteGroupInput{
		Group: groupToDelete,
	}

	if input.Force != nil {
		deleteOptions.Force = *input.Force
	}

	if err := groupService.DeleteGroup(ctx, &deleteOptions); err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: groupToDelete, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func migrateGroupMutation(ctx context.Context, input *MigrateGroupInput) (*GroupMutationPayloadResolver, error) {
	groupService := getGroupService(ctx)

	group, err := groupService.GetGroupByFullPath(ctx, input.GroupPath)
	if err != nil {
		return nil, err
	}

	// If supplied, get the new parent group.
	var newParentID *string
	if input.NewParentPath != nil {
		newParent, iErr := groupService.GetGroupByFullPath(ctx, *input.NewParentPath)
		if iErr != nil {
			return nil, iErr
		}
		newParentID = &newParent.Metadata.ID
	}

	group, err = groupService.MigrateGroup(ctx, group.Metadata.ID, newParentID)
	if err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: group, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

/* Group loader */

const groupLoaderKey = "group"

// RegisterGroupLoader registers a group loader function
func RegisterGroupLoader(collection *loader.Collection) {
	collection.Register(groupLoaderKey, groupBatchFunc)
}

func loadGroup(ctx context.Context, id string) (*models.Group, error) {
	ldr, err := loader.Extract(ctx, groupLoaderKey)
	if err != nil {
		return nil, err
	}

	data, err := ldr.Load(ctx, dataloader.StringKey(id))()
	if err != nil {
		return nil, err
	}

	group, ok := data.(models.Group)
	if !ok {
		return nil, errors.NewError(errors.EInternal, "Wrong type")
	}

	return &group, nil
}

func groupBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	groups, err := getGroupService(ctx).GetGroupsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	// Build map of results
	batch := loader.DataBatch{}
	for _, result := range groups {
		batch[result.Metadata.ID] = result
	}

	return batch, nil
}
