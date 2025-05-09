package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
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

// ToRunnerSession resolver
func (r *NodeResolver) ToRunnerSession() (*RunnerSessionResolver, bool) {
	res, ok := r.result.(*RunnerSessionResolver)
	return res, ok
}

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

// ToNamespaceVariableVersion resolver
func (r *NodeResolver) ToNamespaceVariableVersion() (*NamespaceVariableVersionResolver, bool) {
	res, ok := r.result.(*NamespaceVariableVersionResolver)
	return res, ok
}

// ToWorkspace resolver
func (r *NodeResolver) ToWorkspace() (*WorkspaceResolver, bool) {
	res, ok := r.result.(*WorkspaceResolver)
	return res, ok
}

// ToWorkspaceAssessment resolver
func (r *NodeResolver) ToWorkspaceAssessment() (*WorkspaceAssessmentResolver, bool) {
	res, ok := r.result.(*WorkspaceAssessmentResolver)
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

// ToTerraformModule resolver
func (r *NodeResolver) ToTerraformModule() (*TerraformModuleResolver, bool) {
	res, ok := r.result.(*TerraformModuleResolver)
	return res, ok
}

// ToTerraformModuleVersion resolver
func (r *NodeResolver) ToTerraformModuleVersion() (*TerraformModuleVersionResolver, bool) {
	res, ok := r.result.(*TerraformModuleVersionResolver)
	return res, ok
}

// ToTerraformModuleAttestation resolver
func (r *NodeResolver) ToTerraformModuleAttestation() (*TerraformModuleAttestationResolver, bool) {
	res, ok := r.result.(*TerraformModuleAttestationResolver)
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

// ToVCSProvider resolver
func (r *NodeResolver) ToVCSProvider() (*VCSProviderResolver, bool) {
	res, ok := r.result.(*VCSProviderResolver)
	return res, ok
}

// ToWorkspaceVCSProviderLink resolver
func (r *NodeResolver) ToWorkspaceVCSProviderLink() (*WorkspaceVCSProviderLinkResolver, bool) {
	res, ok := r.result.(*WorkspaceVCSProviderLinkResolver)
	return res, ok
}

// ToVCSEvent resolver
func (r *NodeResolver) ToVCSEvent() (*VCSEventResolver, bool) {
	res, ok := r.result.(*VCSEventResolver)
	return res, ok
}

// ToRole resolver
func (r *NodeResolver) ToRole() (*RoleResolver, bool) {
	res, ok := r.result.(*RoleResolver)
	return res, ok
}

// ToRunner resolver
func (r *NodeResolver) ToRunner() (*RunnerResolver, bool) {
	res, ok := r.result.(*RunnerResolver)
	return res, ok
}

// ToTerraformProviderVersionMirror resolver
func (r *NodeResolver) ToTerraformProviderVersionMirror() (*TerraformProviderVersionMirrorResolver, bool) {
	res, ok := r.result.(*TerraformProviderVersionMirrorResolver)
	return res, ok
}

// ToTerraformProviderPlatformMirror resolver
func (r *NodeResolver) ToTerraformProviderPlatformMirror() (*TerraformProviderPlatformMirrorResolver, bool) {
	res, ok := r.result.(*TerraformProviderPlatformMirrorResolver)
	return res, ok
}

// ToFederatedRegistry resolver
func (r *NodeResolver) ToFederatedRegistry() (*FederatedRegistryResolver, bool) {
	res, ok := r.result.(*FederatedRegistryResolver)
	return res, ok
}

func node(ctx context.Context, globalID string) (*NodeResolver, error) {
	parsedGlobalID, err := gid.ParseGlobalID(globalID)
	if err != nil {
		return nil, err
	}

	var resolver interface{}
	var retErr error

	switch parsedGlobalID.Type {
	case gid.ApplyType:
		apply, err := getRunService(ctx).GetApply(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &ApplyResolver{apply: apply}
	case gid.ConfigurationVersionType:
		cv, err := getWorkspaceService(ctx).GetConfigurationVersion(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &ConfigurationVersionResolver{configurationVersion: cv}
	case gid.GroupType:
		group, err := getGroupService(ctx).GetGroupByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &GroupResolver{group: group}
	case gid.JobType:
		job, err := getJobService(ctx).GetJob(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &JobResolver{job: job}
	case gid.RunnerSessionType:
		session, err := getRunnerService(ctx).GetRunnerSessionByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &RunnerSessionResolver{session: session}
	case gid.ManagedIdentityType:
		managedIdentity, err := getManagedIdentityService(ctx).GetManagedIdentityByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &ManagedIdentityResolver{managedIdentity: managedIdentity}
	case gid.ManagedIdentityAccessRuleType:
		rule, err := getManagedIdentityService(ctx).GetManagedIdentityAccessRule(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &ManagedIdentityAccessRuleResolver{rule: rule}
	case gid.NamespaceMembershipType:
		namespaceMembership, err := getNamespaceMembershipService(ctx).GetNamespaceMembershipByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &NamespaceMembershipResolver{namespaceMembership: namespaceMembership}
	case gid.PlanType:
		plan, err := getRunService(ctx).GetPlan(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &PlanResolver{plan: plan}
	case gid.RunType:
		run, err := getRunService(ctx).GetRun(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &RunResolver{run: run}
	case gid.ServiceAccountType:
		serviceAccount, err := getSAService(ctx).GetServiceAccountByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &ServiceAccountResolver{serviceAccount: serviceAccount}
	case gid.StateVersionType:
		stateVersion, err := getWorkspaceService(ctx).GetStateVersion(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &StateVersionResolver{stateVersion: stateVersion}
	case gid.StateVersionOutputType:
		stateVersionOutput, err := getStateVersionOutputs(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = stateVersionOutput
	case gid.UserType:
		user, err := getUserService(ctx).GetUserByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &UserResolver{user: user}
	case gid.VariableType:
		variable, err := getVariableService(ctx).GetVariableByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &NamespaceVariableResolver{variable: variable}
	case gid.VariableVersionType:
		version, err := getVariableService(ctx).GetVariableVersionByID(ctx, parsedGlobalID.ID, false)
		if err != nil {
			retErr = err
			break
		}
		resolver = &NamespaceVariableVersionResolver{version: version}
	case gid.WorkspaceType:
		workspace, err := getWorkspaceService(ctx).GetWorkspaceByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &WorkspaceResolver{workspace: workspace}
	case gid.WorkspaceAssessmentType:
		assessment, err := getWorkspaceService(ctx).GetWorkspaceAssessmentByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &WorkspaceAssessmentResolver{assessment: assessment}
	case gid.TerraformProviderType:
		provider, err := getProviderRegistryService(ctx).GetProviderByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformProviderResolver{provider: provider}
	case gid.TerraformProviderVersionType:
		providerVersion, err := getProviderRegistryService(ctx).GetProviderVersionByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformProviderVersionResolver{providerVersion: providerVersion}
	case gid.TerraformProviderPlatformType:
		providerPlatform, err := getProviderRegistryService(ctx).GetProviderPlatformByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformProviderPlatformResolver{providerPlatform: providerPlatform}
	case gid.TerraformModuleType:
		module, err := getModuleRegistryService(ctx).GetModuleByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformModuleResolver{module: module}
	case gid.TerraformModuleVersionType:
		moduleVersion, err := getModuleRegistryService(ctx).GetModuleVersionByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformModuleVersionResolver{moduleVersion: moduleVersion}
	case gid.TerraformModuleAttestationType:
		attestation, err := getModuleRegistryService(ctx).GetModuleAttestationByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformModuleAttestationResolver{moduleAttestation: attestation}
	case gid.GPGKeyType:
		gpgKey, err := getGPGKeyService(ctx).GetGPGKeyByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &GPGKeyResolver{gpgKey: gpgKey}
	case gid.TeamType:
		team, err := getTeamService(ctx).GetTeamByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TeamResolver{team: team}
	case gid.VCSProviderType:
		vcsProvider, err := getVCSService(ctx).GetVCSProviderByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &VCSProviderResolver{vcsProvider: vcsProvider}
	case gid.WorkspaceVCSProviderLinkType:
		link, err := getVCSService(ctx).GetWorkspaceVCSProviderLinkByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &WorkspaceVCSProviderLinkResolver{workspaceVCSProviderLink: link}
	case gid.VCSEventType:
		vcsEvent, err := getVCSService(ctx).GetVCSEventByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &VCSEventResolver{vcsEvent: vcsEvent}
	case gid.RoleType:
		gotRole, err := getRoleService(ctx).GetRoleByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &RoleResolver{role: gotRole}
	case gid.RunnerType:
		runner, err := getRunnerService(ctx).GetRunnerByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &RunnerResolver{runner: runner}
	case gid.TerraformProviderVersionMirrorType:
		mirror, err := getProviderMirrorService(ctx).GetProviderVersionMirrorByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformProviderVersionMirrorResolver{versionMirror: mirror}
	case gid.TerraformProviderPlatformMirrorType:
		mirror, err := getProviderMirrorService(ctx).GetProviderPlatformMirrorByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &TerraformProviderPlatformMirrorResolver{platformMirror: mirror}
	case gid.FederatedRegistryType:
		registry, err := getFederatedRegistryService(ctx).GetFederatedRegistryByID(ctx, parsedGlobalID.ID)
		if err != nil {
			retErr = err
			break
		}
		resolver = &FederatedRegistryResolver{federatedRegistry: registry}
	default:
		return nil, fmt.Errorf("node query doesn't support type %s", parsedGlobalID.Type)
	}

	if retErr != nil {
		if errors.ErrorCode(retErr) == errors.ENotFound {
			return nil, nil
		}
		return nil, retErr
	}

	return &NodeResolver{result: resolver}, nil
}
