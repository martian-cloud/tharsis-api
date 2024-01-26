package permissions

// ResourceType is an enum representing a Tharsis resource type.
type ResourceType string

// ResourceType constants.
const (
	GPGKeyResourceType                  ResourceType = "gpg_key"
	GroupResourceType                   ResourceType = "group"
	WorkspaceResourceType               ResourceType = "workspace"
	NamespaceMembershipResourceType     ResourceType = "namespace_membership"
	TeamResourceType                    ResourceType = "team"
	RunResourceType                     ResourceType = "run"
	JobResourceType                     ResourceType = "job"
	PlanResourceType                    ResourceType = "plan"
	ApplyResourceType                   ResourceType = "apply"
	RunnerResourceType                  ResourceType = "runner"
	RunnerSessionResourceType           ResourceType = "runner_session"
	UserResourceType                    ResourceType = "user"
	VariableResourceType                ResourceType = "variable"
	TerraformProviderResourceType       ResourceType = "terraform_provider"
	TerraformModuleResourceType         ResourceType = "terraform_module"
	StateVersionResourceType            ResourceType = "state_version"
	ConfigurationVersionResourceType    ResourceType = "configuration_version"
	ServiceAccountResourceType          ResourceType = "service_account"
	ManagedIdentityResourceType         ResourceType = "managed_identity"
	VCSProviderResourceType             ResourceType = "vcs_provider"
	TerraformProviderMirrorResourceType ResourceType = "terraform_provider_mirror"
)
