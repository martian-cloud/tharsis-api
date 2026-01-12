package resolver

import (
	"context"
	"reflect"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
)

// ConfigResolver resolves the API config
type ConfigResolver struct {
	cfg *config.Config
}

// ServerPort resolver
func (r *ConfigResolver) ServerPort() string {
	return r.cfg.ServerPort
}

// TharsisAPIURL resolver
func (r *ConfigResolver) TharsisAPIURL() string {
	return r.cfg.TharsisAPIURL
}

// TharsisUIURL resolver
func (r *ConfigResolver) TharsisUIURL() string {
	return r.cfg.TharsisUIURL
}

// TharsisSupportURL resolver
func (r *ConfigResolver) TharsisSupportURL() string {
	return r.cfg.TharsisSupportURL
}

// TLSEnabled resolver
func (r *ConfigResolver) TLSEnabled() bool {
	return r.cfg.TLSEnabled
}

// JWTIssuerURL resolver
func (r *ConfigResolver) JWTIssuerURL() string {
	return r.cfg.JWTIssuerURL
}

// DBHost resolver
func (r *ConfigResolver) DBHost() string {
	return r.cfg.DBHost
}

// DBName resolver
func (r *ConfigResolver) DBName() string {
	return r.cfg.DBName
}

// DBSSLMode resolver
func (r *ConfigResolver) DBSSLMode() string {
	return r.cfg.DBSSLMode
}

// DBPort resolver
func (r *ConfigResolver) DBPort() int32 {
	return int32(r.cfg.DBPort)
}

// DBMaxConnections resolver
func (r *ConfigResolver) DBMaxConnections() int32 {
	return int32(r.cfg.DBMaxConnections)
}

// DBAutoMigrateEnabled resolver
func (r *ConfigResolver) DBAutoMigrateEnabled() bool {
	return r.cfg.DBAutoMigrateEnabled
}

// CLILoginOIDCClientID resolver
func (r *ConfigResolver) CLILoginOIDCClientID() *string {
	if r.cfg.CLILoginOIDCClientID == "" {
		return nil
	}
	return &r.cfg.CLILoginOIDCClientID
}

// CLILoginOIDCScopes resolver
func (r *ConfigResolver) CLILoginOIDCScopes() *string {
	if r.cfg.CLILoginOIDCScopes == "" {
		return nil
	}
	return &r.cfg.CLILoginOIDCScopes
}

// OIDCInternalIdentityProviderClientID resolver
func (r *ConfigResolver) OIDCInternalIdentityProviderClientID() string {
	return r.cfg.OIDCInternalIdentityProviderClientID
}

// MaxGraphQLComplexity resolver
func (r *ConfigResolver) MaxGraphQLComplexity() int32 {
	return int32(r.cfg.MaxGraphQLComplexity)
}

// ModuleRegistryMaxUploadSize resolver
func (r *ConfigResolver) ModuleRegistryMaxUploadSize() int32 {
	return int32(r.cfg.ModuleRegistryMaxUploadSize)
}

// AsyncTaskTimeout resolver
func (r *ConfigResolver) AsyncTaskTimeout() int32 {
	return int32(r.cfg.AsyncTaskTimeout)
}

// VCSRepositorySizeLimit resolver
func (r *ConfigResolver) VCSRepositorySizeLimit() int32 {
	return int32(r.cfg.VCSRepositorySizeLimit)
}

// HTTPRateLimit resolver
func (r *ConfigResolver) HTTPRateLimit() int32 {
	return int32(r.cfg.HTTPRateLimit)
}

// OtelTraceEnabled resolver
func (r *ConfigResolver) OtelTraceEnabled() bool {
	return r.cfg.OtelTraceEnabled
}

// OtelTraceType resolver
func (r *ConfigResolver) OtelTraceType() *string {
	if r.cfg.OtelTraceType == "" {
		return nil
	}
	return &r.cfg.OtelTraceType
}

// OtelTraceCollectorHost resolver
func (r *ConfigResolver) OtelTraceCollectorHost() *string {
	if r.cfg.OtelTraceCollectorHost == "" {
		return nil
	}
	return &r.cfg.OtelTraceCollectorHost
}

// OtelTraceCollectorPort resolver
func (r *ConfigResolver) OtelTraceCollectorPort() int32 {
	return int32(r.cfg.OtelTraceCollectorPort)
}

// ServiceDiscoveryHost resolver
func (r *ConfigResolver) ServiceDiscoveryHost() string {
	return r.cfg.ServiceDiscoveryHost
}

// TerraformCLIVersionConstraint resolver
func (r *ConfigResolver) TerraformCLIVersionConstraint() string {
	return r.cfg.TerraformCLIVersionConstraint
}

// WorkspaceAssessmentIntervalHours resolver
func (r *ConfigResolver) WorkspaceAssessmentIntervalHours() int32 {
	return int32(r.cfg.WorkspaceAssessmentIntervalHours)
}

// WorkspaceAssessmentRunLimit resolver
func (r *ConfigResolver) WorkspaceAssessmentRunLimit() int32 {
	return int32(r.cfg.WorkspaceAssessmentRunLimit)
}

// UserSessionAccessTokenExpirationMinutes resolver
func (r *ConfigResolver) UserSessionAccessTokenExpirationMinutes() int32 {
	return int32(r.cfg.UserSessionAccessTokenExpirationMinutes)
}

// UserSessionRefreshTokenExpirationMinutes resolver
func (r *ConfigResolver) UserSessionRefreshTokenExpirationMinutes() int32 {
	return int32(r.cfg.UserSessionRefreshTokenExpirationMinutes)
}

// UserSessionMaxSessionsPerUser resolver
func (r *ConfigResolver) UserSessionMaxSessionsPerUser() int32 {
	return int32(r.cfg.UserSessionMaxSessionsPerUser)
}

// CorsAllowedOrigins resolver
func (r *ConfigResolver) CorsAllowedOrigins() string {
	return r.cfg.CorsAllowedOrigins
}

// AsymmetricSigningKeyRotationPeriodDays resolver
func (r *ConfigResolver) AsymmetricSigningKeyRotationPeriodDays() int32 {
	return int32(r.cfg.AsymmetricSigningKeyRotationPeriodDays)
}

// AsymmetricSigningKeyDecommissionPeriodDays resolver
func (r *ConfigResolver) AsymmetricSigningKeyDecommissionPeriodDays() int32 {
	return int32(r.cfg.AsymmetricSigningKeyDecommissionPeriodDays)
}

// DisableSensitiveVariableFeature resolver
func (r *ConfigResolver) DisableSensitiveVariableFeature() bool {
	return r.cfg.DisableSensitiveVariableFeature
}

// EmailFooter resolver
func (r *ConfigResolver) EmailFooter() *string {
	if r.cfg.EmailFooter == "" {
		return nil
	}
	return &r.cfg.EmailFooter
}

// ObjectStorePluginType resolver
func (r *ConfigResolver) ObjectStorePluginType() string {
	return r.cfg.ObjectStorePluginType
}

// RateLimitStorePluginType resolver
func (r *ConfigResolver) RateLimitStorePluginType() string {
	return r.cfg.RateLimitStorePluginType
}

// JWSProviderPluginType resolver
func (r *ConfigResolver) JWSProviderPluginType() string {
	return r.cfg.JWSProviderPluginType
}

// SecretManagerPluginType resolver
func (r *ConfigResolver) SecretManagerPluginType() string {
	return r.cfg.SecretManagerPluginType
}

// EmailClientPluginType resolver
func (r *ConfigResolver) EmailClientPluginType() string {
	return r.cfg.EmailClientPluginType
}

// TLSCertFile resolver
func (r *ConfigResolver) TLSCertFile() string {
	return r.cfg.TLSCertFile
}

// TLSKeyFile resolver
func (r *ConfigResolver) TLSKeyFile() string {
	return r.cfg.TLSKeyFile
}

// AdminUserEmail resolver
func (r *ConfigResolver) AdminUserEmail() string {
	return r.cfg.AdminUserEmail
}

// IdpConfigResolver resolves IDP config
type IdpConfigResolver struct {
	cfg *config.IdpConfig
}

// OauthProviders resolver
func (r *ConfigResolver) OauthProviders() []*IdpConfigResolver {
	var resolvers []*IdpConfigResolver
	for i := range r.cfg.OauthProviders {
		resolvers = append(resolvers, &IdpConfigResolver{cfg: &r.cfg.OauthProviders[i]})
	}
	return resolvers
}

// FederatedRegistryTrustPolicyResolver resolves federated registry trust policy
type FederatedRegistryTrustPolicyResolver struct {
	cfg *config.FederatedRegistryTrustPolicy
}

// FederatedRegistryTrustPolicies resolver
func (r *ConfigResolver) FederatedRegistryTrustPolicies() []*FederatedRegistryTrustPolicyResolver {
	var resolvers []*FederatedRegistryTrustPolicyResolver
	for i := range r.cfg.FederatedRegistryTrustPolicies {
		resolvers = append(resolvers, &FederatedRegistryTrustPolicyResolver{cfg: &r.cfg.FederatedRegistryTrustPolicies[i]})
	}
	return resolvers
}

// RunnerConfigResolver resolves runner config
type RunnerConfigResolver struct {
	cfg *config.RunnerConfig
}

// InternalRunners resolver
func (r *ConfigResolver) InternalRunners() []*RunnerConfigResolver {
	var resolvers []*RunnerConfigResolver
	for i := range r.cfg.InternalRunners {
		resolvers = append(resolvers, &RunnerConfigResolver{cfg: &r.cfg.InternalRunners[i]})
	}
	return resolvers
}

// PluginDataEntryResolver resolves plugin data entries
type PluginDataEntryResolver struct {
	Key   string
	Value string
}

// ObjectStorePluginData resolver
func (r *ConfigResolver) ObjectStorePluginData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.ObjectStorePluginData)
}

// RateLimitStorePluginData resolver
func (r *ConfigResolver) RateLimitStorePluginData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.RateLimitStorePluginData)
}

// JWSProviderPluginData resolver
func (r *ConfigResolver) JWSProviderPluginData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.JWSProviderPluginData)
}

// SecretManagerPluginData resolver
func (r *ConfigResolver) SecretManagerPluginData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.SecretManagerPluginData)
}

// EmailClientPluginData resolver
func (r *ConfigResolver) EmailClientPluginData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.EmailClientPluginData)
}

// MCPServerConfig resolver
func (r *ConfigResolver) MCPServerConfig() *MCPServerConfigResolver {
	return &MCPServerConfigResolver{cfg: &r.cfg.MCPServerConfig}
}

// MCPServerConfigResolver resolves MCP server config
type MCPServerConfigResolver struct {
	cfg *config.MCPServerConfig
}

// EnabledToolsets resolver
func (r *MCPServerConfigResolver) EnabledToolsets() []string {
	return strings.Split(r.cfg.EnabledToolsets, ",")
}

// EnabledTools resolver
func (r *MCPServerConfigResolver) EnabledTools() []string {
	return strings.Split(r.cfg.EnabledTools, ",")
}

// ReadOnly resolver
func (r *MCPServerConfigResolver) ReadOnly() bool {
	return r.cfg.ReadOnly
}

// IssuerURL resolver
func (r *IdpConfigResolver) IssuerURL() string {
	return r.cfg.IssuerURL
}

// ClientID resolver
func (r *IdpConfigResolver) ClientID() string {
	return r.cfg.ClientID
}

// UsernameClaim resolver
func (r *IdpConfigResolver) UsernameClaim() string {
	return r.cfg.UsernameClaim
}

// Scope resolver
func (r *IdpConfigResolver) Scope() string {
	return r.cfg.Scope
}

// Name resolver
func (r *RunnerConfigResolver) Name() string {
	return r.cfg.Name
}

// JobDispatcherType resolver
func (r *RunnerConfigResolver) JobDispatcherType() string {
	return r.cfg.JobDispatcherType
}

// JobDispatcherData resolver
func (r *RunnerConfigResolver) JobDispatcherData() []*PluginDataEntryResolver {
	return mapToPluginDataEntries(r.cfg.JobDispatcherData)
}

// IssuerURL resolver
func (r *FederatedRegistryTrustPolicyResolver) IssuerURL() string {
	return r.cfg.IssuerURL
}

// Subject resolver
func (r *FederatedRegistryTrustPolicyResolver) Subject() *string {
	return r.cfg.Subject
}

// Audience resolver
func (r *FederatedRegistryTrustPolicyResolver) Audience() *string {
	return r.cfg.Audience
}

// GroupGlobPatterns resolver
func (r *FederatedRegistryTrustPolicyResolver) GroupGlobPatterns() []string {
	return r.cfg.GroupGlobPatterns
}

// Helper functions
func mapToPluginDataEntries(data map[string]string) []*PluginDataEntryResolver {
	var entries []*PluginDataEntryResolver
	for k, v := range data {
		entries = append(entries, &PluginDataEntryResolver{Key: k, Value: v})
	}
	return entries
}

func configQuery(ctx context.Context) (*ConfigResolver, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	config := getConfig(ctx)

	if !caller.IsAdmin() {
		config = filterSensitiveFields(*config)
	}

	return &ConfigResolver{
		cfg: config,
	}, nil
}

func filterSensitiveFields(cfg config.Config) *config.Config {
	// Create a copy of the config
	filtered := cfg
	v := reflect.ValueOf(&filtered).Elem()
	t := reflect.TypeOf(filtered)

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)

		// Check if field is marked as sensitive
		if field.Tag.Get("sensitive") == "true" {
			fieldValue := v.Field(i)
			if fieldValue.CanSet() {
				switch fieldValue.Kind() {
				case reflect.String:
					fieldValue.SetString("***")
				case reflect.Map:
					fieldValue.Set(reflect.MakeMap(fieldValue.Type()))
				case reflect.Slice:
					fieldValue.Set(reflect.MakeSlice(fieldValue.Type(), 0, 0))
				}
			}
		}
	}

	return &filtered
}
