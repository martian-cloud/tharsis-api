// Package types provides all Tharsis model types
package types

import (
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// TRNPrefix is the prefix for all TRNs.
	TRNPrefix = "trn:"
)

// ModelType is represents all Tharsis Model types.
type ModelType struct {
	name    string
	gidCode string
}

// All possible ModelTypes
var (
	ActivityEventModelType                   = ModelType{name: "activity_event", gidCode: "AE"}
	AnnouncementModelType                    = ModelType{name: "announcement", gidCode: "AN"}
	ApplyModelType                           = ModelType{name: "apply", gidCode: "A"}
	ConfigurationVersionModelType            = ModelType{name: "configuration_version", gidCode: "C"}
	GPGKeyModelType                          = ModelType{name: "gpg_key", gidCode: "GPG"}
	GroupModelType                           = ModelType{name: "group", gidCode: "G"}
	JobModelType                             = ModelType{name: "job", gidCode: "J"}
	LogStreamModelType                       = ModelType{name: "log_stream", gidCode: "LS"}
	MaintenanceModeModelType                 = ModelType{name: "maintenance_mode", gidCode: "MM"}
	ManagedIdentityAccessRuleModelType       = ModelType{name: "managed_identity_access_rule", gidCode: "MR"}
	ManagedIdentityModelType                 = ModelType{name: "managed_identity", gidCode: "M"}
	NamespaceMembershipModelType             = ModelType{name: "namespace_membership", gidCode: "NM"}
	NotificationPreferenceModelType          = ModelType{name: "notification_preference", gidCode: "NP"}
	PlanModelType                            = ModelType{name: "plan", gidCode: "P"}
	ResourceLimitModelType                   = ModelType{name: "resource_limit", gidCode: "RLM"}
	RoleModelType                            = ModelType{name: "role", gidCode: "RL"}
	RunnerModelType                          = ModelType{name: "runner", gidCode: "RNR"}
	RunnerSessionModelType                   = ModelType{name: "runner_session", gidCode: "RS"}
	RunModelType                             = ModelType{name: "run", gidCode: "R"}
	SCIMTokenModelType                       = ModelType{name: "scim_token", gidCode: "ST"}
	ServiceAccountModelType                  = ModelType{name: "service_account", gidCode: "SA"}
	StateVersionOutputModelType              = ModelType{name: "state_version_output", gidCode: "SO"}
	StateVersionModelType                    = ModelType{name: "state_version", gidCode: "SV"}
	TeamMemberModelType                      = ModelType{name: "team_member", gidCode: "TM"}
	TeamModelType                            = ModelType{name: "team", gidCode: "T"}
	TerraformModuleAttestationModelType      = ModelType{name: "terraform_module_attestation", gidCode: "TMA"}
	TerraformModuleModelType                 = ModelType{name: "terraform_module", gidCode: "TMO"}
	TerraformModuleVersionModelType          = ModelType{name: "terraform_module_version", gidCode: "TMV"}
	TerraformProviderPlatformMirrorModelType = ModelType{name: "terraform_provider_platform_mirror", gidCode: "TPM"}
	TerraformProviderPlatformModelType       = ModelType{name: "terraform_provider_platform", gidCode: "TPP"}
	TerraformProviderModelType               = ModelType{name: "terraform_provider", gidCode: "TP"}
	TerraformProviderMirrorModelType         = ModelType{name: "terraform_provider_mirror", gidCode: "TMP"}
	TerraformProviderVersionMirrorModelType  = ModelType{name: "terraform_provider_version_mirror", gidCode: "TVM"}
	TerraformProviderVersionModelType        = ModelType{name: "terraform_provider_version", gidCode: "TPV"}
	UserModelType                            = ModelType{name: "user", gidCode: "U"}
	UserSessionModelType                     = ModelType{name: "user_session", gidCode: "US"}
	VariableModelType                        = ModelType{name: "variable", gidCode: "V"}
	VariableVersionModelType                 = ModelType{name: "variable_version", gidCode: "VV"}
	VCSEventModelType                        = ModelType{name: "vcs_event", gidCode: "VE"}
	VCSProviderModelType                     = ModelType{name: "vcs_provider", gidCode: "VP"}
	WorkspaceAssessmentModelType             = ModelType{name: "workspace_assessment", gidCode: "WA"}
	WorkspaceModelType                       = ModelType{name: "workspace", gidCode: "W"}
	WorkspaceVCSProviderLinkModelType        = ModelType{name: "workspace_vcs_provider_link", gidCode: "WPL"}
	FederatedRegistryModelType               = ModelType{name: "federated_registry", gidCode: "FR"}
)

// ResourcePathFromTRN returns the resource path from a TRN.
func (m ModelType) ResourcePathFromTRN(trn string) (string, error) {
	if !strings.HasPrefix(trn, TRNPrefix) {
		return "", errors.New("not a TRN", errors.WithErrorCode(errors.EInvalid))
	}

	parts := strings.Split(trn[len(TRNPrefix):], ":")

	if len(parts) != 2 {
		return "", errors.New("invalid TRN format", errors.WithErrorCode(errors.EInvalid))
	}

	if parts[0] != m.name {
		return "", errors.New("invalid TRN model type", errors.WithErrorCode(errors.EInvalid))
	}

	resourcePath := parts[1]

	if resourcePath == "" || strings.HasPrefix(resourcePath, "/") || strings.HasSuffix(resourcePath, "/") {
		return "", errors.New("invalid TRN resource path", errors.WithErrorCode(errors.EInvalid))
	}

	return resourcePath, nil
}

// BuildTRN builds a TRN from a Model type and a list of path parts.
func (m ModelType) BuildTRN(a ...string) string {
	return fmt.Sprintf("%s%s:%s", TRNPrefix, m.name, strings.Join(a, "/"))
}

// Name returns the model's name.
func (m ModelType) Name() string {
	return m.name
}

// GIDCode returns the GID code.
func (m ModelType) GIDCode() string {
	return m.gidCode
}

// Equals returns true if two ModelType instances are equal
func (m ModelType) Equals(other ModelType) bool {
	return m.name == other.name && m.gidCode == other.gidCode
}

// IsTRN indicates if the given string contains "trn:" prefix.
func IsTRN(value string) bool {
	return strings.HasPrefix(value, TRNPrefix)
}

// GetModelNameFromTRN parses the model type from a TRN-like string.
func GetModelNameFromTRN(trn string) string {
	if !IsTRN(trn) {
		return ""
	}

	return strings.Split(trn[len(TRNPrefix):], ":")[0]
}
