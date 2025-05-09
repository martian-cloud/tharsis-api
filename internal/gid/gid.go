// Package gid package
package gid

import (
	"encoding/base64"
	"fmt"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Type is the type of the global ID
type Type string

// Type constants
const (
	ApplyType                           Type = "A"
	ConfigurationVersionType            Type = "C"
	GroupType                           Type = "G"
	JobType                             Type = "J"
	LogStreamType                       Type = "LS"
	ManagedIdentityType                 Type = "M"
	ManagedIdentityAccessRuleType       Type = "MR"
	NamespaceMembershipType             Type = "NM"
	PlanType                            Type = "P"
	RunType                             Type = "R"
	RunnerType                          Type = "RNR"
	RunnerSessionType                   Type = "RS"
	ServiceAccountType                  Type = "SA"
	StateVersionType                    Type = "SV"
	StateVersionOutputType              Type = "SO"
	TeamType                            Type = "T"
	TeamMemberType                      Type = "TM"
	UserType                            Type = "U"
	VariableType                        Type = "V"
	VariableVersionType                 Type = "VV"
	WorkspaceType                       Type = "W"
	TerraformProviderType               Type = "TP"
	TerraformProviderVersionType        Type = "TPV"
	TerraformProviderPlatformType       Type = "TPP"
	TerraformModuleType                 Type = "TMO"
	TerraformModuleVersionType          Type = "TMV"
	TerraformModuleAttestationType      Type = "TMA"
	GPGKeyType                          Type = "GPG"
	ActivityEventType                   Type = "AE"
	VCSProviderType                     Type = "VP"
	WorkspaceVCSProviderLinkType        Type = "WPL"
	WorkspaceAssessmentType             Type = "WA"
	VCSEventType                        Type = "VE"
	RoleType                            Type = "RL"
	ResourceLimitType                   Type = "RLM"
	TerraformProviderVersionMirrorType  Type = "TVM"
	TerraformProviderPlatformMirrorType Type = "TPM"
	MaintenanceModeType                 Type = "MM"
	FederatedRegistryType               Type = "FR"
)

// IsValid returns true if this is a valid Type enum
func (t Type) IsValid() error {
	switch t {
	case ApplyType,
		ConfigurationVersionType,
		GroupType,
		JobType,
		LogStreamType,
		ManagedIdentityType,
		ManagedIdentityAccessRuleType,
		NamespaceMembershipType,
		PlanType,
		RunType,
		RunnerType,
		RunnerSessionType,
		ServiceAccountType,
		StateVersionType,
		StateVersionOutputType,
		TeamType,
		TeamMemberType,
		UserType,
		VariableType,
		VariableVersionType,
		WorkspaceType,
		TerraformProviderType,
		TerraformProviderVersionType,
		TerraformProviderPlatformType,
		TerraformModuleType,
		TerraformModuleVersionType,
		TerraformModuleAttestationType,
		GPGKeyType,
		ActivityEventType,
		VCSProviderType,
		WorkspaceVCSProviderLinkType,
		WorkspaceAssessmentType,
		VCSEventType,
		RoleType,
		ResourceLimitType,
		TerraformProviderVersionMirrorType,
		TerraformProviderPlatformMirrorType,
		MaintenanceModeType,
		FederatedRegistryType:
		return nil
	}
	return errors.New("invalid ID type %s", t, errors.WithErrorCode(errors.EInvalid))
}

// GlobalID is a model ID with type information
type GlobalID struct {
	Type Type
	ID   string
}

// NewGlobalID returns a new GlobalID
func NewGlobalID(modelType Type, modelID string) *GlobalID {
	return &GlobalID{Type: modelType, ID: modelID}
}

// String returns the string representation of the global ID
func (g *GlobalID) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s_%s", g.Type, g.ID)))
}

// ParseGlobalID parses a global ID string and returns a GlobalID type
func ParseGlobalID(globalID string) (*GlobalID, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(globalID)
	if err != nil {
		return nil, err
	}

	decodedGlobalID := string(decodedBytes)

	index := strings.Index(decodedGlobalID, "_")
	if index == -1 {
		return nil, errors.New("Invalid ID", errors.WithErrorCode(errors.EInvalid))
	}

	t := Type(decodedGlobalID[:index])
	if err := t.IsValid(); err != nil {
		return nil, err
	}

	return NewGlobalID(t, decodedGlobalID[index+1:]), nil
}

// ToGlobalID converts a model type and DB ID to a global ID string
func ToGlobalID(idType Type, id string) string {
	return NewGlobalID(idType, id).String()
}

// FromGlobalID converts a global ID string to a DB ID string
func FromGlobalID(globalID string) string {
	gid, err := ParseGlobalID(globalID)
	if err != nil {
		return fmt.Sprintf("invalid[%s]", globalID)
	}
	return gid.ID
}
