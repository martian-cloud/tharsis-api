package resolver

import (
	"context"
	"strconv"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/federatedregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"

	"github.com/aws/smithy-go/ptr"
	"github.com/graph-gophers/dataloader"
	graphql "github.com/graph-gophers/graphql-go"
)

/* Group Query Resolvers */

// GroupConnectionQueryArgs are used to query a group connection
type GroupConnectionQueryArgs struct {
	ConnectionQueryArgs
	ParentPath *string // DEPRECATED: use ParentID instead with a TRN
	ParentID   *string
	Search     *string
	Favorites  *bool
}

// GroupQueryArgs are used to query a single group
// DEPRECATED: should use node query instead with a TRN
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
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(&group)
	return *cursor, err
}

// Node returns a group node
func (r *GroupEdgeResolver) Node() (*GroupResolver, error) {
	group, ok := r.edge.Node.(models.Group)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}

	return &GroupResolver{group: &group}, nil
}

// GroupConnectionResolver resolves a group connection
type GroupConnectionResolver struct {
	connection Connection
}

// NewGroupConnectionResolver creates a new GroupConnectionResolver
func NewGroupConnectionResolver(ctx context.Context, input *group.GetGroupsInput) (*GroupConnectionResolver, error) {
	groupService := getServiceCatalog(ctx).GroupService

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
	return graphql.ID(r.group.GetGlobalID())
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
		ParentGroupID:     &r.group.Metadata.ID,
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
		GroupID:           &r.group.Metadata.ID,
	}

	if args.Sort != nil {
		sort := db.WorkspaceSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewWorkspaceConnectionResolver(ctx, &input)
}

// Runs resolver
func (r *GroupResolver) Runs(ctx context.Context, args *RunConnectionQueryArgs) (*RunConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := run.GetRunsInput{
		PaginationOptions:   &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Group:               r.group,
		WorkspaceAssessment: args.WorkspaceAssessment,
		IncludeNestedRuns:   args.IncludeNestedRuns,
	}

	if args.Sort != nil {
		sort := db.RunSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewRunConnectionResolver(ctx, &input)
}

// Memberships resolver
// The field is called "memberships", but most everything else is called "namespace memberships".
func (r *GroupResolver) Memberships(ctx context.Context) ([]*NamespaceMembershipResolver, error) {
	resolvers := []*NamespaceMembershipResolver{}

	result, err := getServiceCatalog(ctx).NamespaceMembershipService.GetNamespaceMembershipsForNamespace(ctx, r.group.FullPath)
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

	if args.Sort != nil {
		sort := db.GPGKeySortableField(*args.Sort)
		input.Sort = &sort
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

// TerraformModules resolver
func (r *GroupResolver) TerraformModules(ctx context.Context, args *TerraformModuleConnectionQueryArgs) (*TerraformModuleConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &moduleregistry.GetModulesInput{
		PaginationOptions: &pagination.Options{
			First:  args.First,
			Last:   args.Last,
			Before: args.Before,
			After:  args.After,
		},
		Group:  r.group,
		Search: args.Search,
	}

	if args.IncludeInherited != nil && *args.IncludeInherited {
		input.IncludeInherited = true
	}

	if args.Sort != nil {
		sort := db.TerraformModuleSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformModuleConnectionResolver(ctx, input)
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

	if args.Sort != nil {
		sort := db.ServiceAccountSortableField(*args.Sort)
		input.Sort = &sort
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
		NamespacePath:     &r.group.FullPath,
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

// RunnerTags resolver
func (r *GroupResolver) RunnerTags(ctx context.Context) (*namespace.RunnerTagsSetting, error) {
	return getServiceCatalog(ctx).GroupService.GetRunnerTagsSetting(ctx, r.group)
}

// DriftDetectionEnabled resolver
func (r *GroupResolver) DriftDetectionEnabled(ctx context.Context) (*namespace.DriftDetectionEnabledSetting, error) {
	return getServiceCatalog(ctx).GroupService.GetDriftDetectionEnabledSetting(ctx, r.group)
}

// ProviderMirrorEnabled resolver
func (r *GroupResolver) ProviderMirrorEnabled(ctx context.Context) (*namespace.ProviderMirrorEnabledSetting, error) {
	return getServiceCatalog(ctx).GroupService.GetProviderMirrorEnabledSetting(ctx, r.group)
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

// TerraformProviderMirrors resolver
func (r *GroupResolver) TerraformProviderMirrors(
	ctx context.Context,
	args *TerraformProviderVersionMirrorConnectionQueryArgs,
) (*TerraformProviderVersionMirrorConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := &providermirror.GetProviderVersionMirrorsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		NamespacePath:     r.group.FullPath,
		Search:            args.Search,
	}

	if args.Sort != nil {
		sort := db.TerraformProviderVersionMirrorSortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewTerraformProviderVersionMirrorConnectionResolver(ctx, input)
}

// FederatedRegistries resolver
func (r GroupResolver) FederatedRegistries(ctx context.Context,
	args *FederatedRegistryConnectionQueryArgs,
) (*FederatedRegistryConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	input := federatedregistry.GetFederatedRegistriesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		GroupPath:         &r.group.FullPath,
	}

	if args.Sort != nil {
		sort := db.FederatedRegistrySortableField(*args.Sort)
		input.Sort = &sort
	}

	return NewFederatedRegistryConnectionResolver(ctx, &input)
}

// DEPRECATED: should use node query instead since it supports both TRN and GID
func groupQuery(ctx context.Context, args *GroupQueryArgs) (*GroupResolver, error) {
	group, err := getServiceCatalog(ctx).GroupService.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(args.FullPath))
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

	// If parent-path is not nil and empty, set RootOnly in the input struct.
	input := group.GetGroupsInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		Search:            args.Search,
		RootOnly:          (args.ParentPath != nil) && (*args.ParentPath == ""),
	}

	if args.ParentID != nil && args.ParentPath != nil {
		return nil, errors.New("must specify either parentId or parentPath, not both", errors.WithErrorCode(errors.EInvalid))
	}

	if args.ParentID != nil {
		parentID, err := getServiceCatalog(ctx).FetchModelID(ctx, *args.ParentID)
		if err != nil {
			return nil, err
		}
		input.ParentGroupID = &parentID
	} else if (args.ParentPath != nil) && (*args.ParentPath != "") {
		parent, err := getServiceCatalog(ctx).GroupService.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(*args.ParentPath))
		if err != nil {
			return nil, err
		}
		input.ParentGroupID = &parent.Metadata.ID
	}

	if args.Sort != nil {
		sort := db.GroupSortableField(*args.Sort)
		input.Sort = &sort
	}

	if args.Favorites != nil && *args.Favorites {
		input.Favorites = args.Favorites
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
	ClientMutationID      *string
	Name                  string
	ParentPath            *string // DEPRECATED: use ParentID instead with a TRN
	ParentID              *string
	RunnerTags            *NamespaceRunnerTagsInput
	DriftDetectionEnabled *NamespaceDriftDetectionEnabledInput
	ProviderMirrorEnabled *NamespaceProviderMirrorEnabledInput
	Description           string
}

// UpdateGroupInput contains the input for updating a group
type UpdateGroupInput struct {
	ClientMutationID      *string
	Metadata              *MetadataInput
	Description           *string
	GroupPath             *string // DEPRECATED: use ID instead with a TRN
	ID                    *string
	RunnerTags            *NamespaceRunnerTagsInput
	DriftDetectionEnabled *NamespaceDriftDetectionEnabledInput
	ProviderMirrorEnabled *NamespaceProviderMirrorEnabledInput
}

// DeleteGroupInput contains the input for deleting a group
type DeleteGroupInput struct {
	ClientMutationID *string
	Metadata         *MetadataInput
	Force            *bool
	GroupPath        *string // DEPRECATED: use ID instead with a TRN
	ID               *string
}

// MigrateGroupInput contains the input for migrating a group
type MigrateGroupInput struct {
	ClientMutationID *string
	NewParentPath    *string // DEPRECATED: use NewParentID instead with a TRN
	GroupPath        *string // DEPRECATED: use GroupID instead with a TRN
	NewParentID      *string
	GroupID          *string
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
	groupCreateOptions := models.Group{
		Name:        input.Name,
		Description: input.Description,
	}

	if input.RunnerTags != nil {
		if err := input.RunnerTags.Validate(); err != nil {
			return nil, err
		}

		if input.RunnerTags.Tags != nil {
			groupCreateOptions.RunnerTags = *input.RunnerTags.Tags
		}
	}

	if input.DriftDetectionEnabled != nil {
		if err := input.DriftDetectionEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.DriftDetectionEnabled.Enabled != nil {
			groupCreateOptions.EnableDriftDetection = input.DriftDetectionEnabled.Enabled
		}
	}

	if input.ProviderMirrorEnabled != nil {
		if err := input.ProviderMirrorEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.ProviderMirrorEnabled.Enabled != nil {
			groupCreateOptions.EnableProviderMirror = input.ProviderMirrorEnabled.Enabled
		}
	}

	if input.ParentID != nil && input.ParentPath != nil {
		return nil, errors.New("must specify either parentId or parentPath, not both", errors.WithErrorCode(errors.EInvalid))
	}

	var valueToResolve *string
	if input.ParentID != nil {
		valueToResolve = input.ParentID
	} else if input.ParentPath != nil {
		valueToResolve = ptr.String(types.GroupModelType.BuildTRN(*input.ParentPath))
	}

	serviceCatalog := getServiceCatalog(ctx)

	if valueToResolve != nil {
		parentID, err := serviceCatalog.FetchModelID(ctx, *valueToResolve)
		if err != nil {
			return nil, err
		}
		groupCreateOptions.ParentID = parentID
	}

	createdGroup, err := serviceCatalog.GroupService.CreateGroup(ctx, &groupCreateOptions)
	if err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: createdGroup, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func updateGroupMutation(ctx context.Context, input *UpdateGroupInput) (*GroupMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.ID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	groupService := getServiceCatalog(ctx).GroupService

	group, err := groupService.GetGroupByID(ctx, groupID)
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

	if input.RunnerTags != nil {
		if rErr := input.RunnerTags.Validate(); rErr != nil {
			return nil, rErr
		}

		if input.RunnerTags.Tags != nil {
			group.RunnerTags = *input.RunnerTags.Tags
		}

		if input.RunnerTags.Inherit {
			group.RunnerTags = nil
		}
	}

	if input.DriftDetectionEnabled != nil {
		if err = input.DriftDetectionEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.DriftDetectionEnabled.Enabled != nil {
			group.EnableDriftDetection = input.DriftDetectionEnabled.Enabled
		}

		if input.DriftDetectionEnabled.Inherit {
			group.EnableDriftDetection = nil
		}
	}

	if input.ProviderMirrorEnabled != nil {
		if err = input.ProviderMirrorEnabled.Validate(); err != nil {
			return nil, err
		}

		if input.ProviderMirrorEnabled.Enabled != nil {
			group.EnableProviderMirror = input.ProviderMirrorEnabled.Enabled
		}

		if input.ProviderMirrorEnabled.Inherit {
			group.EnableProviderMirror = nil
		}
	}

	group, err = groupService.UpdateGroup(ctx, group)
	if err != nil {
		return nil, err
	}

	payload := GroupMutationPayload{ClientMutationID: input.ClientMutationID, Group: group, Problems: []Problem{}}
	return &GroupMutationPayloadResolver{GroupMutationPayload: payload}, nil
}

func deleteGroupMutation(ctx context.Context, input *DeleteGroupInput) (*GroupMutationPayloadResolver, error) {
	groupID, err := toModelID(ctx, input.GroupPath, input.ID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	groupService := getServiceCatalog(ctx).GroupService

	groupToDelete, err := groupService.GetGroupByID(ctx, groupID)
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
	groupID, err := toModelID(ctx, input.GroupPath, input.GroupID, types.GroupModelType)
	if err != nil {
		return nil, err
	}

	groupService := getServiceCatalog(ctx).GroupService

	group, err := groupService.GetGroupByID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	// If supplied, get the new parent group.
	var newParentID *string
	if input.NewParentID != nil || input.NewParentPath != nil {
		id, err := toModelID(ctx, input.NewParentPath, input.NewParentID, types.GroupModelType)
		if err != nil {
			// Function will return generic errors
			return nil, errors.Wrap(err, "failed to resolve new parent id")
		}

		newParentID = &id
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
		return nil, errors.New("Wrong type")
	}

	return &group, nil
}

func groupBatchFunc(ctx context.Context, ids []string) (loader.DataBatch, error) {
	groups, err := getServiceCatalog(ctx).GroupService.GetGroupsByIDs(ctx, ids)
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
