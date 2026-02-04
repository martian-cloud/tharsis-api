package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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
	serviceCatalog := getServiceCatalog(ctx)

	group, err := serviceCatalog.GroupService.GetGroupByTRN(ctx, types.GroupModelType.BuildTRN(args.FullPath))
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}
	if group != nil {
		return &NamespaceResolver{result: &GroupResolver{group: group}}, nil
	}

	ws, err := serviceCatalog.WorkspaceService.GetWorkspaceByTRN(ctx, types.WorkspaceModelType.BuildTRN(args.FullPath))
	if err != nil && errors.ErrorCode(err) != errors.ENotFound {
		return nil, err
	}

	if ws == nil {
		return nil, nil
	}

	return &NamespaceResolver{result: &WorkspaceResolver{workspace: ws}}, nil
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
