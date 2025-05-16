package resolver

import (
	"context"
	"fmt"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

// NodeResolver resolves a node type
type NodeResolver struct {
	result any
}

type idGetter interface {
	ID() graphql.ID
}

// ID resolver
func (r *NodeResolver) ID() (graphql.ID, error) {
	switch v := r.result.(type) {
	case models.Model:
		// This is a Model
		return graphql.ID(v.GetGlobalID()), nil
	case idGetter:
		// This is a GraphQL resolver
		return v.ID(), nil
	default:
		return "", fmt.Errorf("unexpected type in node resolver: %T", r.result)
	}
}

// ToApply resolver
func (r *NodeResolver) ToApply() (*ApplyResolver, bool) {
	switch res := r.result.(type) {
	case *ApplyResolver:
		return res, true
	case *models.Apply:
		return &ApplyResolver{apply: res}, true
	default:
		return nil, false
	}
}

// ToConfigurationVersion resolver
func (r *NodeResolver) ToConfigurationVersion() (*ConfigurationVersionResolver, bool) {
	switch res := r.result.(type) {
	case *ConfigurationVersionResolver:
		return res, true
	case *models.ConfigurationVersion:
		return &ConfigurationVersionResolver{configurationVersion: res}, true
	default:
		return nil, false
	}
}

// ToGroup resolver
func (r *NodeResolver) ToGroup() (*GroupResolver, bool) {
	switch res := r.result.(type) {
	case *GroupResolver:
		return res, true
	case *models.Group:
		return &GroupResolver{group: res}, true
	default:
		return nil, false
	}
}

// ToJob resolver
func (r *NodeResolver) ToJob() (*JobResolver, bool) {
	switch res := r.result.(type) {
	case *JobResolver:
		return res, true
	case *models.Job:
		return &JobResolver{job: res}, true
	default:
		return nil, false
	}
}

// ToRunnerSession resolver
func (r *NodeResolver) ToRunnerSession() (*RunnerSessionResolver, bool) {
	switch res := r.result.(type) {
	case *RunnerSessionResolver:
		return res, true
	case *models.RunnerSession:
		return &RunnerSessionResolver{session: res}, true
	default:
		return nil, false
	}
}

// ToManagedIdentity resolver
func (r *NodeResolver) ToManagedIdentity() (*ManagedIdentityResolver, bool) {
	switch res := r.result.(type) {
	case *ManagedIdentityResolver:
		return res, true
	case *models.ManagedIdentity:
		return &ManagedIdentityResolver{managedIdentity: res}, true
	default:
		return nil, false
	}
}

// ToManagedIdentityAccessRule resolver
func (r *NodeResolver) ToManagedIdentityAccessRule() (*ManagedIdentityAccessRuleResolver, bool) {
	switch res := r.result.(type) {
	case *ManagedIdentityAccessRuleResolver:
		return res, true
	case *models.ManagedIdentityAccessRule:
		return &ManagedIdentityAccessRuleResolver{rule: res}, true
	default:
		return nil, false
	}
}

// ToNamespaceMembership resolver
func (r *NodeResolver) ToNamespaceMembership() (*NamespaceMembershipResolver, bool) {
	switch res := r.result.(type) {
	case *NamespaceMembershipResolver:
		return res, true
	case *models.NamespaceMembership:
		return &NamespaceMembershipResolver{namespaceMembership: res}, true
	default:
		return nil, false
	}
}

// ToPlan resolver
func (r *NodeResolver) ToPlan() (*PlanResolver, bool) {
	switch res := r.result.(type) {
	case *PlanResolver:
		return res, true
	case *models.Plan:
		return &PlanResolver{plan: res}, true
	default:
		return nil, false
	}
}

// ToRun resolver
func (r *NodeResolver) ToRun() (*RunResolver, bool) {
	switch res := r.result.(type) {
	case *RunResolver:
		return res, true
	case *models.Run:
		return &RunResolver{run: res}, true
	default:
		return nil, false
	}
}

// ToServiceAccount resolver
func (r *NodeResolver) ToServiceAccount() (*ServiceAccountResolver, bool) {
	switch res := r.result.(type) {
	case *ServiceAccountResolver:
		return res, true
	case *models.ServiceAccount:
		return &ServiceAccountResolver{serviceAccount: res}, true
	default:
		return nil, false
	}
}

// ToStateVersion resolver
func (r *NodeResolver) ToStateVersion() (*StateVersionResolver, bool) {
	switch res := r.result.(type) {
	case *StateVersionResolver:
		return res, true
	case *models.StateVersion:
		return &StateVersionResolver{stateVersion: res}, true
	default:
		return nil, false
	}
}

// ToStateVersionOutput resolver
func (r *NodeResolver) ToStateVersionOutput() (*StateVersionOutputResolver, bool) {
	switch res := r.result.(type) {
	case *StateVersionOutputResolver:
		return res, true
	case *models.StateVersionOutput:
		return &StateVersionOutputResolver{stateVersionOutput: res}, true
	default:
		return nil, false
	}
}

// ToUser resolver
func (r *NodeResolver) ToUser() (*UserResolver, bool) {
	switch res := r.result.(type) {
	case *UserResolver:
		return res, true
	case *models.User:
		return &UserResolver{user: res}, true
	default:
		return nil, false
	}
}

// ToNamespaceVariable resolver
func (r *NodeResolver) ToNamespaceVariable() (*NamespaceVariableResolver, bool) {
	switch res := r.result.(type) {
	case *NamespaceVariableResolver:
		return res, true
	case *models.Variable:
		return &NamespaceVariableResolver{variable: res}, true
	default:
		return nil, false
	}
}

// ToNamespaceVariableVersion resolver
func (r *NodeResolver) ToNamespaceVariableVersion() (*NamespaceVariableVersionResolver, bool) {
	switch res := r.result.(type) {
	case *NamespaceVariableVersionResolver:
		return res, true
	case *models.VariableVersion:
		return &NamespaceVariableVersionResolver{version: res}, true
	default:
		return nil, false
	}
}

// ToWorkspace resolver
func (r *NodeResolver) ToWorkspace() (*WorkspaceResolver, bool) {
	switch res := r.result.(type) {
	case *WorkspaceResolver:
		return res, true
	case *models.Workspace:
		return &WorkspaceResolver{workspace: res}, true
	default:
		return nil, false
	}
}

// ToWorkspaceAssessment resolver
func (r *NodeResolver) ToWorkspaceAssessment() (*WorkspaceAssessmentResolver, bool) {
	switch res := r.result.(type) {
	case *WorkspaceAssessmentResolver:
		return res, true
	case *models.WorkspaceAssessment:
		return &WorkspaceAssessmentResolver{assessment: res}, true
	default:
		return nil, false
	}
}

// ToTeam resolver
func (r *NodeResolver) ToTeam() (*TeamResolver, bool) {
	switch res := r.result.(type) {
	case *TeamResolver:
		return res, true
	case *models.Team:
		return &TeamResolver{team: res}, true
	default:
		return nil, false
	}
}

// ToTerraformProvider resolver
func (r *NodeResolver) ToTerraformProvider() (*TerraformProviderResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformProviderResolver:
		return res, true
	case *models.TerraformProvider:
		return &TerraformProviderResolver{provider: res}, true
	default:
		return nil, false
	}
}

// ToTerraformProviderVersion resolver
func (r *NodeResolver) ToTerraformProviderVersion() (*TerraformProviderVersionResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformProviderVersionResolver:
		return res, true
	case *models.TerraformProviderVersion:
		return &TerraformProviderVersionResolver{providerVersion: res}, true
	default:
		return nil, false
	}
}

// ToTerraformProviderPlatform resolver
func (r *NodeResolver) ToTerraformProviderPlatform() (*TerraformProviderPlatformResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformProviderPlatformResolver:
		return res, true
	case *models.TerraformProviderPlatform:
		return &TerraformProviderPlatformResolver{providerPlatform: res}, true
	default:
		return nil, false
	}
}

// ToTerraformModule resolver
func (r *NodeResolver) ToTerraformModule() (*TerraformModuleResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformModuleResolver:
		return res, true
	case *models.TerraformModule:
		return &TerraformModuleResolver{module: res}, true
	default:
		return nil, false
	}
}

// ToTerraformModuleVersion resolver
func (r *NodeResolver) ToTerraformModuleVersion() (*TerraformModuleVersionResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformModuleVersionResolver:
		return res, true
	case *models.TerraformModuleVersion:
		return &TerraformModuleVersionResolver{moduleVersion: res}, true
	default:
		return nil, false
	}
}

// ToTerraformModuleAttestation resolver
func (r *NodeResolver) ToTerraformModuleAttestation() (*TerraformModuleAttestationResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformModuleAttestationResolver:
		return res, true
	case *models.TerraformModuleAttestation:
		return &TerraformModuleAttestationResolver{moduleAttestation: res}, true
	default:
		return nil, false
	}
}

// ToGPGKey resolver
func (r *NodeResolver) ToGPGKey() (*GPGKeyResolver, bool) {
	switch res := r.result.(type) {
	case *GPGKeyResolver:
		return res, true
	case *models.GPGKey:
		return &GPGKeyResolver{gpgKey: res}, true
	default:
		return nil, false
	}
}

// ToActivityEvent resolver
func (r *NodeResolver) ToActivityEvent() (*ActivityEventResolver, bool) {
	switch res := r.result.(type) {
	case *ActivityEventResolver:
		return res, true
	case *models.ActivityEvent:
		return &ActivityEventResolver{activityEvent: res}, true
	default:
		return nil, false
	}
}

// ToVCSProvider resolver
func (r *NodeResolver) ToVCSProvider() (*VCSProviderResolver, bool) {
	switch res := r.result.(type) {
	case *VCSProviderResolver:
		return res, true
	case *models.VCSProvider:
		return &VCSProviderResolver{vcsProvider: res}, true
	default:
		return nil, false
	}
}

// ToWorkspaceVCSProviderLink resolver
func (r *NodeResolver) ToWorkspaceVCSProviderLink() (*WorkspaceVCSProviderLinkResolver, bool) {
	switch res := r.result.(type) {
	case *WorkspaceVCSProviderLinkResolver:
		return res, true
	case *models.WorkspaceVCSProviderLink:
		return &WorkspaceVCSProviderLinkResolver{workspaceVCSProviderLink: res}, true
	default:
		return nil, false
	}
}

// ToVCSEvent resolver
func (r *NodeResolver) ToVCSEvent() (*VCSEventResolver, bool) {
	switch res := r.result.(type) {
	case *VCSEventResolver:
		return res, true
	case *models.VCSEvent:
		return &VCSEventResolver{vcsEvent: res}, true
	default:
		return nil, false
	}
}

// ToRole resolver
func (r *NodeResolver) ToRole() (*RoleResolver, bool) {
	switch res := r.result.(type) {
	case *RoleResolver:
		return res, true
	case *models.Role:
		return &RoleResolver{role: res}, true
	default:
		return nil, false
	}
}

// ToRunner resolver
func (r *NodeResolver) ToRunner() (*RunnerResolver, bool) {
	switch res := r.result.(type) {
	case *RunnerResolver:
		return res, true
	case *models.Runner:
		return &RunnerResolver{runner: res}, true
	default:
		return nil, false
	}
}

// ToTerraformProviderVersionMirror resolver
func (r *NodeResolver) ToTerraformProviderVersionMirror() (*TerraformProviderVersionMirrorResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformProviderVersionMirrorResolver:
		return res, true
	case *models.TerraformProviderVersionMirror:
		return &TerraformProviderVersionMirrorResolver{versionMirror: res}, true
	default:
		return nil, false
	}
}

// ToTerraformProviderPlatformMirror resolver
func (r *NodeResolver) ToTerraformProviderPlatformMirror() (*TerraformProviderPlatformMirrorResolver, bool) {
	switch res := r.result.(type) {
	case *TerraformProviderPlatformMirrorResolver:
		return res, true
	case *models.TerraformProviderPlatformMirror:
		return &TerraformProviderPlatformMirrorResolver{platformMirror: res}, true
	default:
		return nil, false
	}
}

// ToFederatedRegistry resolver
func (r *NodeResolver) ToFederatedRegistry() (*FederatedRegistryResolver, bool) {
	switch res := r.result.(type) {
	case *FederatedRegistryResolver:
		return res, true
	case *models.FederatedRegistry:
		return &FederatedRegistryResolver{federatedRegistry: res}, true
	default:
		return nil, false
	}
}

func node(ctx context.Context, value string) (*NodeResolver, error) {
	model, err := getServiceCatalog(ctx).FetchModel(ctx, value)
	if err != nil {
		return nil, err
	}

	return &NodeResolver{result: model}, nil
}
