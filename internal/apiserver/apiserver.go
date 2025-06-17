// Package apiserver is used to initialize the api
package apiserver

import (
	"context"
	"crypto/tls"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/email"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logstream"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/registry"
	rnr "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/federatedregistry"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/state/eventhandlers"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/version"
	workspacesvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/workspace"
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
	tlsConfig     *tls.Config
	shutdownOnce  sync.Once
}

// New creates a new APIServer instance
func New(ctx context.Context, cfg *config.Config, logger logger.Logger, apiVersion string, buildTimestamp string) (*APIServer, error) {
	openIDConfigFetcher := auth.NewOpenIDConfigFetcher()

	tlsConfig, err := loadTLSConfig(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Initialize a trace provider.
	traceProviderShutdown, err := tracing.NewProvider(ctx,
		&tracing.NewProviderInput{
			Enabled: cfg.OtelTraceEnabled,
			Type:    cfg.OtelTraceType,
			Host:    cfg.OtelTraceCollectorHost,
			Port:    cfg.OtelTraceCollectorPort,
			Version: apiVersion,
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

	tharsisIDP := auth.NewIdentityProvider(pluginCatalog.JWSProvider, cfg.JWTIssuerURL)
	userAuth := auth.NewUserAuth(ctx, cfg.OauthProviders, logger, dbClient, maintenanceMonitor, openIDConfigFetcher)
	federatedRegistryAuth := auth.NewFederatedRegistryAuth(ctx, cfg.FederatedRegistryTrustPolicies, logger, openIDConfigFetcher, dbClient)
	authenticator := auth.NewAuthenticator(userAuth, federatedRegistryAuth, tharsisIDP, dbClient, maintenanceMonitor, cfg.JWTIssuerURL)

	respWriter := response.NewWriter(logger)

	taskManager := asynctask.NewManager(time.Duration(cfg.AsyncTaskTimeout) * time.Second)

	artifactStore := workspacesvc.NewArtifactStore(pluginCatalog.ObjectStore)
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

	limits := limits.NewLimitChecker(dbClient)
	inheritedSettingsResolver := namespace.NewInheritedSettingResolver(dbClient)
	notificationManager := namespace.NewNotificationManager(dbClient, inheritedSettingsResolver)
	federatedRegistryClient := registry.NewFederatedRegistryClient(tharsisIDP)

	emailClient := email.NewClient(pluginCatalog.EmailProvider, taskManager, dbClient, logger, cfg.TharsisUIURL, cfg.EmailFooter)
	runStateManager := state.NewRunStateManager(dbClient, logger)
	eventhandlers.NewErroredRunEmailHandler(logger, dbClient, runStateManager, emailClient, notificationManager, taskManager).RegisterHandlers()
	eventhandlers.NewAssessmentRunHandler(logger, dbClient, runStateManager).RegisterHandlers()

	// Services.
	var (
		activityService            = activityevent.NewService(dbClient, logger)
		userService                = user.NewService(logger, dbClient, inheritedSettingsResolver)
		namespaceMembershipService = namespacemembership.NewService(logger, dbClient, activityService)
		groupService               = group.NewService(logger, dbClient, limits, namespaceMembershipService, activityService, inheritedSettingsResolver)
		cliService                 = cli.NewService(logger, httpClient, taskManager, cliStore, cfg.TerraformCLIVersionConstraint)
		workspaceService           = workspacesvc.NewService(logger, dbClient, limits, artifactStore, eventManager, cliService, activityService, inheritedSettingsResolver)
		jobService                 = job.NewService(logger, dbClient, tharsisIDP, logStreamManager, eventManager, runStateManager)
		managedIdentityService     = managedidentity.NewService(logger, dbClient, limits, managedIdentityDelegates, workspaceService, jobService, activityService)
		saService                  = serviceaccount.NewService(logger, dbClient, limits, tharsisIDP, openIDConfigFetcher, activityService)
		variableService            = variable.NewService(logger, dbClient, limits, activityService, pluginCatalog.SecretManager, cfg.DisableSensitiveVariableFeature)
		teamService                = team.NewService(logger, dbClient, activityService)
		providerRegistryService    = providerregistry.NewService(logger, dbClient, limits, providerRegistryStore, activityService)
		moduleRegistryService      = moduleregistry.NewService(logger, dbClient, limits, moduleRegistryStore, activityService, taskManager)
		gpgKeyService              = gpgkey.NewService(logger, dbClient, limits, activityService)
		scimService                = scim.NewService(logger, dbClient, tharsisIDP)
		federatedRegistryService   = federatedregistry.NewService(logger, dbClient, limits, activityService, tharsisIDP)
		moduleResolver             = registry.NewModuleResolver(dbClient, httpClient, federatedRegistryClient, logger, cfg.TharsisAPIURL, tharsisIDP)
		runService                 = run.NewService(logger, dbClient, artifactStore, eventManager, jobService, cliService, activityService, moduleResolver, runStateManager, limits, pluginCatalog.SecretManager)
		runnerService              = runner.NewService(logger, dbClient, limits, activityService, logStreamManager, eventManager)
		roleService                = role.NewService(logger, dbClient, activityService)
		resourceLimitService       = resourcelimit.NewService(logger, dbClient)
		providerMirrorService      = providermirror.NewService(logger, dbClient, httpClient, limits, activityService, mirrorStore)
		maintenanceModeService     = maint.NewService(logger, dbClient)
	)

	versionService, err := version.NewService(dbClient, apiVersion, buildTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize version service %v", err)
	}

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

	serviceCatalog := &services.Catalog{
		ActivityEventService:             activityService,
		CLIService:                       cliService,
		FederatedRegistryService:         federatedRegistryService,
		GPGKeyService:                    gpgKeyService,
		GroupService:                     groupService,
		JobService:                       jobService,
		MaintenanceModeService:           maintenanceModeService,
		ManagedIdentityService:           managedIdentityService,
		NamespaceMembershipService:       namespaceMembershipService,
		ResourceLimitService:             resourceLimitService,
		RoleService:                      roleService,
		RunnerService:                    runnerService,
		RunService:                       runService,
		SCIMService:                      scimService,
		ServiceAccountService:            saService,
		TeamService:                      teamService,
		TerraformModuleRegistryService:   moduleRegistryService,
		TerraformProviderMirrorService:   providerMirrorService,
		TerraformProviderRegistryService: providerRegistryService,
		UserService:                      userService,
		VCSService:                       vcsService,
		VariableService:                  variableService,
		VersionService:                   versionService,
		WorkspaceService:                 workspaceService,
	}
	serviceCatalog.Init()

	// Start workspace assessment scheduler
	workspace.NewAssessmentScheduler(
		dbClient,
		logger,
		runService,
		inheritedSettingsResolver,
		maintenanceMonitor,
		time.Duration(cfg.WorkspaceAssessmentIntervalHours)*time.Hour,
		cfg.WorkspaceAssessmentRunLimit,
	).Start(ctx)

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
		var loginIdp *config.IdpConfig
		// Find IDP that matches client ID
		for _, idp := range cfg.OauthProviders {
			if idp.ClientID == cfg.TFELoginClientID {
				idp := idp
				loginIdp = &idp
				break
			}
		}

		if loginIdp == nil {
			return nil, errors.New("OIDC Identity Provider not found for TFE login")
		}

		tfeHandler, sdErr := tfe.BuildTFEServiceDiscoveryHandler(ctx, logger, loginIdp, cfg.TFELoginScopes, cfg.TharsisAPIURL, tfeBasePath, openIDConfigFetcher)
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
		Config:         cfg,
		Logger:         logger,
		ServiceCatalog: serviceCatalog,
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
		runnerModel, err := dbClient.Runners.CreateRunner(ctx, &models.Runner{
			Type:            models.SharedRunnerType,
			Name:            r.Name,
			CreatedBy:       "system",
			RunUntaggedJobs: true,
		})
		if err != nil {
			if errors.ErrorCode(err) != errors.EConflict {
				return nil, err
			}
		}

		if runnerModel == nil {
			runnerModel, err = dbClient.Runners.GetRunnerByTRN(ctx, types.RunnerModelType.BuildTRN(r.Name))
			if err != nil {
				return nil, fmt.Errorf("failed to get internal runner %q: %v", r.Name, err)
			}
		}

		logger.Infof("starting internal runner %q with job dispatcher type %q", r.Name, r.JobDispatcherType)

		runner, err := rnr.NewRunner(ctx, r.Name, logger, apiVersion, runnerClient, &rnr.JobDispatcherSettings{
			DispatcherType:       r.JobDispatcherType,
			ServiceDiscoveryHost: cfg.ServiceDiscoveryHost,
			PluginData:           r.JobDispatcherData,
			TokenGetterFunc:      rnr.NewInternalTokenProvider(r.Name, runnerModel.Metadata.ID, tharsisIDP).GetToken,
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
			TLSConfig:         tlsConfig,
		},
		traceShutdown: traceProviderShutdown,
		tlsConfig:     tlsConfig,
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
			TLSConfig:         api.tlsConfig,
		}

		api.logger.Infof("Prometheus server listening on %s", promServer.Addr)

		var err error
		if api.tlsConfig != nil {
			err = promServer.ListenAndServeTLS("", "")
		} else {
			err = promServer.ListenAndServe()
		}

		if err != nil {
			api.logger.Error("Prometheus server failed to start: %v", err)
			return
		}
	}()

	var err error
	if api.tlsConfig != nil {
		api.logger.Infof("HTTPS server listening on %s", api.srv.Addr)
		err = api.srv.ListenAndServeTLS("", "")
	} else {
		api.logger.Infof("HTTP server listening on %s", api.srv.Addr)
		err = api.srv.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		api.logger.Errorf("HTTP server failed to start: %v", err)
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
