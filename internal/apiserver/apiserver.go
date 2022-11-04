// Package apiserver is used to initialize the api
package apiserver

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/events"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/gpgkey"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/providerregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/scim"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/variable"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tfe"
)

// APIServer represents an instance of a server
type APIServer struct {
	router   chi.Router
	logger   logger.Logger
	cfg      *config.Config
	dbClient *db.Client
}

// New creates a new APIServer instance
func New(ctx context.Context, cfg *config.Config, logger logger.Logger) (*APIServer, error) {
	var oauthProviders []auth.IdentityProviderConfig
	for _, idpConfig := range cfg.OauthProviders {
		idp, err := auth.GetOpenIDConfig(ctx, idpConfig.IssuerURL)
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
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB client %v", err)
	}

	pluginCatalog, err := plugin.NewCatalog(ctx, logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin catalog %v", err)
	}

	// Used by CLI service.
	httpClient := tharsishttp.NewHTTPClient()

	tharsisIDP := auth.NewIdentityProvider(pluginCatalog.JWSProvider, cfg.ServiceAccountIssuerURL)
	userAuth := auth.NewUserAuth(ctx, oauthProviders, logger, dbClient)
	authenticator := auth.NewAuthenticator(userAuth, tharsisIDP, dbClient, cfg.ServiceAccountIssuerURL)

	respWriter := response.NewWriter(logger)

	eventManager := events.NewEventManager(dbClient)
	eventManager.Start(ctx)

	logStore := job.NewLogStore(pluginCatalog.ObjectStore, dbClient)
	artifactStore := workspace.NewArtifactStore(pluginCatalog.ObjectStore)
	providerRegistryStore := providerregistry.NewRegistryStore(pluginCatalog.ObjectStore)
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
		cliService                 = cli.NewService(logger, httpClient, cliStore)
		workspaceService           = workspace.NewService(logger, dbClient, artifactStore, eventManager, cliService,
			activityService)
		jobService = job.NewService(logger, dbClient, eventManager, logStore)
		runService = run.NewService(logger, dbClient, artifactStore, eventManager, tharsisIDP, jobService,
			cliService, activityService)
		managedIdentityService = managedidentity.NewService(logger, dbClient, managedIdentityDelegates, workspaceService,
			jobService, activityService)
		saService               = serviceaccount.NewService(logger, dbClient, tharsisIDP, activityService)
		variableService         = variable.NewService(logger, dbClient, activityService)
		teamService             = team.NewService(logger, dbClient, activityService)
		providerRegistryService = providerregistry.NewService(logger, dbClient, providerRegistryStore, activityService)
		gpgKeyService           = gpgkey.NewService(logger, dbClient, activityService)
		scimService             = scim.NewService(logger, dbClient, tharsisIDP)
	)

	routeBuilder := api.NewRouteBuilder(
		middleware.PrometheusMiddleware,
	)

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
		GPGKeyService:              gpgKeyService,
		CliService:                 cliService,
		SCIMService:                scimService,
		ActivityService:            activityService,
	}
	graphqlHandler, err := graphql.NewGraphQL(&resolverState, logger, pluginCatalog.RateLimitStore, cfg.MaxGraphQLComplexity, authenticator, jwtAuthMiddleware)
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
	routeBuilder.AddV1Routes(controllers.NewSCIMController(
		logger,
		respWriter,
		jwtAuthMiddleware,
		userService,
		teamService,
		scimService,
	))

	runner := runner.NewRunner(runService, pluginCatalog.JobDispatcher, logger)
	runner.Start(auth.WithCaller(ctx, &auth.SystemCaller{}))

	return &APIServer{
		logger:   logger,
		router:   routeBuilder.Build(),
		cfg:      cfg,
		dbClient: dbClient,
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
	if err := http.ListenAndServe(fmt.Sprintf(":%v", api.cfg.ServerPort), api.router); err != nil {
		api.logger.Error(err)
		os.Exit(-1)
	}
}

// Shutdown will shutdown the API server
func (api *APIServer) Shutdown(ctx context.Context) {
	api.logger.Info("Starting API shutdown")
	api.dbClient.Close(ctx)
	api.logger.Info("Completed API shutdown")
}
