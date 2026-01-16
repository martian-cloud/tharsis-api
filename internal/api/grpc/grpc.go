// Package grpc implements gRPC functionality.
package grpc

import (
	"net"
	"net/http"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/grpc/interceptors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/grpc/servers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	log "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ServerOptions contains the options to configure the gRPC server.
type ServerOptions struct {
	Listener        net.Listener
	Logger          log.Logger
	Authenticator   auth.Authenticator
	APIServerConfig *config.Config
	ServiceCatalog  *services.Catalog
	OAuthProviders  []config.IdpConfig
	RateLimitStore  ratelimitstore.Store
}

// Server implements functions needed to configure, start and stop the gRPC server.
type Server struct {
	server  *grpc.Server
	options *ServerOptions
}

// NewServer creates a new gRPC server.
func NewServer(options *ServerOptions) *Server {
	opts := []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainStreamInterceptor(
			interceptors.RequestIDStream(),
			interceptors.UserAgentStream(),
			interceptors.ErrorHandlerStream(options.Logger),
			interceptors.AuthenticationStream(options.Authenticator),
			interceptors.SubjectStream(),
			interceptors.RateLimiterStream(options.RateLimitStore),
		),
		grpc.ChainUnaryInterceptor(
			interceptors.RequestIDUnary(),
			interceptors.UserAgentUnary(),
			interceptors.ErrorHandlerUnary(options.Logger),
			interceptors.AuthenticationUnary(options.Authenticator),
			interceptors.SubjectUnary(),
			interceptors.RateLimiterUnary(options.RateLimitStore),
		),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			// Max time that a connection can exist without any active RPCs
			MaxConnectionIdle: 1 * time.Hour,
			// After this time the server will ping the client to determine if it's still alive
			Time: 1 * time.Minute,
			// The amount of time the server will wait for the client to respond to a ping
			Timeout: 30 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			// Minumum time a client should wait before sending a keepalive ping.
			// IMPORTANT: This value should be less than the client's keepalive time.
			MinTime: 30 * time.Second,
			// Permit keepalive pings without active streams.
			PermitWithoutStream: true,
		}),
	}

	s := grpc.NewServer(opts...)

	// Create server instances.
	var (
		authSettingsServer            = servers.NewAuthSettingsServer(options.OAuthProviders)
		configurationVersionServer    = servers.NewConfigurationVersionServer(options.ServiceCatalog)
		gpgKeyServer                  = servers.NewGPGKeyServer(options.ServiceCatalog)
		groupServer                   = servers.NewGroupServer(options.ServiceCatalog)
		jobServer                     = servers.NewJobServer(options.ServiceCatalog)
		managedIdentityServer         = servers.NewManagedIdentityServer(options.ServiceCatalog)
		namespaceMembershipServer     = servers.NewNamespaceMembershipServer(options.ServiceCatalog)
		namespaceVariableServer       = servers.NewNamespaceVariableServer(options.ServiceCatalog)
		resourceLimitServer           = servers.NewResourceLimitServer(options.ServiceCatalog)
		roleServer                    = servers.NewRoleServer(options.ServiceCatalog)
		runServer                     = servers.NewRunServer(options.ServiceCatalog)
		runnerServer                  = servers.NewRunnerServer(options.ServiceCatalog)
		serviceAccountServer          = servers.NewServiceAccountServer(options.ServiceCatalog)
		stateVersionServer            = servers.NewStateVersionServer(options.ServiceCatalog)
		teamServer                    = servers.NewTeamServer(options.ServiceCatalog)
		terraformModuleServer         = servers.NewTerraformModuleServer(options.ServiceCatalog)
		terraformProviderServer       = servers.NewTerraformProviderServer(options.ServiceCatalog)
		terraformProviderMirrorServer = servers.NewTerraformProviderMirrorServer(options.ServiceCatalog)
		userServer                    = servers.NewUserServer(options.ServiceCatalog)
		vcsProviderServer             = servers.NewVCSProviderServer(options.ServiceCatalog)
		versionServer                 = servers.NewVersionServer(options.ServiceCatalog.VersionService)
		workspaceServer               = servers.NewWorkspaceServer(options.ServiceCatalog)
	)

	// Register servers.
	pb.RegisterAuthSettingsServer(s, authSettingsServer)
	pb.RegisterConfigurationVersionsServer(s, configurationVersionServer)
	pb.RegisterGPGKeysServer(s, gpgKeyServer)
	pb.RegisterGroupsServer(s, groupServer)
	pb.RegisterJobsServer(s, jobServer)
	pb.RegisterManagedIdentitiesServer(s, managedIdentityServer)
	pb.RegisterNamespaceMembershipsServer(s, namespaceMembershipServer)
	pb.RegisterNamespaceVariablesServer(s, namespaceVariableServer)
	pb.RegisterResourceLimitsServer(s, resourceLimitServer)
	pb.RegisterRolesServer(s, roleServer)
	pb.RegisterRunsServer(s, runServer)
	pb.RegisterRunnersServer(s, runnerServer)
	pb.RegisterServiceAccountsServer(s, serviceAccountServer)
	pb.RegisterStateVersionsServer(s, stateVersionServer)
	pb.RegisterTeamsServer(s, teamServer)
	pb.RegisterTerraformModulesServer(s, terraformModuleServer)
	pb.RegisterTerraformProvidersServer(s, terraformProviderServer)
	pb.RegisterTerraformProviderMirrorsServer(s, terraformProviderMirrorServer)
	pb.RegisterUsersServer(s, userServer)
	pb.RegisterVCSProvidersServer(s, vcsProviderServer)
	pb.RegisterVersionServer(s, versionServer)
	pb.RegisterWorkspacesServer(s, workspaceServer)

	// Enable reflection which makes it easier to use grpcui and alike
	// without needing to import proto files.
	reflection.Register(s)

	return &Server{
		options: options,
		server:  s,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.ServeHTTP(w, r)
}

// Start starts the server.
func (s *Server) Start() error {
	s.options.Logger.Infof("gRPC server listening on %s", s.options.Listener.Addr())
	return s.server.Serve(s.options.Listener)
}

// Shutdown gracefully stops the server. Will block until
// all pending RPCs are finished.
func (s *Server) Shutdown() {
	s.options.Logger.Info("Gracefully shutting down gRPC server")
	s.server.GracefulStop()
	s.options.Logger.Info("Successfully shutdown gRPC server")
}

// IsGRPCRequest returns true if this is a GRPC request
func IsGRPCRequest(r *http.Request) bool {
	return r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc")
}
