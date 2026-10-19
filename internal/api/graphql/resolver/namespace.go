package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	namespacesvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// NamespaceQueryArgs for querying a namespace by path
type NamespaceQueryArgs struct {
	FullPath string
}

// NamespaceResolver resolves the namespace union type
type NamespaceResolver struct {
	result interface{}
}

// ID resolver
func (r *NamespaceResolver) ID() (graphql.ID, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.ID(), nil
	case *WorkspaceResolver:
		return v.ID(), nil
	}
	return "", r.invalidNamespaceType()
}

// Name resolver
func (r *NamespaceResolver) Name() (string, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.Name(), nil
	case *WorkspaceResolver:
		return v.Name(), nil
	}
	return "", r.invalidNamespaceType()
}

// Description resolver
func (r *NamespaceResolver) Description() (string, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.Description(), nil
	case *WorkspaceResolver:
		return v.Description(), nil
	}
	return "", r.invalidNamespaceType()
}

// FullPath resolver
func (r *NamespaceResolver) FullPath() (string, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.FullPath(), nil
	case *WorkspaceResolver:
		return v.FullPath(), nil
	}
	return "", r.invalidNamespaceType()
}

// Metadata resolver
func (r *NamespaceResolver) Metadata() (*MetadataResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.Metadata(), nil
	case *WorkspaceResolver:
		return v.Metadata(), nil
	}
	return nil, r.invalidNamespaceType()
}

// Memberships resolver
// The field is called "memberships", but most everything else is called "namespace memberships".
func (r *NamespaceResolver) Memberships(ctx context.Context) ([]*NamespaceMembershipResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.Memberships(ctx)
	case *WorkspaceResolver:
		return v.Memberships(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// Variables resolver
func (r *NamespaceResolver) Variables(ctx context.Context) ([]*NamespaceVariableResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.Variables(ctx)
	case *WorkspaceResolver:
		return v.Variables(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// ServiceAccounts resolver
func (r *NamespaceResolver) ServiceAccounts(ctx context.Context, args *ServiceAccountsConnectionQueryArgs) (*ServiceAccountConnectionResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.ServiceAccounts(ctx, args)
	case *WorkspaceResolver:
		return v.ServiceAccounts(ctx, args)
	}
	return nil, r.invalidNamespaceType()
}

// ManagedIdentities resolver
func (r *NamespaceResolver) ManagedIdentities(ctx context.Context, args *ManagedIdentityConnectionQueryArgs) (*ManagedIdentityConnectionResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.ManagedIdentities(ctx, args)
	case *WorkspaceResolver:
		return v.ManagedIdentities(ctx, args)
	}
	return nil, r.invalidNamespaceType()
}

// ActivityEvents resolver
func (r *NamespaceResolver) ActivityEvents(ctx context.Context,
	args *ActivityEventConnectionQueryArgs,
) (*ActivityEventConnectionResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.ActivityEvents(ctx, args)
	case *WorkspaceResolver:
		return v.ActivityEvents(ctx, args)
	}
	return nil, r.invalidNamespaceType()
}

// TerraformProviderMirrors resolver
func (r *NamespaceResolver) TerraformProviderMirrors(ctx context.Context,
	args *TerraformProviderVersionMirrorConnectionQueryArgs,
) (*TerraformProviderVersionMirrorConnectionResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.TerraformProviderMirrors(ctx, args)
	case *WorkspaceResolver:
		return v.TerraformProviderMirrors(ctx, args)
	}
	return nil, r.invalidNamespaceType()
}

// ToGroup resolves the group namespace type
func (r *NamespaceResolver) ToGroup() (*GroupResolver, bool) {
	res, ok := r.result.(*GroupResolver)
	return res, ok
}

// ToWorkspace resolves the workspace namespace type
func (r *NamespaceResolver) ToWorkspace() (*WorkspaceResolver, bool) {
	res, ok := r.result.(*WorkspaceResolver)
	return res, ok
}

func (r *NamespaceResolver) invalidNamespaceType() error {
	return fmt.Errorf("invalid namespace type: %T", r.result)
}

func namespaceQuery(ctx context.Context, args *NamespaceQueryArgs) (*NamespaceResolver, error) {
	ns, err := getServiceCatalog(ctx).NamespaceService.GetNamespace(ctx, args.FullPath)
	if err != nil {
		if errors.ErrorCode(err) == errors.ENotFound {
			return nil, nil
		}
		return nil, err
	}

	return newNamespaceResolver(ns), nil
}

// NamespaceRunnerTagsInput represents the settings for runner tags.
type NamespaceRunnerTagsInput struct {
	Tags    *[]string
	Inherit bool
}

// Validate returns an error if the input is not valid.
func (r *NamespaceRunnerTagsInput) Validate() error {

	// Tags and Inherit are mutually exclusive.
	if r != nil && r.Tags != nil && r.Inherit {
		return errors.New("cannot specify both tags and inherit", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// RunnerTags resolver
func (r *NamespaceResolver) RunnerTags(ctx context.Context) (*namespace.RunnerTagsSetting, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.RunnerTags(ctx)
	case *WorkspaceResolver:
		return v.RunnerTags(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// NamespaceDriftDetectionEnabledInput represents the settings for enabling drift detection
type NamespaceDriftDetectionEnabledInput struct {
	Enabled *bool
	Inherit bool
}

// Validate returns an error if the input is not valid.
func (r *NamespaceDriftDetectionEnabledInput) Validate() error {

	// Tags and Inherit are mutually exclusive.
	if r != nil && r.Enabled != nil && r.Inherit {
		return errors.New("cannot specify both enabled and inherit", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// NamespaceProviderMirrorEnabledInput represents the settings for enabling provider mirror
type NamespaceProviderMirrorEnabledInput struct {
	Enabled *bool
	Inherit bool
}

// Validate returns an error if the input is not valid.
func (r *NamespaceProviderMirrorEnabledInput) Validate() error {
	if r != nil && r.Enabled != nil && r.Inherit {
		return errors.New("cannot specify both enabled and inherit", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}

// NamespaceOutputVisibilityInput represents the settings for output visibility
type NamespaceOutputVisibilityInput struct {
	Visibility *models.NamespaceOutputVisibilityLevel
	Inherit    bool
}

// Validate returns an error if the input is not valid.
func (r *NamespaceOutputVisibilityInput) Validate() error {
	if r != nil && r.Visibility != nil && r.Inherit {
		return errors.New("cannot specify both visibility and inherit", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}

// DriftDetectionEnabled resolver
func (r *NamespaceResolver) DriftDetectionEnabled(ctx context.Context) (*namespace.DriftDetectionEnabledSetting, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.DriftDetectionEnabled(ctx)
	case *WorkspaceResolver:
		return v.DriftDetectionEnabled(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// ProviderMirrorEnabled resolver
func (r *NamespaceResolver) ProviderMirrorEnabled(ctx context.Context) (*namespace.ProviderMirrorEnabledSetting, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.ProviderMirrorEnabled(ctx)
	case *WorkspaceResolver:
		return v.ProviderMirrorEnabled(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// OutputVisibility resolver
func (r *NamespaceResolver) OutputVisibility(ctx context.Context) (*namespace.OutputVisibilitySetting, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.OutputVisibility(ctx)
	case *WorkspaceResolver:
		return v.OutputVisibility(ctx)
	}
	return nil, r.invalidNamespaceType()
}

// EffectiveCleanupPolicies resolver
func (r *NamespaceResolver) EffectiveCleanupPolicies(ctx context.Context) ([]*CleanupPolicyResolver, error) {
	switch v := r.result.(type) {
	case *GroupResolver:
		return v.EffectiveCleanupPolicies(ctx)
	case *WorkspaceResolver:
		return v.EffectiveCleanupPolicies(ctx)
	}
	return nil, r.invalidNamespaceType()
}

/* Namespace Connection */

// NamespaceConnectionQueryArgs are used to query a namespace connection
type NamespaceConnectionQueryArgs struct {
	ConnectionQueryArgs
	ParentID      *string
	Search        *string
	Sort          *string
	TopLevel      *bool
	IncludeNested *bool
}

// NamespaceEdgeResolver resolves namespace edges
type NamespaceEdgeResolver struct {
	edge Edge
}

// Cursor returns an opaque cursor
func (r *NamespaceEdgeResolver) Cursor() (string, error) {
	ns, ok := r.edge.Node.(*db.Namespace)
	if !ok {
		return "", errors.New("Failed to convert node type")
	}
	cursor, err := r.edge.CursorFunc(namespaceModel(ns))
	if err != nil {
		return "", err
	}
	return *cursor, nil
}

// Node returns a namespace node
func (r *NamespaceEdgeResolver) Node() (*NamespaceResolver, error) {
	ns, ok := r.edge.Node.(*db.Namespace)
	if !ok {
		return nil, errors.New("Failed to convert node type")
	}
	return newNamespaceResolver(ns), nil
}

// NamespaceConnectionResolver resolves a namespace connection
type NamespaceConnectionResolver struct {
	connection Connection
}

// TotalCount returns the total number of namespaces matching the query
func (r *NamespaceConnectionResolver) TotalCount(ctx context.Context) (int32, error) {
	return r.connection.TotalCount(ctx)
}

// PageInfo returns the connection page information
func (r *NamespaceConnectionResolver) PageInfo() *PageInfoResolver {
	return &PageInfoResolver{pageInfo: r.connection.PageInfo}
}

// Edges returns the connection edges
func (r *NamespaceConnectionResolver) Edges() *[]*NamespaceEdgeResolver {
	resolvers := make([]*NamespaceEdgeResolver, len(r.connection.Edges))
	for i, edge := range r.connection.Edges {
		resolvers[i] = &NamespaceEdgeResolver{edge: edge}
	}
	return &resolvers
}

// NewNamespaceConnectionResolver creates a NamespaceConnectionResolver from a service input.
func NewNamespaceConnectionResolver(ctx context.Context, input *namespacesvc.GetNamespacesInput) (*NamespaceConnectionResolver, error) {
	result, err := getServiceCatalog(ctx).NamespaceService.GetNamespaces(ctx, input)
	if err != nil {
		return nil, err
	}

	edges := make([]Edge, len(result.Namespaces))
	for i := range result.Namespaces {
		edges[i] = Edge{CursorFunc: result.PageInfo.Cursor, Node: &result.Namespaces[i]}
	}

	pageInfo := PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
	}

	if len(result.Namespaces) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(namespaceModel(&result.Namespaces[0]))
		if err != nil {
			return nil, err
		}
		pageInfo.EndCursor, err = result.PageInfo.Cursor(namespaceModel(&result.Namespaces[len(edges)-1]))
		if err != nil {
			return nil, err
		}
	}

	return &NamespaceConnectionResolver{connection: Connection{
		TotalCount: result.PageInfo.TotalCount,
		PageInfo:   pageInfo,
		Edges:      edges,
	}}, nil
}

// newNamespaceResolver wraps whichever of the namespace's group or workspace is set.
func newNamespaceResolver(ns *db.Namespace) *NamespaceResolver {
	if ns.Group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: ns.Group}}
	}
	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ns.Workspace}}
}

// namespaceModel returns whichever of the namespace's group or workspace is set.
func namespaceModel(ns *db.Namespace) models.Model {
	if ns.Group != nil {
		return ns.Group
	}
	return ns.Workspace
}

func namespacesQuery(ctx context.Context, args *NamespaceConnectionQueryArgs) (*NamespaceConnectionResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	topLevel := args.TopLevel != nil && *args.TopLevel
	if topLevel && args.ParentID != nil {
		return nil, errors.New("parentId and topLevel cannot both be specified", errors.WithErrorCode(errors.EInvalid))
	}

	includeNested := args.IncludeNested != nil && *args.IncludeNested
	if topLevel && includeNested {
		return nil, errors.New("topLevel and includeNested cannot both be specified", errors.WithErrorCode(errors.EInvalid))
	}

	var parentGroupID *string
	if args.ParentID != nil {
		id, err := getServiceCatalog(ctx).FetchModelID(ctx, *args.ParentID)
		if err != nil {
			return nil, err
		}
		parentGroupID = &id
	}

	input := newGetNamespacesInput(args, parentGroupID)
	input.TopLevel = topLevel

	return NewNamespaceConnectionResolver(ctx, input)
}

// newGetNamespacesInput builds the service input shared by the namespaces query and
// GroupResolver.Namespaces. parentGroupID must already be resolved to an internal ID.
func newGetNamespacesInput(args *NamespaceConnectionQueryArgs, parentGroupID *string) *namespacesvc.GetNamespacesInput {
	input := &namespacesvc.GetNamespacesInput{
		PaginationOptions: &pagination.Options{First: args.First, Last: args.Last, After: args.After, Before: args.Before},
		ParentGroupID:     parentGroupID,
		IncludeNested:     args.IncludeNested != nil && *args.IncludeNested,
		Search:            args.Search,
	}

	if args.Sort != nil {
		sf := db.NamespaceSortableField(*args.Sort)
		input.Sort = &sf
	}

	return input
}
