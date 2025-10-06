// Package config package
package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/qiangxue/go-env"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gopkg.in/yaml.v2"
)

const (
	defaultServerPort                                 = "8000"
	envOidcProviderConfigPrefix                       = "THARSIS_OAUTH_PROVIDERS_"
	envRunnerConfigPrefix                             = "THARSIS_INTERNAL_RUNNERS_"
	envFederatedRegistryTrustPolicyName               = "THARSIS_FEDERATED_REGISTRY_TRUST_POLICIES"
	defaultMaxGraphQLComplexity                       = 0
	defaultRateLimitStorePluginType                   = "memory"
	defaultModuleRegistryMaxUploadSize                = 1024 * 1024 * 128 // 128 MiB
	defaultVCSRepositorySizeLimit                     = 1024 * 1024 * 5   // 5 MebiBytes in bytes.
	defaultAsyncTaskTimeout                           = 180               // seconds
	defaultDBAutoMigrateEnabled                       = true
	defaultOtelTraceEnabled                           = false
	defaultHTTPRateLimit                              = 60 // in calls per second
	defaultTerraformCLIVersions                       = ">= 1.0.0"
	defaultWorkspaceAssessmentIntervalHours           = 24
	defaultWorkspaceAssessmentRunLimit                = 20
	defaultUserSessionAccessTokenExpirationMinutes    = 5
	defaultUserSessionRefreshTokenExpirationMinutes   = 60 * 12 // 12 hours
	defaultUserSessionMaxSessionsPerUser              = 20
	defaultAsymmetricSigningKeyDecommissionPeriodDays = 7
)

// IdpConfig contains the config fields for an Identity Provider
type IdpConfig struct {
	IssuerURL     string `yaml:"issuer_url"`
	ClientID      string `yaml:"client_id"`
	UsernameClaim string `yaml:"username_claim"`
	Scope         string `yaml:"scope"`
}

// RunnerConfig contains the config fields for a system runner
type RunnerConfig struct {
	JobDispatcherData map[string]string `yaml:"job_dispatcher_data"`
	Name              string            `yaml:"name"`
	JobDispatcherType string            `yaml:"job_dispatcher_type"`
}

// FederatedRegistryTrustPolicy contains the config fields to allow federated registry access to this Tharsis instance.
type FederatedRegistryTrustPolicy struct {
	IssuerURL         string   `yaml:"issuer_url" json:"issuer_url"`
	Subject           *string  `yaml:"subject" json:"subject"`
	Audience          *string  `yaml:"audience" json:"audience"`
	GroupGlobPatterns []string `yaml:"group_glob_patterns" json:"group_glob_patterns"` // list of groups with potential wildcard in path for access to private modules and providers
}

// Config represents an application configuration.
type Config struct {
	// Plugin Data
	ObjectStorePluginData    map[string]string `yaml:"object_store_plugin_data" sensitive:"true"`
	RateLimitStorePluginData map[string]string `yaml:"rate_limit_store_plugin_data" env:"RATE_LIMIT_STORE_PLUGIN_DATA" sensitive:"true"`
	JWSProviderPluginData    map[string]string `yaml:"jws_provider_plugin_data" sensitive:"true"`
	SecretManagerPluginData  map[string]string `yaml:"secret_manager_plugin_data" sensitive:"true"`
	EmailClientPluginData    map[string]string `yaml:"email_client_plugin_data" sensitive:"true"`

	// Plugin Type
	ObjectStorePluginType    string `yaml:"object_store_plugin_type" env:"OBJECT_STORE_PLUGIN_TYPE"`
	RateLimitStorePluginType string `yaml:"rate_limit_store_plugin_type" env:"RATE_LIMIT_STORE_PLUGIN_TYPE"`
	JWSProviderPluginType    string `yaml:"jws_provider_plugin_type" env:"JWS_PROVIDER_PLUGIN_TYPE"`
	SecretManagerPluginType  string `yaml:"secret_manager_plugin_type" env:"SECRET_MANAGER_PLUGIN_TYPE"`
	EmailClientPluginType    string `yaml:"email_client_plugin_type" env:"EMAIL_CLIENT_PLUGIN_TYPE"`

	DisableSensitiveVariableFeature bool `yaml:"disable_sensitive_variable_feature" env:"DISABLE_SENSITIVE_VARIABLE_FEATURE"`

	EmailFooter string `yaml:"email_footer" env:"EMAIL_FOOTER"`

	// The external facing URL for the Tharsis API
	TharsisAPIURL string `yaml:"tharsis_api_url" env:"API_URL"`

	// The external facing URL for the Tharsis Frontend UI
	TharsisUIURL string `yaml:"tharsis_ui_url" env:"UI_URL"`

	TharsisSupportURL string `yaml:"tharsis_support_url" env:"SUPPORT_URL"`

	TLSEnabled bool `yaml:"tls_enabled" env:"TLS_ENABLED"`

	TLSCertFile string `yaml:"tls_cert_file" env:"TLS_CERT_FILE" sensitive:"true"`

	TLSKeyFile string `yaml:"tls_key_file" env:"TLS_KEY_FILE" sensitive:"true"`

	// the server port. Defaults to 8000
	ServerPort string `yaml:"server_port" env:"SERVER_PORT"`

	JWTIssuerURL string `yaml:"JWT_ISSUER_URL" env:"JWT_ISSUER_URL"`

	// the url for connecting to the database. required.
	DBHost     string `yaml:"db_host" env:"DB_HOST"`
	DBName     string `yaml:"db_name" env:"DB_NAME"`
	DBSSLMode  string `yaml:"db_ssl_mode" env:"DB_SSL_MODE"`
	DBUsername string `yaml:"db_username" env:"DB_USERNAME,secret" sensitive:"true"`
	DBPassword string `yaml:"db_password" env:"DB_PASSWORD,secret" sensitive:"true"`

	// TFE Login
	TFELoginClientID string `yaml:"tfe_login_client_id" env:"TFE_LOGIN_CLIENT_ID"`
	TFELoginScopes   string `yaml:"tfe_login_scopes" env:"TFE_LOGIN_SCOPES"`

	// ServiceDiscoveryHost is optional and will default to the API URL host if it's not defined
	ServiceDiscoveryHost string `yaml:"service_discovery_host" env:"SERVICE_DISCOVERY_HOST"`

	// AdminUserEmail is optional and will create a system admin user with this email.
	AdminUserEmail string `yaml:"admin_user_email" env:"ADMIN_USER_EMAIL" sensitive:"true"`

	// Otel
	OtelTraceType          string `yaml:"otel_trace_type" env:"OTEL_TRACE_TYPE"`
	OtelTraceCollectorHost string `yaml:"otel_trace_host" env:"OTEL_TRACE_HOST"`

	// TerraformCLIVersionConstraint is a comma-separated list of constraints used to limit the returned Terraform CLI versions.
	TerraformCLIVersionConstraint string `yaml:"terraform_cli_version_constraint" env:"TERRAFORM_CLI_VERSION_CONSTRAINT"`

	// The OIDC identity providers
	OauthProviders []IdpConfig `yaml:"oauth_providers"`

	// Federated Registry Trust Policies.
	FederatedRegistryTrustPolicies []FederatedRegistryTrustPolicy `yaml:"federated_registry_trust_policies" sensitive:"true"`

	InternalRunners []RunnerConfig `yaml:"internal_runners" sensitive:"true"`

	// Database Configuration
	DBMaxConnections int `yaml:"db_max_connections" env:"DB_MAX_CONNECTIONS"`
	DBPort           int `yaml:"db_port" env:"DB_PORT"`

	MaxGraphQLComplexity int `yaml:"max_graphql_complexity" env:"MAX_GRAPHQL_COMPLEXITY"`

	// Max upload size when uploading a module to the module registry
	ModuleRegistryMaxUploadSize int `yaml:"module_registry_max_upload_size" env:"MODULE_REGISTRY_MAX_UPLOAD_SIZE"`

	// Timeout for async background tasks
	AsyncTaskTimeout int `yaml:"async_task_timeout" env:"ASYNC_TASK_TIMEOUT"`

	// VCS repository size limit
	VCSRepositorySizeLimit int `yaml:"vcs_repository_size_limit" env:"VCS_REPOSITORY_SIZE_LIMIT"`

	// HTTP rate limit value
	HTTPRateLimit int `yaml:"http_rate_limit" env:"HTTP_RATE_LIMIT"`

	OtelTraceCollectorPort int  `yaml:"otel_trace_port" env:"OTEL_TRACE_PORT"`
	OtelTraceEnabled       bool `yaml:"otel_trace_enabled" env:"OTEL_TRACE_ENABLED"`

	// Enable TFE
	TFELoginEnabled bool `yaml:"tfe_login_enabled" env:"TFE_LOGIN_ENABLED"`

	// Whether to auto migrate the database
	DBAutoMigrateEnabled bool `yaml:"db_auto_migrate_enabled" env:"DB_AUTO_MIGRATE_ENABLED"`

	// WorkspaceAssessmentIntervalHours is the min duration for running workspace assessments
	WorkspaceAssessmentIntervalHours int `yaml:"workspace_assessment_interval_hours" env:"WORKSPACE_ASSESSMENT_INTERVAL_HOURS"`

	// WorkspaceAssessmentRunLimit is the max number of assessment runs that can be created at a time
	WorkspaceAssessmentRunLimit int `yaml:"workspace_assessment_run_limit" env:"WORKSPACE_ASSESSMENT_RUN_LIMIT"`

	// UserSessionAccessTokenExpirationMinutes is the duration in minutes for when a user session access token will expire
	UserSessionAccessTokenExpirationMinutes int `yaml:"user_session_access_token_expiration_minutes" env:"USER_SESSION_ACCESS_TOKEN_EXPIRATION_MINUTES"`

	// UserSessionRefreshTokenExpirationMinutes is the duration in minutes for when a user session refresh token will expire
	UserSessionRefreshTokenExpirationMinutes int `yaml:"user_session_refresh_token_expiration_minutes" env:"USER_SESSION_REFRESH_TOKEN_EXPIRATION_MINUTES"`

	// UserSessionMaxSessionsPerUser is the max number of sessions that a user can have at a time
	UserSessionMaxSessionsPerUser int `yaml:"user_session_max_sessions_per_user" env:"USER_SESSION_MAX_SESSIONS_PER_USER"`

	// CorsAllowedOrigins is a comma delimited list of allowed origins (defaults to the UI URL)
	CorsAllowedOrigins string `yaml:"cors_allowed_origins" env:"CORS_ALLOWED_ORIGINS"`

	// AsymmetricSigningKeyRotationPeriodDays is the number of days after which an asymmetric signing key should be rotated (0 means no rotation)
	AsymmetricSigningKeyRotationPeriodDays int `yaml:"asymmetric_signing_key_rotation_period_days" env:"ASYMMETRIC_SIGNING_KEY_ROTATION_PERIOD_DAYS"`

	// AsymmetricSigningKeyDecommissionPeriodDays is the number of days after which an asymmetric signing key should be decommissioned
	AsymmetricSigningKeyDecommissionPeriodDays int `yaml:"asymmetric_signing_key_decommission_period_days" env:"ASYMMETRIC_SIGNING_KEY_DECOMMISSION_PERIOD_DAYS"`
}

// Validate validates the application configuration.
func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.ServerPort, is.Port),
		validation.Field(&c.ObjectStorePluginType, validation.Required),
		validation.Field(&c.JWSProviderPluginType, validation.Required),
	)
}

// Load returns an application configuration which is populated from the given configuration file and environment variables.
func Load(file string, logger logger.Logger) (*Config, error) {
	// default config
	c := Config{
		ServerPort:                                 defaultServerPort,
		MaxGraphQLComplexity:                       defaultMaxGraphQLComplexity,
		RateLimitStorePluginType:                   defaultRateLimitStorePluginType,
		ModuleRegistryMaxUploadSize:                defaultModuleRegistryMaxUploadSize,
		VCSRepositorySizeLimit:                     defaultVCSRepositorySizeLimit,
		AsyncTaskTimeout:                           defaultAsyncTaskTimeout,
		DBAutoMigrateEnabled:                       defaultDBAutoMigrateEnabled,
		OtelTraceEnabled:                           defaultOtelTraceEnabled,
		HTTPRateLimit:                              defaultHTTPRateLimit,
		TerraformCLIVersionConstraint:              defaultTerraformCLIVersions,
		WorkspaceAssessmentIntervalHours:           defaultWorkspaceAssessmentIntervalHours,
		WorkspaceAssessmentRunLimit:                defaultWorkspaceAssessmentRunLimit,
		UserSessionAccessTokenExpirationMinutes:    defaultUserSessionAccessTokenExpirationMinutes,
		UserSessionRefreshTokenExpirationMinutes:   defaultUserSessionRefreshTokenExpirationMinutes,
		UserSessionMaxSessionsPerUser:              defaultUserSessionMaxSessionsPerUser,
		AsymmetricSigningKeyDecommissionPeriodDays: defaultAsymmetricSigningKeyDecommissionPeriodDays,
	}

	// load from YAML config file
	if file != "" {
		bytes, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		if err = yaml.Unmarshal(bytes, &c); err != nil {
			return nil, fmt.Errorf("failed to parse yaml config file: %w", err)
		}
	}

	// load from environment variables prefixed with "THARSIS_"
	if err := env.New("THARSIS_", logger.Infof).Load(&c); err != nil {
		return nil, fmt.Errorf("failed to load env variables: %w", err)
	}

	if c.TharsisAPIURL == "" {
		c.TharsisAPIURL = fmt.Sprintf("http://localhost:%s", c.ServerPort)
	}

	// Set service discovery host if it's not defined
	if c.ServiceDiscoveryHost == "" {
		apiURL, err := url.Parse(c.TharsisAPIURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL used for THARSIS_API_URL: %v", err)
		}
		c.ServiceDiscoveryHost = apiURL.Host
	}

	// Load OAUTH IDP config from environment is available
	oauthProviders, err := loadOauthConfigFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load oauth provider env variables: %w", err)
	}

	if len(oauthProviders) > 0 {
		c.OauthProviders = oauthProviders
	}

	// Load Federated Registry trust policies from environment if available
	federatedRegistryTrustPolicies, err := loadFederatedRegistryTrustPoliciesFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load federated registry trust policies from env variable: %w", err)
	}

	if len(federatedRegistryTrustPolicies) > 0 {
		c.FederatedRegistryTrustPolicies = federatedRegistryTrustPolicies
	}

	runners, err := loadRunnerConfigFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to load runner env variables %w", err)
	}

	if len(runners) > 0 {
		c.InternalRunners = runners
	}

	if c.JWSProviderPluginData == nil {
		c.JWSProviderPluginData = make(map[string]string)
	}

	if c.SecretManagerPluginData == nil {
		c.SecretManagerPluginData = make(map[string]string)
	}

	if c.ObjectStorePluginData == nil {
		c.ObjectStorePluginData = make(map[string]string)
	}
	if c.RateLimitStorePluginData == nil {
		c.RateLimitStorePluginData = make(map[string]string)
	}

	if c.EmailClientPluginData == nil {
		c.EmailClientPluginData = make(map[string]string)
	}

	// Load JWS Provider plugin data
	for k, v := range loadPluginData("THARSIS_JWS_PROVIDER_PLUGIN_DATA_") {
		c.JWSProviderPluginData[k] = v
	}

	// Load Secret Manager plugin data
	for k, v := range loadPluginData("THARSIS_SECRET_MANAGER_PLUGIN_DATA_") {
		c.SecretManagerPluginData[k] = v
	}

	// Load Object Store plugin data
	for k, v := range loadPluginData("THARSIS_OBJECT_STORE_PLUGIN_DATA_") {
		c.ObjectStorePluginData[k] = v
	}

	// Load Rate Limiter plugin data
	for k, v := range loadPluginData("THARSIS_RATE_LIMIT_STORE_PLUGIN_DATA_") {
		c.RateLimitStorePluginData[k] = v
	}

	// Load Email Client plugin data
	for k, v := range loadPluginData("THARSIS_EMAIL_CLIENT_PLUGIN_DATA_") {
		c.EmailClientPluginData[k] = v
	}

	// Default JWTIssuerURL to TharsisURL
	if c.JWTIssuerURL == "" {
		c.JWTIssuerURL = c.TharsisAPIURL
	}

	// Default TharsisUIURL to the API since we are running the UI from the API
	if c.TharsisUIURL == "" {
		c.TharsisUIURL = c.TharsisAPIURL
	}

	// Default to UI URL
	if c.CorsAllowedOrigins == "" {
		c.CorsAllowedOrigins = c.TharsisUIURL
	}

	// validation
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &c, nil
}

func loadPluginData(envPrefix string) map[string]string {
	pluginData := make(map[string]string)

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		key := pair[0]
		val := pair[1]

		if strings.HasPrefix(key, envPrefix) {
			pluginDataKey := strings.ToLower(key[len(envPrefix):])
			pluginData[pluginDataKey] = val
		}
	}

	return pluginData
}

func loadOauthConfigFromEnvironment() ([]IdpConfig, error) {
	var idpConfigs []IdpConfig
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		key := pair[0]
		val := pair[1]
		if strings.HasPrefix(key, envOidcProviderConfigPrefix) && strings.HasSuffix(key, "_ISSUER_URL") {
			// Build IDP config
			index := key[len(envOidcProviderConfigPrefix) : len(key)-len("_ISSUER_URL")]
			issuerURL := val

			clientIDKey := envOidcProviderConfigPrefix + index + "_CLIENT_ID"
			usernameClaimKey := envOidcProviderConfigPrefix + index + "_USERNAME_CLAIM"
			scopeKey := envOidcProviderConfigPrefix + index + "_SCOPE"

			clientID := os.Getenv(clientIDKey)
			usernameClaim := os.Getenv(usernameClaimKey)
			scope := os.Getenv(scopeKey)

			if clientID == "" {
				return nil, errors.New(clientIDKey + " environment variable is required")
			}

			if usernameClaim == "" {
				usernameClaim = "sub"
			}

			idpConfigs = append(idpConfigs, IdpConfig{
				IssuerURL:     issuerURL,
				ClientID:      clientID,
				UsernameClaim: usernameClaim,
				Scope:         scope,
			})
		}
	}
	return idpConfigs, nil
}

func loadFederatedRegistryTrustPoliciesFromEnvironment() ([]FederatedRegistryTrustPolicy, error) {
	var federatedRegistryTrustPolicies []FederatedRegistryTrustPolicy

	val, ok := os.LookupEnv(envFederatedRegistryTrustPolicyName)
	if ok {
		if err := json.Unmarshal([]byte(val), &federatedRegistryTrustPolicies); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal federated registry trust policies")
		}
	}

	return federatedRegistryTrustPolicies, nil
}

func loadRunnerConfigFromEnvironment() ([]RunnerConfig, error) {
	var runnerConfigs []RunnerConfig
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		key := pair[0]
		val := pair[1]
		if strings.HasPrefix(key, envRunnerConfigPrefix) && strings.HasSuffix(key, "_NAME") {
			// Build runner config
			index := key[len(envRunnerConfigPrefix) : len(key)-len("_NAME")]
			name := val

			dispatcherTypeKey := envRunnerConfigPrefix + index + "_JOB_DISPATCHER_TYPE"
			dispatcherDataKey := envRunnerConfigPrefix + index + "_JOB_DISPATCHER_DATA_"

			dispatcherType := os.Getenv(dispatcherTypeKey)

			if dispatcherType == "" {
				return nil, errors.New(dispatcherTypeKey + " environment variable is required")
			}

			jobDispatcherData := make(map[string]string)

			// Load Job Dispatcher plugin data
			for k, v := range loadPluginData(dispatcherDataKey) {
				jobDispatcherData[k] = v
			}

			runnerConfigs = append(runnerConfigs, RunnerConfig{
				Name:              name,
				JobDispatcherType: dispatcherType,
				JobDispatcherData: jobDispatcherData,
			})
		}
	}
	return runnerConfigs, nil
}
