// Package config package
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/qiangxue/go-env"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gopkg.in/yaml.v2"
)

const (
	defaultServerPort                  = "8000"
	envOidcProviderConfigPrefix        = "THARSIS_OAUTH_PROVIDERS_"
	envRunnerConfigPrefix              = "THARSIS_INTERNAL_RUNNERS_"
	defaultMaxGraphQLComplexity        = 0
	defaultRateLimitStorePluginType    = "memory"
	defaultModuleRegistryMaxUploadSize = 1024 * 1024 * 128 // 128 MiB
	defaultVCSRepositorySizeLimit      = 1024 * 1024 * 5   // 5 MebiBytes in bytes.
	defaultAsyncTaskTimeout            = 100               // seconds
	defaultDBAutoMigrateEnabled        = true
	defaultOtelTraceEnabled            = false
	defaultHTTPRateLimit               = 60 // in calls per second
	defaultTerraformCLIVersions        = ">= 1.0.0"
)

// IdpConfig contains the config fields for an Identity Provider
type IdpConfig struct {
	IssuerURL     string `yaml:"issuer_url"`
	ClientID      string `yaml:"client_id"`
	UsernameClaim string `yaml:"username_claim"`
	Scope         string `yaml:"scope"`
	LogoutURL     string `yaml:"logout_url"`
}

// RunnerConfig contains the config fields for a system runner
type RunnerConfig struct {
	JobDispatcherData map[string]string `yaml:"job_dispatcher_data"`
	Name              string            `yaml:"name"`
	JobDispatcherType string            `yaml:"job_dispatcher_type"`
}

// Config represents an application configuration.
type Config struct {
	// Plugin Data
	ObjectStorePluginData    map[string]string `yaml:"object_store_plugin_data"`
	RateLimitStorePluginData map[string]string `yaml:"rate_limit_store_plugin_data" env:"RATE_LIMIT_STORE_PLUGIN_DATA"`
	JWSProviderPluginData    map[string]string `yaml:"jws_provider_plugin_data"`

	// Plugin Typ
	ObjectStorePluginType    string `yaml:"object_store_plugin_type" env:"OBJECT_STORE_PLUGIN_TYPE"`
	RateLimitStorePluginType string `yaml:"rate_limit_store_plugin_type" env:"RATE_LIMIT_STORE_PLUGIN_TYPE"`
	JWSProviderPluginType    string `yaml:"jws_provider_plugin_type" env:"JWS_PROVIDER_PLUGIN_TYPE"`

	// The external facing URL for the Tharsis API
	TharsisAPIURL string `yaml:"tharsis_api_url" env:"API_URL"`

	TLSEnabled bool `yaml:"tls_enabled" env:"TLS_ENABLED"`

	TLSCertFile string `yaml:"tls_cert_file" env:"TLS_CERT_FILE"`

	TLSKeyFile string `yaml:"tls_key_file" env:"TLS_KEY_FILE"`

	// the server port. Defaults to 8000
	ServerPort string `yaml:"server_port" env:"SERVER_PORT"`

	ServiceAccountIssuerURL string `yaml:"service_account_issuer_url" env:"SERVICE_ACCOUNT_ISSUER_URL"`

	// the url for connecting to the database. required.
	DBHost     string `yaml:"db_host" env:"DB_HOST"`
	DBName     string `yaml:"db_name" env:"DB_NAME"`
	DBSSLMode  string `yaml:"db_ssl_mode" env:"DB_SSL_MODE"`
	DBUsername string `yaml:"db_username" env:"DB_USERNAME,secret"`
	DBPassword string `yaml:"db_password" env:"DB_PASSWORD,secret"`

	// TFE Login
	TFELoginClientID string `yaml:"tfe_login_client_id" env:"TFE_LOGIN_CLIENT_ID"`
	TFELoginScopes   string `yaml:"tfe_login_scopes" env:"TFE_LOGIN_SCOPES"`

	// ServiceDiscoveryHost is optional and will default to the API URL host if it's not defined
	ServiceDiscoveryHost string `yaml:"service_discovery_host" env:"SERVICE_DISCOVERY_HOST"`

	// AdminUserEmail is optional and will create a system admin user with this email.
	AdminUserEmail string `yaml:"admin_user_email" env:"ADMIN_USER_EMAIL"`

	// Otel
	OtelTraceType          string `yaml:"otel_trace_type" env:"OTEL_TRACE_TYPE"`
	OtelTraceCollectorHost string `yaml:"otel_trace_host" env:"OTEL_TRACE_HOST"`

	// TerraformCLIVersionConstraint is a comma-separated list of constraints used to limit the returned Terraform CLI versions.
	TerraformCLIVersionConstraint string `yaml:"terraform_cli_version_constraint" env:"TERRAFORM_CLI_VERSION_CONSTRAINT"`

	// The OIDC identity providers
	OauthProviders []IdpConfig `yaml:"oauth_providers"`

	InternalRunners []RunnerConfig `yaml:"internal_runners"`

	// Database Configuration
	DBMaxConnections int `yaml:"db_max_connections" env:"DB_MAX_CONNECTIONS"`
	DBPort           int `yaml:"db_port" env:"DB_PORT"`

	MaxGraphQLComplexity int `yaml:"max_graphql_complexity" env:"MAX_GRAPHQL_COMPLEXITY"`

	// Max upload size when uploading a module to the module registry
	ModuleRegistryMaxUploadSize int `yaml:"module_registry_max_upload_size" env:"MODULE_REGISTRY_MAX_UPLOAD_SIZE"`

	// Timout for async background tasks
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
}

// Validate validates the application configuration.
func (c Config) Validate() error {
	return validation.ValidateStruct(&c,
		validation.Field(&c.ServerPort, is.Port),
		validation.Field(&c.ObjectStorePluginType, validation.Required),
		validation.Field(&c.JWSProviderPluginType, validation.Required),
		validation.Field(&c.TharsisAPIURL, validation.Required),
	)
}

// Load returns an application configuration which is populated from the given configuration file and environment variables.
func Load(file string, logger logger.Logger) (*Config, error) {
	// default config
	c := Config{
		ServerPort:                    defaultServerPort,
		MaxGraphQLComplexity:          defaultMaxGraphQLComplexity,
		RateLimitStorePluginType:      defaultRateLimitStorePluginType,
		ModuleRegistryMaxUploadSize:   defaultModuleRegistryMaxUploadSize,
		VCSRepositorySizeLimit:        defaultVCSRepositorySizeLimit,
		AsyncTaskTimeout:              defaultAsyncTaskTimeout,
		DBAutoMigrateEnabled:          defaultDBAutoMigrateEnabled,
		OtelTraceEnabled:              defaultOtelTraceEnabled,
		HTTPRateLimit:                 defaultHTTPRateLimit,
		TerraformCLIVersionConstraint: defaultTerraformCLIVersions,
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

	if c.ObjectStorePluginData == nil {
		c.ObjectStorePluginData = make(map[string]string)
	}
	if c.RateLimitStorePluginData == nil {
		c.RateLimitStorePluginData = make(map[string]string)
	}

	// Load JWS Provider plugin data
	for k, v := range loadPluginData("THARSIS_JWS_PROVIDER_PLUGIN_DATA_") {
		c.JWSProviderPluginData[k] = v
	}

	// Load Object Store plugin data
	for k, v := range loadPluginData("THARSIS_OBJECT_STORE_PLUGIN_DATA_") {
		c.ObjectStorePluginData[k] = v
	}

	// Load Rate Limiter plugin data
	for k, v := range loadPluginData("THARSIS_RATE_LIMIT_STORE_PLUGIN_DATA_") {
		c.RateLimitStorePluginData[k] = v
	}

	// Default ServiceAccountIssuerURL to TharsisURL
	if c.ServiceAccountIssuerURL == "" {
		c.ServiceAccountIssuerURL = c.TharsisAPIURL
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
			logoutKey := envOidcProviderConfigPrefix + index + "_LOGOUT_URL"

			clientID := os.Getenv(clientIDKey)
			usernameClaim := os.Getenv(usernameClaimKey)
			scope := os.Getenv(scopeKey)
			logoutURL := os.Getenv(logoutKey)

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
				LogoutURL:     logoutURL,
			})
		}
	}
	return idpConfigs, nil
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
