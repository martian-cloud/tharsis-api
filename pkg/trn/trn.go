// Package trn provides utilities for working with Tharsis Resource Names (TRNs).
//
// A TRN has the format "trn:<type>:<path>" where type is the resource type
// name (e.g. "workspace") and path is the resource path (e.g. "group/my-ws").
//
// Parse a TRN and access its fields through methods:
//
//	parsed, err := trn.ParseAny("trn:workspace:group/my-ws")
//	parsed.Type()       // trn.TypeWorkspace
//	parsed.Path()       // "group/my-ws"
//	parsed.PathParts()  // ["group", "my-ws"]
//	parsed.ParentPath() // "group"
//	parsed.BaseName()   // "my-ws"
//	parsed.HasParent()  // true
//	parsed.String()     // "trn:workspace:group/my-ws"
//
// Build TRNs from type constants:
//
//	trn.TypeWorkspace.Build("group/my-ws")  // "trn:workspace:group/my-ws"
//	trn.TypeWorkspace.Normalize(identifier) // path/GID/TRN → TRN or GID
package trn

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const (
	trnPrefix = "trn:"
)

// Type represents a TRN resource type name.
type Type string

// Resource type constants.
const (
	TypeActivityEvent                   Type = "activity_event"
	TypeAgentCreditQuota                Type = "agent_credit_quota"
	TypeAgentSession                    Type = "agent_session"
	TypeAgentSessionMessage             Type = "agent_session_message"
	TypeAgentSessionRun                 Type = "agent_session_run"
	TypeAnnouncement                    Type = "announcement"
	TypeApply                           Type = "apply"
	TypeAsymSigningKey                  Type = "asym_signing_key"
	TypeConfigurationVersion            Type = "configuration_version"
	TypeFederatedRegistry               Type = "federated_registry"
	TypeGPGKey                          Type = "gpg_key"
	TypeGroup                           Type = "group"
	TypeJob                             Type = "job"
	TypeLogStream                       Type = "log_stream"
	TypeLogStreamChunk                  Type = "log_stream_chunk"
	TypeMaintenanceMode                 Type = "maintenance_mode"
	TypeManagedIdentity                 Type = "managed_identity"
	TypeManagedIdentityAccessRule       Type = "managed_identity_access_rule"
	TypeNamespaceFavorite               Type = "namespace_favorite"
	TypeNamespaceMembership             Type = "namespace_membership"
	TypeNotificationPreference          Type = "notification_preference"
	TypePlan                            Type = "plan"
	TypeResourceLimit                   Type = "resource_limit"
	TypeRole                            Type = "role"
	TypeRun                             Type = "run"
	TypeRunner                          Type = "runner"
	TypeRunnerSession                   Type = "runner_session"
	TypeSCIMToken                       Type = "scim_token"
	TypeServiceAccount                  Type = "service_account"
	TypeStateVersion                    Type = "state_version"
	TypeStateVersionOutput              Type = "state_version_output"
	TypeTeam                            Type = "team"
	TypeTeamMember                      Type = "team_member"
	TypeTerraformModule                 Type = "terraform_module"
	TypeTerraformModuleAttestation      Type = "terraform_module_attestation"
	TypeTerraformModuleVersion          Type = "terraform_module_version"
	TypeTerraformProvider               Type = "terraform_provider"
	TypeTerraformProviderMirror         Type = "terraform_provider_mirror"
	TypeTerraformProviderPlatform       Type = "terraform_provider_platform"
	TypeTerraformProviderPlatformMirror Type = "terraform_provider_platform_mirror"
	TypeTerraformProviderVersion        Type = "terraform_provider_version"
	TypeTerraformProviderVersionMirror  Type = "terraform_provider_version_mirror"
	TypeUser                            Type = "user"
	TypeUserSession                     Type = "user_session"
	TypeVariable                        Type = "variable"
	TypeVariableVersion                 Type = "variable_version"
	TypeVCSEvent                        Type = "vcs_event"
	TypeVCSProvider                     Type = "vcs_provider"
	TypeWorkspace                       Type = "workspace"
	TypeWorkspaceAssessment             Type = "workspace_assessment"
	TypeWorkspaceVCSProviderLink        Type = "workspace_vcs_provider_link"
)

// String returns the type name as a string.
func (t Type) String() string {
	return string(t)
}

// Parse validates that the identifier is a TRN of this type and returns the parsed TRN.
func (t Type) Parse(value string) (TRN, error) {
	parsed, err := ParseAny(value)
	if err != nil {
		return TRN{}, err
	}

	if parsed.Type() != t {
		return TRN{}, fmt.Errorf("expected TRN type %q, got %q", t, parsed.Type())
	}

	return parsed, nil
}

// Build constructs a TRN string: trn:<type>:<path>.
func (t Type) Build(pathParts ...string) string {
	return trnPrefix + string(t) + ":" + strings.Join(pathParts, "/")
}

// Normalize converts a path, GID, or TRN to the canonical identifier.
// TRNs and GIDs are returned unchanged; bare paths become TRNs.
func (t Type) Normalize(identifier string) string {
	if IsTRN(identifier) {
		return identifier
	}

	// Loosely detect GIDs (base64-encoded "CODE_UUID").
	if decoded, err := base64.RawURLEncoding.DecodeString(identifier); err == nil && strings.Contains(string(decoded), "_") {
		return identifier
	}

	return t.Build(identifier)
}

// TRN holds a parsed Tharsis Resource Name.
type TRN struct {
	typeName   Type
	path       string
	parentPath string // everything before the last "/", empty if single segment
	baseName   string // last segment, or full path if single segment
}

// Type returns the resource type.
func (t TRN) Type() Type { return t.typeName }

// Path returns the resource path.
func (t TRN) Path() string { return t.path }

// ParentPath returns the path with the last segment removed.
// Returns empty string for single-segment paths.
func (t TRN) ParentPath() string { return t.parentPath }

// BaseName returns the last segment of the path.
func (t TRN) BaseName() string { return t.baseName }

// HasParent reports whether the path has more than one segment.
func (t TRN) HasParent() bool { return t.parentPath != "" }

// PathParts returns the resource path split into segments.
func (t TRN) PathParts() []string { return strings.Split(t.path, "/") }

// String returns the full TRN string.
func (t TRN) String() string {
	return trnPrefix + string(t.typeName) + ":" + t.path
}

// ParseAny parses a TRN string of any type into its components.
func ParseAny(value string) (TRN, error) {
	if !IsTRN(value) {
		return TRN{}, fmt.Errorf("not a TRN: %q", value)
	}

	parts := strings.SplitN(value[len(trnPrefix):], ":", 2)
	if len(parts) != 2 {
		return TRN{}, fmt.Errorf("invalid TRN format: %q", value)
	}

	path := parts[1]
	if path == "" || strings.HasPrefix(path, "/") || strings.HasSuffix(path, "/") {
		return TRN{}, fmt.Errorf("invalid TRN resource path: %q", path)
	}

	parentPath, baseName := "", path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		parentPath = path[:i]
		baseName = path[i+1:]
	}

	return TRN{
		typeName:   Type(parts[0]),
		path:       path,
		parentPath: parentPath,
		baseName:   baseName,
	}, nil
}

// MustParseAny is like ParseAny but panics on error.
// Use for values known to be valid (e.g. model.Metadata.TRN).
func MustParseAny(value string) TRN {
	t, err := ParseAny(value)
	if err != nil {
		panic(err)
	}

	return t
}

// IsTRN reports whether value has the "trn:" prefix.
func IsTRN(value string) bool {
	return strings.HasPrefix(value, trnPrefix)
}
