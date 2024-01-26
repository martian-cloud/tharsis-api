// Package apiserver is used to initialize the api
package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "gitlab.com/infor-cloud/martian-cloud/tharsis/graphql-query-complexity" // Placeholder to ensure private packages are being downloaded
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	tfecontrollers "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers/tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	rnr "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	maint "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providermirror"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/resourcelimit"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/role"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	tfpAPIEndpointHeader  = "TFP-API-Version"
	tfpAPIEndpointVersion = "2.5.0"

	tfeBasePath    = "/tfe"
	tfeVersionPath = "/v2"
)

// APIServer represents an instance of a server
type APIServer struct {
	logger        logger.Logger
	taskManager   asynctask.Manager
	dbClient      *db.Client
	srv           *http.Server
	traceShutdown func(context.Context) error
	shutdownOnce  sync.Once
}

// New creates a new APIServer instance
func New(ctx context.Context, cfg *config.Config, logger logger.Logger, version string) (*APIServer, error) {
	openIDConfigFetcher := auth.NewOpenIDConfigFetcher()

	var oauthProviders []auth.IdentityProviderConfig
	for _, idpConfig := range cfg.OauthProviders {
		idp, err := openIDConfigFetcher.GetOpenIDConfig(ctx, idpConfig.IssuerURL)
		if err != nil {
			return nil, fmt.Errorf("failed to get OIDC config for issuer %s %v", idpConfig.IssuerURL, err)
		}

		oauthProviders = append(oauthProviders, auth.IdentityProviderConfig{
			Issuer:        idp.Issuer,
			TokenEndpoint: idp.TokenEndpoint,
			AuthEndpoint:  idp.AuthEndpoint,
			JwksURI:       idp.JwksURI,
			ClientID:      idpConfig.ClientID,
			UsernameClaim: idpConfig.UsernameClaim,
		})
	}

	// Initialize a trace provider.
	traceProviderShutdown, err := tracing.NewProvider(ctx,
		&tracing.NewProviderInput{
			Enabled: cfg.OtelTraceEnabled,
			Type:    cfg.OtelTraceType,
			Host:    cfg.OtelTraceCollectorHost,
			Port:    cfg.OtelTraceCollectorPort,
			Version: version,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize trace provider: %w", err)
	}
	if !cfg.OtelTraceEnabled {
		logger.Info("Tracing is disabled.")
	}

	dbClient, err := db.NewClient(
		ctx,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
		cfg.DBSSLMode,
		cfg.DBUsername,
		cfg.DBPassword,
		cfg.DBMaxConnections,
		cfg.DBAutoMigrateEnabled,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB client %v", err)
	}

	pluginCatalog, err := plugin.NewCatalog(ctx, logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin catalog %v", err)
	}

	httpClient := tharsishttp.NewHTTPClient()

	eventManager := events.NewEventManager(dbClient, logger)
	eventManager.Start(ctx)

	maintenanceMonitor := maintenance.NewMonitor(logger, dbClient, eventManager)
	maintenanceMonitor.Start(ctx)

	tharsisIDP := auth.NewIdentityProvider(pluginCatalog.JWSProvider, cfg.ServiceAccountIssuerURL)
	userAuth := auth.NewUserAuth(ctx, oauthProviders, logger, dbClient, maintenanceMonitor)
	authenticator := auth.NewAuthenticator(userAuth, tharsisIDP, dbClient, maintenanceMonitor, cfg.ServiceAccountIssuerURL)

	respWriter := response.NewWriter(logger)

	taskManager := asynctask.NewManager(time.Duration(cfg.AsyncTaskTimeout) * time.Second)

	artifactStore := workspace.NewArtifactStore(pluginCatalog.ObjectStore)
	providerRegistryStore := providerregistry.NewRegistryStore(pluginCatalog.ObjectStore)
	moduleRegistryStore := moduleregistry.NewRegistryStore(pluginCatalog.ObjectStore)
	cliStore := cli.NewCLIStore(pluginCatalog.ObjectStore)
	mirrorStore := providermirror.NewProviderMirrorStore(pluginCatalog.ObjectStore)

	logStreamStore := logstream.NewLogStore(pluginCatalog.ObjectStore, dbClient)
	logStreamManager := logstream.New(logStreamStore, dbClient, eventManager, logger)

	managedIdentityDelegates, err := managedidentity.NewManagedIdentityDelegateMap(ctx, cfg, pluginCatalog)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize managed identity delegate map %v", err)
	}

	runStateManager := state.NewRunStateManager(dbClient, logger)

	limits := limits.NewLimitChecker(dbClient)

	// Services.
	var (
		activityService            = activityevent.NewService(dbClient, logger)
		userService                = user.NewService(logger, dbClient)
		namespaceMembershipService = namespacemembership.NewService(logger, dbClient, activityService)
		groupService               = group.NewService(logger, dbClient, limits, namespaceMembershipService, activityService)
		cliService                 = cli.NewService(logger, httpClient, taskManager, cliStore, cfg.TerraformCLIVersionConstraint)
		workspaceService           = workspace.NewService(logger, dbClient, limits, artifactStore, eventManager, cliService, activityService)
		jobService                 = job.NewService(logger, dbClient, tharsisIDP, logStreamManager, eventManager, runStateManager)
		managedIdentityService     = managedidentity.NewService(logger, dbClient, limits, managedIdentityDelegates, workspaceService, jobService, activityService)
		saService                  = serviceaccount.NewService(logger, dbClient, limits, tharsisIDP, openIDConfigFetcher, activityService)
		variableService            = variable.NewService(logger, dbClient, limits, activityService)
		teamService                = team.NewService(logger, dbClient, activityService)
		providerRegistryService    = providerregistry.NewService(logger, dbClient, limits, providerRegistryStore, activityService)
		moduleRegistryService      = moduleregistry.NewService(logger, dbClient, limits, moduleRegistryStore, activityService, taskManager)
		gpgKeyService              = gpgkey.NewService(logger, dbClient, limits, activityService)
		scimService                = scim.NewService(logger, dbClient, tharsisIDP)
		runService                 = run.NewService(logger, dbClient, artifactStore, eventManager, jobService, cliService, activityService, moduleRegistryService, run.NewModuleResolver(moduleRegistryService, httpClient, logger, cfg.TharsisAPIURL), runStateManager)
		runnerService              = runner.NewService(logger, dbClient, limits, activityService, logStreamManager, eventManager)
		roleService                = role.NewService(logger, dbClient, activityService)
		resourceLimitService       = resourcelimit.NewService(logger, dbClient)
		providerMirrorService      = providermirror.NewService(logger, dbClient, httpClient, limits, activityService, mirrorStore)
		maintenanceModeService     = maint.NewService(logger, dbClient)
	)

	vcsService, err := vcs.NewService(
		ctx,
		logger,
		dbClient,
		limits,
		tharsisIDP,
		httpClient,
		activityService,
		runService,
		workspaceService,
		taskManager,
		cfg.TharsisAPIURL,
		cfg.VCSRepositorySizeLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize vcs service %v", err)
	}

	routeBuilder := api.NewRouteBuilder(
		middleware.PrometheusMiddleware,
		middleware.NewAuthenticationMiddleware(authenticator, logger, respWriter),
		middleware.HTTPRateLimiterMiddleware(
			logger,
			respWriter,
			pluginCatalog.HTTPRateLimitStore,
		), // catch all calls, including GraphQL
	).WithSubRouter("/v1").
		WithSubRouter(tfeBasePath,
			middleware.NewCommonHeadersMiddleware(map[string]string{
				tfpAPIEndpointHeader: tfpAPIEndpointVersion,
			}),
		)

	v1RouteBuilder := routeBuilder.SubRouteBuilder("/v1")

	// set up terraform /v2 routes
	routeBuilder.SubRouteBuilder(tfeBasePath).WithSubRouter(tfeVersionPath)
	terraformV2RouteBuilder := routeBuilder.SubRouteBuilder(tfeBasePath).SubRouteBuilder(tfeVersionPath)

	// Create the admin user if an email is provided.
	if cfg.AdminUserEmail != "" {
		user, uErr := dbClient.Users.GetUserByEmail(ctx, cfg.AdminUserEmail)
		if uErr != nil {
			return nil, uErr
		}
		if user == nil {
			if _, err = dbClient.Users.CreateUser(ctx, &models.User{
				Username: auth.ParseUsername(cfg.AdminUserEmail),
				Email:    cfg.AdminUserEmail,
				Admin:    true,
				Active:   true,
			}); err != nil {
				return nil, fmt.Errorf("failed to create admin user: %v", err)
			}
			logger.Infof("User with email %s created.", cfg.AdminUserEmail)
		} else {
			logger.Infof("User with email %s already exists. Skipping creation.", cfg.AdminUserEmail)
		}
	}

	if cfg.TFELoginEnabled {
		var loginIdp *auth.IdentityProviderConfig
		// Find IDP that matches client ID
		for _, idp := range oauthProviders {
			if idp.ClientID == cfg.TFELoginClientID {
				idp := idp
				loginIdp = &idp
				break
			}
		}

		if loginIdp == nil {
			return nil, errors.New("OIDC Identity Provider not found for TFE login")
		}

		tfeHandler, sdErr := tfe.BuildTFEServiceDiscoveryHandler(logger, loginIdp, cfg.TFELoginScopes, cfg.TharsisAPIURL, tfeBasePath)
		if sdErr != nil {
			return nil, fmt.Errorf("failed to build TFE discovery document handler %v", sdErr)
		}

		routeBuilder.AddHandlerFunc(
			"GET",
			"/.well-known/terraform.json",
			tfeHandler,
		)
	}

	requireAuthenticatedCallerMiddleware := middleware.NewRequireAuthenticatedCallerMiddleware(logger, respWriter)

	resolverState := resolver.State{
		Config:                     cfg,
		GroupService:               groupService,
		WorkspaceService:           workspaceService,
		RunService:                 runService,
		JobService:                 jobService,
		ManagedIdentityService:     managedIdentityService,
		ServiceAccountService:      saService,
		UserService:                userService,
		NamespaceMembershipService: namespaceMembershipService,
		VariableService:            variableService,
		Logger:                     logger,
		TeamService:                teamService,
		ProviderRegistryService:    providerRegistryService,
		ModuleRegistryService:      moduleRegistryService,
		GPGKeyService:              gpgKeyService,
		CliService:                 cliService,
		SCIMService:                scimService,
		VCSService:                 vcsService,
		ActivityService:            activityService,
		RoleService:                roleService,
		RunnerService:              runnerService,
		ResourceLimitService:       resourceLimitService,
		ProviderMirrorService:      providerMirrorService,
		MaintenanceModeService:     maintenanceModeService,
	}

	graphqlHandler, err := graphql.NewGraphQL(&resolverState, logger, pluginCatalog.GraphqlRateLimitStore, cfg.MaxGraphQLComplexity, authenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize graphql handler %v", err)
	}

	routeBuilder.AddHandler("/graphql", graphqlHandler)
	routeBuilder.AddHandlerFunc("GET", "/swagger/*", httpSwagger.WrapHandler)

	// Terraform Backend Ping Endpoint
	terraformV2RouteBuilder.AddHandlerFunc("GET", "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Controllers.
	routeBuilder.AddRoutes(controllers.NewHealthController(
		respWriter,
	))
	routeBuilder.AddRoutes(controllers.NewOIDCController(
		respWriter,
		pluginCatalog.JWSProvider,
		cfg.TharsisAPIURL,
	))

	// TFE Controllers
	terraformV2RouteBuilder.AddRoutes(tfecontrollers.NewStateController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		workspaceService,
		cfg.TharsisAPIURL,
		tfeBasePath+tfeVersionPath,
	))
	terraformV2RouteBuilder.AddRoutes(tfecontrollers.NewRunController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		pluginCatalog.JWSProvider,
		runService,
		cfg.TharsisAPIURL,
	))
	terraformV2RouteBuilder.AddRoutes(tfecontrollers.NewOrgController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		runService,
		groupService,
	))
	terraformV2RouteBuilder.AddRoutes(tfecontrollers.NewWorkspaceController(
		logger,
		respWriter,
		runService,
		workspaceService,
		groupService,
		managedIdentityService,
		pluginCatalog.JWSProvider,
		variableService,
		cfg.TharsisAPIURL,
		tfeBasePath+tfeVersionPath,
	))

	// Tharsis Controllers
	v1RouteBuilder.AddRoutes(controllers.NewRunController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		runService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewJobController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		pluginCatalog.JWSProvider,
		jobService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewServiceAccountController(
		logger,
		respWriter,
		saService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewProviderRegistryController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		providerRegistryService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewModuleRegistryController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		moduleRegistryService,
		cfg.ModuleRegistryMaxUploadSize,
	))
	v1RouteBuilder.AddRoutes(controllers.NewSCIMController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		userService,
		teamService,
		scimService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewVCSController(
		logger,
		respWriter,
		authenticator,
		vcsService,
	))
	v1RouteBuilder.AddRoutes(controllers.NewProviderMirrorController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		providerMirrorService,
	))

	runnerClient := rnr.NewInternalClient(runnerService, jobService)

	for _, r := range cfg.InternalRunners {
		// Create DB entry for runner
		_, err := dbClient.Runners.CreateRunner(ctx, &models.Runner{
			Type:      models.SharedRunnerType,
			Name:      r.Name,
			CreatedBy: "system",
		})
		if err != nil {
			if errors.ErrorCode(err) != errors.EConflict {
				return nil, err
			}
		}

		logger.Infof("starting internal runner %s", r.Name)

		runner, err := rnr.NewRunner(ctx, r.Name, logger, runnerClient, &rnr.JobDispatcherSettings{
			DispatcherType:       r.JobDispatcherType,
			ServiceDiscoveryHost: cfg.ServiceDiscoveryHost,
			PluginData:           r.JobDispatcherData,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create runner %v", err)
		}

		go runner.Start(auth.WithCaller(ctx, &auth.SystemCaller{}))
	}

	return &APIServer{
		logger:      logger,
		dbClient:    dbClient,
		taskManager: taskManager,
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%v", cfg.ServerPort),
			Handler:           routeBuilder.Router(),
			ReadHeaderTimeout: time.Minute,
		},
		traceShutdown: traceProviderShutdown,
	}, nil
}

// Start will start the server
func (api *APIServer) Start() {
	go func() {
		// Serve Prometheus endpoint on its own port since it
		// won't be publicly exposed
		promServer := &http.Server{
			Addr:              ":9090",
			Handler:           promhttp.Handler(),
			ReadHeaderTimeout: 3 * time.Second,
		}

		if err := promServer.ListenAndServe(); err != nil {
			api.logger.Error(err)
		}
	}()

	// Start main server
	if err := api.srv.ListenAndServe(); err != nil {
		api.logger.Infof("HTTP server ListenAndServe %v", err)
	}
}

// Shutdown will shutdown the API server
func (api *APIServer) Shutdown(ctx context.Context) {
	api.shutdownOnce.Do(func() {
		api.logger.Info("Starting HTTP server shutdown")

		// Shutdown HTTP server
		if err := api.srv.Shutdown(ctx); err != nil {
			api.logger.Errorf("failed to shutdown HTTP server gracefully: %v", err)
		}

		api.logger.Info("HTTP server shutdown successfully")

		// Shutdown trace provider.
		if err := api.traceShutdown(ctx); err != nil {
			api.logger.Errorf("Shutdown trace provider failed: %w", err)
		} else {
			api.logger.Info("Shutdown trace provider successfully.")
		}

		api.logger.Info("Starting Async Task Manager shutdown")
		api.taskManager.Shutdown()
		api.logger.Info("Async Task Manager shutdown successfully")

		api.logger.Info("Completed graceful shutdown")
	})
}
