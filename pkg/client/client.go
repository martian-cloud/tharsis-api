// Package client contains a client implementation for interfacing with the Tharsis server
package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Retry configuration with maximum of 4 attempts. Add more services and methods under 'name'.
// It will only retry for UNAVAILABLE and UNKNOWN status codes.
// See: https://github.com/grpc/grpc-go/blob/master/examples/features/retry/README.md#define-your-retry-policy
const retryPolicy = `{
	"methodConfig": [{
		"name": [
			{"service": "martiancloud.tharsis.api.auth_settings.AuthSettings"},
			{"service": "martiancloud.tharsis.api.configuration_version.ConfigurationVersions"},
			{"service": "martiancloud.tharsis.api.gpg_key.GPGKeys"},
			{"service": "martiancloud.tharsis.api.group.Groups"},
			{"service": "martiancloud.tharsis.api.job.Jobs"},
			{"service": "martiancloud.tharsis.api.managed_identity.ManagedIdentities"},
			{"service": "martiancloud.tharsis.api.namespace_membership.NamespaceMemberships"},
			{"service": "martiancloud.tharsis.api.namespace_variable.NamespaceVariables"},
			{"service": "martiancloud.tharsis.api.resource_limit.ResourceLimits"},
			{"service": "martiancloud.tharsis.api.role.Roles"},
			{"service": "martiancloud.tharsis.api.run.Runs"},
			{"service": "martiancloud.tharsis.api.runner.Runners"},
			{"service": "martiancloud.tharsis.api.service_account.ServiceAccounts"},
			{"service": "martiancloud.tharsis.api.state_version.StateVersions"},
			{"service": "martiancloud.tharsis.api.team.Teams"},
			{"service": "martiancloud.tharsis.api.terraform_module.TerraformModules"},
			{"service": "martiancloud.tharsis.api.terraform_provider.TerraformProviders"},
			{"service": "martiancloud.tharsis.api.terraform_provider_mirror.TerraformProviderMirrors"},
			{"service": "martiancloud.tharsis.api.user.Users"},
			{"service": "martiancloud.tharsis.api.vcs_provider.VCSProviders"},
			{"service": "martiancloud.tharsis.api.version.Version"},
			{"service": "martiancloud.tharsis.api.workspace.Workspaces"}
		],

		"waitForReady": true,

		"retryPolicy": {
			"MaxAttempts": 4,
			"InitialBackoff": ".5s",
			"MaxBackoff": "30s",
			"BackoffMultiplier": 2,
			"RetryableStatusCodes": [ "UNAVAILABLE", "ABORTED", "UNKNOWN" ]
		}
	}]
}`

// contextCredentials implements the credentials.PerRPCCredentials interface and
// allows us to use a token passed into the RPC request metadata.
type contextCredentials struct {
	getter TokenGetter
}

// GetRequestMetadata sets the token on the context metadata.
func (c *contextCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	newToken, err := c.getter.Token(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": string(newToken),
	}, nil
}

// RequireTransportSecurity indicates if transport security is required.
func (c *contextCredentials) RequireTransportSecurity() bool {
	// If true, it won't be possible to use token auth without TLS (locally).
	return false
}

// TokenGetter is an interface for retrieving and renewing a service account token.
type TokenGetter interface {
	Token(ctx context.Context) (string, error)
}

// LeveledLogger is an interface that can be implemented by any logger or a
// logger wrapper to provide leveled logging. The methods accept a message
// string and a variadic number of key-value pairs. For log.Printf style
// formatting where message string contains a format specifier, use Logger
// interface.
// This interface has been copied from the retryablehttp package.
type LeveledLogger interface {
	Error(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
}

// Client is the gateway to interact with the Tharsis API.
type Client struct {
	connection                     *grpc.ClientConn
	AuthSettingsClient             pb.AuthSettingsClient
	ConfigurationVersionsClient    pb.ConfigurationVersionsClient
	GPGKeysClient                  pb.GPGKeysClient
	GroupsClient                   pb.GroupsClient
	JobsClient                     pb.JobsClient
	ManagedIdentitiesClient        pb.ManagedIdentitiesClient
	NamespaceMembershipsClient     pb.NamespaceMembershipsClient
	NamespaceVariablesClient       pb.NamespaceVariablesClient
	ResourceLimitsClient           pb.ResourceLimitsClient
	RolesClient                    pb.RolesClient
	RunsClient                     pb.RunsClient
	RunnersClient                  pb.RunnersClient
	ServiceAccountsClient          pb.ServiceAccountsClient
	StateVersionsClient            pb.StateVersionsClient
	TeamsClient                    pb.TeamsClient
	TerraformModulesClient         pb.TerraformModulesClient
	TerraformProvidersClient       pb.TerraformProvidersClient
	TerraformProviderMirrorsClient pb.TerraformProviderMirrorsClient
	UsersClient                    pb.UsersClient
	VCSProvidersClient             pb.VCSProvidersClient
	VersionClient                  pb.VersionClient
	WorkspacesClient               pb.WorkspacesClient
}

// Config is used to configure the client
type Config struct {
	TokenGetter   TokenGetter
	HTTPEndpoint  string
	TLSSkipVerify bool
	Logger        LeveledLogger
	UserAgent     string
}

// New returns a new Client struct.
func New(ctx context.Context, c *Config) (*Client, error) {
	dialOptions := []grpc.DialOption{
		// Set Retry policy.
		grpc.WithDefaultServiceConfig(retryPolicy),
		// Configure keepalive
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			// After a duration of this time if the client doesn't see any activity it pings the server to see if the transport is still alive.
			Time: 1 * time.Minute,
			// After having pinged for keepalive check, the client waits for a duration of Timeout before deciding it is dead.
			Timeout: 15 * time.Second,
			// Send keepalive pings when there are no active RPCs.
			PermitWithoutStream: true,
		}),
	}

	if c.UserAgent != "" {
		// Add user agent if specified
		dialOptions = append(dialOptions, grpc.WithUserAgent(c.UserAgent))
	}

	if c.TokenGetter != nil {
		// Add token based auth since we're using it.
		dialOptions = append(
			dialOptions,
			grpc.WithPerRPCCredentials(&contextCredentials{
				getter: c.TokenGetter,
			}),
		)
	}

	if c.TLSSkipVerify {
		// Override the default transport to skip TLS verification.
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12} // #nosec G402 -- This is used in development
	}

	// Fetch discovery document from API.
	discoveryDocument, err := NewGRPCDiscoveryDocument(ctx, c.HTTPEndpoint, WithLogger(c.Logger))
	if err != nil && errors.Is(err, context.DeadlineExceeded) {
		// The context deadline was exceeded, try one more time before returning an error
		discoveryDocument, err = NewGRPCDiscoveryDocument(ctx, c.HTTPEndpoint, WithLogger(c.Logger))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery document: %w", err)
	}

	if !discoveryDocument.HasTransportSecurity() {
		// Disable TLS since it isn't being used on this connection.
		dialOptions = append(
			dialOptions,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
	} else if c.TLSSkipVerify {
		// Use TLS but don't verify server's certificate chain and hostname. For testing only.
		dialOptions = append(
			dialOptions,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})), // #nosec G402 -- This is used in development
		)
	} else {
		// Use TLS by default.
		dialOptions = append(
			dialOptions,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})),
		)
	}

	if c.Logger != nil {
		c.Logger.Info("dialing GRPC connection", "host", discoveryDocument.Host, "port", discoveryDocument.Port)
	}

	clientConn, err := grpc.NewClient(
		fmt.Sprintf("%s:%s", discoveryDocument.Host, discoveryDocument.Port),
		dialOptions...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC endpoint: %w", err)
	}

	if c.Logger != nil {
		c.Logger.Info("successfully initialized GRPC connection")
	}

	return &Client{
		connection:                     clientConn,
		AuthSettingsClient:             pb.NewAuthSettingsClient(clientConn),
		ConfigurationVersionsClient:    pb.NewConfigurationVersionsClient(clientConn),
		GPGKeysClient:                  pb.NewGPGKeysClient(clientConn),
		GroupsClient:                   pb.NewGroupsClient(clientConn),
		JobsClient:                     pb.NewJobsClient(clientConn),
		ManagedIdentitiesClient:        pb.NewManagedIdentitiesClient(clientConn),
		NamespaceMembershipsClient:     pb.NewNamespaceMembershipsClient(clientConn),
		NamespaceVariablesClient:       pb.NewNamespaceVariablesClient(clientConn),
		ResourceLimitsClient:           pb.NewResourceLimitsClient(clientConn),
		RolesClient:                    pb.NewRolesClient(clientConn),
		RunsClient:                     pb.NewRunsClient(clientConn),
		RunnersClient:                  pb.NewRunnersClient(clientConn),
		ServiceAccountsClient:          pb.NewServiceAccountsClient(clientConn),
		StateVersionsClient:            pb.NewStateVersionsClient(clientConn),
		TeamsClient:                    pb.NewTeamsClient(clientConn),
		TerraformModulesClient:         pb.NewTerraformModulesClient(clientConn),
		TerraformProvidersClient:       pb.NewTerraformProvidersClient(clientConn),
		TerraformProviderMirrorsClient: pb.NewTerraformProviderMirrorsClient(clientConn),
		UsersClient:                    pb.NewUsersClient(clientConn),
		VCSProvidersClient:             pb.NewVCSProvidersClient(clientConn),
		VersionClient:                  pb.NewVersionClient(clientConn),
		WorkspacesClient:               pb.NewWorkspacesClient(clientConn),
	}, nil
}

// Close closes any underlying connections for this client.
func (c *Client) Close() error {
	return c.connection.Close()
}

// ServiceDiscoveryPath is the path to the service discovery document.
const ServiceDiscoveryPath = "/.well-known/tharsis.json"

// GRPCDiscoveryDocument represents the contents of the GRPC discovery document.
type GRPCDiscoveryDocument struct {
	Host              string `json:"host"`
	TransportSecurity string `json:"transport_security"`
	Port              string `json:"port"`
}

// grpcDiscoveryOptions represents the options for the GRPC discovery document.
type grpcDiscoveryOptions struct {
	logger LeveledLogger
}

// GRPCDiscoveryOption is a function that sets options for the GRPC discovery document.
type GRPCDiscoveryOption func(*grpcDiscoveryOptions)

// WithLogger sets the logger for the GRPC discovery document.
func WithLogger(logger LeveledLogger) GRPCDiscoveryOption {
	return func(o *grpcDiscoveryOptions) {
		o.logger = logger
	}
}

// NewGRPCDiscoveryDocument returns a new GRPC discovery document.
// The HTTP get request of the discovery document is done in this function.
func NewGRPCDiscoveryDocument(ctx context.Context, endpoint string, options ...GRPCDiscoveryOption) (*GRPCDiscoveryDocument, error) {
	opts := &grpcDiscoveryOptions{}
	for _, o := range options {
		o(opts)
	}

	discoveryURL, err := url.JoinPath(endpoint, ServiceDiscoveryPath)
	if err != nil {
		return nil, err
	}

	fetchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := retryablehttp.NewRequestWithContext(fetchCtx, http.MethodGet, discoveryURL, nil) // nosemgrep: gosec.G107-1
	if err != nil {
		return nil, err
	}

	retryableClient := &retryablehttp.Client{
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout: 60 * time.Second,
				}).Dial,
				DisableKeepAlives:   true,
				TLSHandshakeTimeout: 30 * time.Second,
			},
		},
		RetryWaitMin: 1 * time.Second,
		RetryWaitMax: 30 * time.Second,
		RetryMax:     5,
		CheckRetry:   retryablehttp.DefaultRetryPolicy,
		Backoff:      retryablehttp.DefaultBackoff,
		Logger:       opts.logger,
	}

	resp, err := retryableClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received status code %d from well-known URL", resp.StatusCode)
	}

	var discoveryDocument struct {
		GRPCDiscoveryDocument *GRPCDiscoveryDocument `json:"grpc"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&discoveryDocument); err != nil {
		return nil, err
	}

	return discoveryDocument.GRPCDiscoveryDocument, nil
}

// HasTransportSecurity returns true if the GRPC endpoint has transport security enabled.
func (d *GRPCDiscoveryDocument) HasTransportSecurity() bool {
	return d.TransportSecurity != "plaintext"
}
