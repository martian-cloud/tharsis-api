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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/asynctask"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tfe"
)

// APIServer represents an instance of a server
type APIServer struct {
	shutdownOnce sync.Once
	logger       logger.Logger
	dbClient     *db.Client
	taskManager  asynctask.Manager
	srv          *http.Server
}

// New creates a new APIServer instance
func New(ctx context.Context, cfg *config.Config, logger logger.Logger) (*APIServer, error) {
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

	tharsisIDP := auth.NewIdentityProvider(pluginCatalog.JWSProvider, cfg.ServiceAccountIssuerURL)
	userAuth := auth.NewUserAuth(ctx, oauthProviders, logger, dbClient)
	authenticator := auth.NewAuthenticator(userAuth, tharsisIDP, dbClient, cfg.ServiceAccountIssuerURL)

	respWriter := response.NewWriter(logger)

	taskManager := asynctask.NewManager(time.Duration(cfg.AsyncTaskTimeout) * time.Second)

	eventManager := events.NewEventManager(dbClient)
	eventManager.Start(ctx)

	logStore := job.NewLogStore(pluginCatalog.ObjectStore, dbClient)
	artifactStore := workspace.NewArtifactStore(pluginCatalog.ObjectStore)
	providerRegistryStore := providerregistry.NewRegistryStore(pluginCatalog.ObjectStore)
	moduleRegistryStore := moduleregistry.NewRegistryStore(pluginCatalog.ObjectStore)
	cliStore := cli.NewCLIStore(pluginCatalog.ObjectStore)

	managedIdentityDelegates, err := managedidentity.NewManagedIdentityDelegateMap(ctx, cfg, pluginCatalog)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize managed identity delegate map %v", err)
	}

	// Services.
	var (
		activityService            = activityevent.NewService(dbClient, logger)
		userService                = user.NewService(logger, dbClient)
		namespaceMembershipService = namespacemembership.NewService(logger, dbClient, activityService)
		groupService               = group.NewService(logger, dbClient, namespaceMembershipService, activityService)
		cliService                 = cli.NewService(logger, httpClient, taskManager, cliStore)
		workspaceService           = workspace.NewService(logger, dbClient, artifactStore, eventManager, cliService, activityService)
		jobService                 = job.NewService(logger, dbClient, eventManager, logStore)
		managedIdentityService     = managedidentity.NewService(logger, dbClient, managedIdentityDelegates, workspaceService, jobService, activityService)
		saService                  = serviceaccount.NewService(logger, dbClient, tharsisIDP, openIDConfigFetcher, activityService)
		variableService            = variable.NewService(logger, dbClient, activityService)
		teamService                = team.NewService(logger, dbClient, activityService)
		providerRegistryService    = providerregistry.NewService(logger, dbClient, providerRegistryStore, activityService)
		moduleRegistryService      = moduleregistry.NewService(logger, dbClient, moduleRegistryStore, activityService, taskManager)
		gpgKeyService              = gpgkey.NewService(logger, dbClient, activityService)
		scimService                = scim.NewService(logger, dbClient, tharsisIDP)
		runService                 = run.NewService(logger, dbClient, artifactStore, eventManager, tharsisIDP, jobService, cliService, activityService, moduleRegistryService, run.NewModuleResolver(moduleRegistryService, httpClient, logger, cfg.TharsisAPIURL))
	)

	vcsService, err := vcs.NewService(
		ctx,
		logger,
		dbClient,
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
	)

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
			return nil, errors.NewError(
				errors.EInternal,
				"OIDC Identity Provider not found for TFE login",
				errors.WithErrorErr(err),
			)
		}

		tfeHandler, sdErr := tfe.BuildTFEServiceDiscoveryHandler(logger, loginIdp, cfg.TFELoginScopes, cfg.TharsisAPIURL)
		if sdErr != nil {
			return nil, fmt.Errorf("failed to build TFE discovery document handler %v", sdErr)
		}

		routeBuilder.AddBaseHandlerFunc(
			"GET",
			"/.well-known/terraform.json",
			tfeHandler,
		)
	}

	jwtAuthMiddleware := middleware.NewJwtAuthMiddleware(authenticator, logger, respWriter)

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
	}

	graphqlHandler, err := graphql.NewGraphQL(&resolverState, logger, pluginCatalog.RateLimitStore, cfg.MaxGraphQLComplexity, authenticator)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize graphql handler %v", err)
	}

	routeBuilder.AddBaseHandler("/graphql", graphqlHandler)
	routeBuilder.AddBaseHandlerFunc("GET", "/swagger/*", httpSwagger.WrapHandler)

	// Controllers.
	routeBuilder.AddBaseRoutes(controllers.NewHealthController(
		respWriter,
	))
	routeBuilder.AddBaseRoutes(controllers.NewOIDCController(
		respWriter,
		pluginCatalog.JWSProvider,
		cfg.TharsisAPIURL,
	))

	routeBuilder.AddV1Routes(controllers.NewRunController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		pluginCatalog.JWSProvider,
		runService,
		cfg.TharsisAPIURL,
	))
	routeBuilder.AddV1Routes(controllers.NewJobController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		pluginCatalog.JWSProvider,
		jobService,
	))
	routeBuilder.AddV1Routes(controllers.NewOrgController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		runService,
		groupService,
	))
	routeBuilder.AddV1Routes(controllers.NewWorkspaceController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		runService,
		workspaceService,
		groupService,
		managedIdentityService,
		variableService,
		cfg.TharsisAPIURL,
	))
	routeBuilder.AddV1Routes(controllers.NewServiceAccountController(
		logger,
		respWriter,
		saService,
	))
	routeBuilder.AddV1Routes(controllers.NewProviderRegistryController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		providerRegistryService,
	))
	routeBuilder.AddV1Routes(controllers.NewModuleRegistryController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		moduleRegistryService,
		cfg.ModuleRegistryMaxUploadSize,
	))
	routeBuilder.AddV1Routes(controllers.NewSCIMController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		userService,
		teamService,
		scimService,
	))
	routeBuilder.AddV1Routes(controllers.NewVCSController(
		logger,
		respWriter,
		authenticator,
		vcsService,
	))

	runner := runner.NewRunner(runService, pluginCatalog.JobDispatcher, logger)
	runner.Start(auth.WithCaller(ctx, &auth.SystemCaller{}))

	return &APIServer{
		logger:      logger,
		dbClient:    dbClient,
		taskManager: taskManager,
		srv: &http.Server{
			Addr:              fmt.Sprintf(":%v", cfg.ServerPort),
			Handler:           routeBuilder.Build(),
			ReadHeaderTimeout: time.Minute,
		},
	}, nil
}

// Start will start the server
func (api *APIServer) Start() {
	go func() {
		// Serve Prometheus endpoint on its own port since it
		// won't be publicly exposed
		if err := http.ListenAndServe(":9090", promhttp.Handler()); err != nil {
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

		api.logger.Info("Starting Async Task Manager shutdown")
		api.taskManager.Shutdown()
		api.logger.Info("Async Task Manager shutdown successfully")

		api.logger.Info("Completed graceful shutdown")
	})
}
