package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
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
	group, err := getGroupService(ctx).GetGroupByFullPath(ctx, args.FullPath)
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}
	if group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := getWorkspaceService(ctx).GetWorkspaceByFullPath(ctx, args.FullPath)
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}

	if ws == nil {
		return nil, nil
	}

	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
}
