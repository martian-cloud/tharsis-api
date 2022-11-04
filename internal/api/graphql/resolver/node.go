package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
)

// NodeResolver resolves a node type
type NodeResolver struct {
	result interface{}
}

type idGetter interface {
	ID() graphql.ID
}

// ID resolver
func (r *NodeResolver) ID() (graphql.ID, error) {
	node, ok := r.result.(idGetter)
	if !ok {
		return "", fmt.Errorf("invalid node resolver")
	}
	return node.ID(), nil
}

// ToApply resolver
func (r *NodeResolver) ToApply() (*ApplyResolver, bool) {
	res, ok := r.result.(*ApplyResolver)
	return res, ok
}

// ToConfigurationVersion resolver
func (r *NodeResolver) ToConfigurationVersion() (*ConfigurationVersionResolver, bool) {
	res, ok := r.result.(*ConfigurationVersionResolver)
	return res, ok
}

// ToGroup resolver
func (r *NodeResolver) ToGroup() (*GroupResolver, bool) {
	res, ok := r.result.(*GroupResolver)
	return res, ok
}

// ToJob resolver
func (r *NodeResolver) ToJob() (*JobResolver, bool) {
	res, ok := r.result.(*JobResolver)
	return res, ok
}

// TODO: ToJobLogDescriptor resolver

// ToManagedIdentity resolver
func (r *NodeResolver) ToManagedIdentity() (*ManagedIdentityResolver, bool) {
	res, ok := r.result.(*ManagedIdentityResolver)
	return res, ok
}

// ToManagedIdentityAccessRule resolver
func (r *NodeResolver) ToManagedIdentityAccessRule() (*ManagedIdentityAccessRuleResolver, bool) {
	res, ok := r.result.(*ManagedIdentityAccessRuleResolver)
	return res, ok
}

// ToNamespaceMembership resolver
func (r *NodeResolver) ToNamespaceMembership() (*NamespaceMembershipResolver, bool) {
	res, ok := r.result.(*NamespaceMembershipResolver)
	return res, ok
}

// ToPlan resolver
func (r *NodeResolver) ToPlan() (*PlanResolver, bool) {
	res, ok := r.result.(*PlanResolver)
	return res, ok
}

// ToRun resolver
func (r *NodeResolver) ToRun() (*RunResolver, bool) {
	res, ok := r.result.(*RunResolver)
	return res, ok
}

// TODO: ToRunner resolver

// ToServiceAccount resolver
func (r *NodeResolver) ToServiceAccount() (*ServiceAccountResolver, bool) {
	res, ok := r.result.(*ServiceAccountResolver)
	return res, ok
}

// ToStateVersion resolver
func (r *NodeResolver) ToStateVersion() (*StateVersionResolver, bool) {
	res, ok := r.result.(*StateVersionResolver)
	return res, ok
}

// ToStateVersionOutput resolver
func (r *NodeResolver) ToStateVersionOutput() (*StateVersionOutputResolver, bool) {
	res, ok := r.result.(*StateVersionOutputResolver)
	return res, ok
}

// ToUser resolver
func (r *NodeResolver) ToUser() (*UserResolver, bool) {
	res, ok := r.result.(*UserResolver)
	return res, ok
}

// ToNamespaceVariable resolver
func (r *NodeResolver) ToNamespaceVariable() (*NamespaceVariableResolver, bool) {
	res, ok := r.result.(*NamespaceVariableResolver)
	return res, ok
}

// ToWorkspace resolver
func (r *NodeResolver) ToWorkspace() (*WorkspaceResolver, bool) {
	res, ok := r.result.(*WorkspaceResolver)
	return res, ok
}

// ToTeam resolver
func (r *NodeResolver) ToTeam() (*TeamResolver, bool) {
	res, ok := r.result.(*TeamResolver)
	return res, ok
}

// ToTerraformProvider resolver
func (r *NodeResolver) ToTerraformProvider() (*TerraformProviderResolver, bool) {
	res, ok := r.result.(*TerraformProviderResolver)
	return res, ok
}

// ToTerraformProviderVersion resolver
func (r *NodeResolver) ToTerraformProviderVersion() (*TerraformProviderVersionResolver, bool) {
	res, ok := r.result.(*TerraformProviderVersionResolver)
	return res, ok
}

// ToTerraformProviderPlatform resolver
func (r *NodeResolver) ToTerraformProviderPlatform() (*TerraformProviderPlatformResolver, bool) {
	res, ok := r.result.(*TerraformProviderPlatformResolver)
	return res, ok
}

// ToGPGKey resolver
func (r *NodeResolver) ToGPGKey() (*GPGKeyResolver, bool) {
	res, ok := r.result.(*GPGKeyResolver)
	return res, ok
}

// ToActivityEvent resolver
func (r *NodeResolver) ToActivityEvent() (*ActivityEventResolver, bool) {
	res, ok := r.result.(*ActivityEventResolver)
	return res, ok
}

func node(ctx context.Context, globalID string) (*NodeResolver, error) {
	parsedGlobalID, err := gid.ParseGlobalID(globalID)
	if err != nil {
		return nil, err
	}

	var resolver interface{}

	switch parsedGlobalID.Type {
	case gid.ApplyType:
		apply, err := getRunService(ctx).GetApply(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &ApplyResolver{apply: apply}
	case gid.ConfigurationVersionType:
		cv, err := getWorkspaceService(ctx).GetConfigurationVersion(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &ConfigurationVersionResolver{configurationVersion: cv}
	case gid.GroupType:
		group, err := getGroupService(ctx).GetGroupByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &GroupResolver{group: group}
	case gid.JobType:
		job, err := getJobService(ctx).GetJob(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &JobResolver{job: job}
	// TODO: JobLogDescriptorType
	case gid.ManagedIdentityType:
		managedIdentity, err := getManagedIdentityService(ctx).GetManagedIdentityByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &ManagedIdentityResolver{managedIdentity: managedIdentity}
	case gid.ManagedIdentityAccessRuleType:
		rule, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRule(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &ManagedIdentityAccessRuleResolver{rule: rule}
	case gid.NamespaceMembershipType:
		namespaceMembership, err := getNamespaceMembershipService(ctx).GetNamespaceMembershipByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &NamespaceMembershipResolver{namespaceMembership: namespaceMembership}
	case gid.PlanType:
		plan, err := getRunService(ctx).GetPlan(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &PlanResolver{plan: plan}
	case gid.RunType:
		run, err := getRunService(ctx).GetRun(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &RunResolver{run: run}
	// TODO: RunnerType
	case gid.ServiceAccountType:
		serviceAccount, err := getSAService(ctx).GetServiceAccountByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &ServiceAccountResolver{serviceAccount: serviceAccount}
	case gid.StateVersionType:
		stateVersion, err := getWorkspaceService(ctx).GetStateVersion(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &StateVersionResolver{stateVersion: stateVersion}
	case gid.StateVersionOutputType:
		stateVersionOutput, err := getStateVersionOutputs(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = stateVersionOutput
	case gid.UserType:
		user, err := getUserService(ctx).GetUserByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &UserResolver{user: user}
	case gid.VariableType:
		variable, err := getVariableService(ctx).GetVariableByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &NamespaceVariableResolver{variable: variable}
	case gid.WorkspaceType:
		workspace, err := getWorkspaceService(ctx).GetWorkspaceByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &WorkspaceResolver{workspace: workspace}
	case gid.TerraformProviderType:
		provider, err := getProviderRegistryService(ctx).GetProviderByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &TerraformProviderResolver{provider: provider}
	case gid.TerraformProviderVersionType:
		providerVersion, err := getProviderRegistryService(ctx).GetProviderVersionByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &TerraformProviderVersionResolver{providerVersion: providerVersion}
	case gid.TerraformProviderPlatformType:
		providerPlatform, err := getProviderRegistryService(ctx).GetProviderPlatformByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &TerraformProviderPlatformResolver{providerPlatform: providerPlatform}
	case gid.GPGKeyType:
		gpgKey, err := getGPGKeyService(ctx).GetGPGKeyByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &GPGKeyResolver{gpgKey: gpgKey}
	case gid.TeamType:
		team, err := getTeamService(ctx).GetTeamByID(ctx, parsedGlobalID.ID)
		if err != nil {
			return nil, err
		}
		resolver = &TeamResolver{team: team}
	default:
		return nil, fmt.Errorf("node query doesn't support type %s", parsedGlobalID.Type)
	}

	return &NodeResolver{result: resolver}, nil
}
