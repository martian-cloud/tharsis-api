// Package types provides all Tharsis model types
package types

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// ModelType is represents all Tharsis Model types.
type ModelType struct {
	trnType trn.Type
	gidCode string
}

// All possible ModelTypes
var (
	ActivityEventModelType                   = ModelType{trnType: trn.TypeActivityEvent, gidCode: "AE"}
	AgentCreditQuotaModelType                = ModelType{trnType: trn.TypeAgentCreditQuota, gidCode: "ACQ"}
	AgentSessionMessageModelType             = ModelType{trnType: trn.TypeAgentSessionMessage, gidCode: "ASM"}
	AgentSessionModelType                    = ModelType{trnType: trn.TypeAgentSession, gidCode: "AGS"}
	AgentSessionRunModelType                 = ModelType{trnType: trn.TypeAgentSessionRun, gidCode: "ASR"}
	AnnouncementModelType                    = ModelType{trnType: trn.TypeAnnouncement, gidCode: "AN"}
	ApplyModelType                           = ModelType{trnType: trn.TypeApply, gidCode: "A"}
	AsymSigningKeyModelType                  = ModelType{trnType: trn.TypeAsymSigningKey, gidCode: "ASK"}
	ConfigurationVersionModelType            = ModelType{trnType: trn.TypeConfigurationVersion, gidCode: "C"}
	FederatedRegistryModelType               = ModelType{trnType: trn.TypeFederatedRegistry, gidCode: "FR"}
	GPGKeyModelType                          = ModelType{trnType: trn.TypeGPGKey, gidCode: "GPG"}
	GroupModelType                           = ModelType{trnType: trn.TypeGroup, gidCode: "G"}
	JobModelType                             = ModelType{trnType: trn.TypeJob, gidCode: "J"}
	LogStreamModelType                       = ModelType{trnType: trn.TypeLogStream, gidCode: "LS"}
	MaintenanceModeModelType                 = ModelType{trnType: trn.TypeMaintenanceMode, gidCode: "MM"}
	ManagedIdentityAccessRuleModelType       = ModelType{trnType: trn.TypeManagedIdentityAccessRule, gidCode: "MR"}
	ManagedIdentityModelType                 = ModelType{trnType: trn.TypeManagedIdentity, gidCode: "M"}
	NamespaceFavoriteModelType               = ModelType{trnType: trn.TypeNamespaceFavorite, gidCode: "NF"}
	NamespaceMembershipModelType             = ModelType{trnType: trn.TypeNamespaceMembership, gidCode: "NM"}
	NotificationPreferenceModelType          = ModelType{trnType: trn.TypeNotificationPreference, gidCode: "NP"}
	PlanModelType                            = ModelType{trnType: trn.TypePlan, gidCode: "P"}
	ResourceLimitModelType                   = ModelType{trnType: trn.TypeResourceLimit, gidCode: "RLM"}
	RoleModelType                            = ModelType{trnType: trn.TypeRole, gidCode: "RL"}
	RunModelType                             = ModelType{trnType: trn.TypeRun, gidCode: "R"}
	RunnerModelType                          = ModelType{trnType: trn.TypeRunner, gidCode: "RNR"}
	RunnerSessionModelType                   = ModelType{trnType: trn.TypeRunnerSession, gidCode: "RS"}
	SCIMTokenModelType                       = ModelType{trnType: trn.TypeSCIMToken, gidCode: "ST"}
	ServiceAccountModelType                  = ModelType{trnType: trn.TypeServiceAccount, gidCode: "SA"}
	StateVersionModelType                    = ModelType{trnType: trn.TypeStateVersion, gidCode: "SV"}
	StateVersionOutputModelType              = ModelType{trnType: trn.TypeStateVersionOutput, gidCode: "SO"}
	TeamMemberModelType                      = ModelType{trnType: trn.TypeTeamMember, gidCode: "TM"}
	TeamModelType                            = ModelType{trnType: trn.TypeTeam, gidCode: "T"}
	TerraformModuleAttestationModelType      = ModelType{trnType: trn.TypeTerraformModuleAttestation, gidCode: "TMA"}
	TerraformModuleModelType                 = ModelType{trnType: trn.TypeTerraformModule, gidCode: "TMO"}
	TerraformModuleVersionModelType          = ModelType{trnType: trn.TypeTerraformModuleVersion, gidCode: "TMV"}
	TerraformProviderMirrorModelType         = ModelType{trnType: trn.TypeTerraformProviderMirror, gidCode: "TMP"}
	TerraformProviderModelType               = ModelType{trnType: trn.TypeTerraformProvider, gidCode: "TP"}
	TerraformProviderPlatformMirrorModelType = ModelType{trnType: trn.TypeTerraformProviderPlatformMirror, gidCode: "TPM"}
	TerraformProviderPlatformModelType       = ModelType{trnType: trn.TypeTerraformProviderPlatform, gidCode: "TPP"}
	TerraformProviderVersionMirrorModelType  = ModelType{trnType: trn.TypeTerraformProviderVersionMirror, gidCode: "TVM"}
	TerraformProviderVersionModelType        = ModelType{trnType: trn.TypeTerraformProviderVersion, gidCode: "TPV"}
	UserModelType                            = ModelType{trnType: trn.TypeUser, gidCode: "U"}
	UserSessionModelType                     = ModelType{trnType: trn.TypeUserSession, gidCode: "US"}
	VariableModelType                        = ModelType{trnType: trn.TypeVariable, gidCode: "V"}
	VariableVersionModelType                 = ModelType{trnType: trn.TypeVariableVersion, gidCode: "VV"}
	VCSEventModelType                        = ModelType{trnType: trn.TypeVCSEvent, gidCode: "VE"}
	VCSProviderModelType                     = ModelType{trnType: trn.TypeVCSProvider, gidCode: "VP"}
	WorkspaceAssessmentModelType             = ModelType{trnType: trn.TypeWorkspaceAssessment, gidCode: "WA"}
	WorkspaceModelType                       = ModelType{trnType: trn.TypeWorkspace, gidCode: "W"}
	WorkspaceVCSProviderLinkModelType        = ModelType{trnType: trn.TypeWorkspaceVCSProviderLink, gidCode: "WPL"}
)

// Name returns the model's name.
func (m ModelType) Name() string {
	return m.trnType.String()
}

// TRNType returns the trn.Type for this model.
func (m ModelType) TRNType() trn.Type {
	return m.trnType
}

// GIDCode returns the GID code.
func (m ModelType) GIDCode() string {
	return m.gidCode
}

// Equals returns true if two ModelType instances are equal
func (m ModelType) Equals(other ModelType) bool {
	return m.trnType == other.trnType && m.gidCode == other.gidCode
}
