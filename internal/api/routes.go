// Package api package
package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
	tfecontrollers "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers/tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tfe"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/ui"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	tfpAPIEndpointHeader  = "TFP-API-Version"
	tfpAPIEndpointVersion = "2.5.0"

	tfeBasePath    = "/tfe"
	tfeVersionPath = "/v2"
)

// BuildRouter builds the http router for the API server
func BuildRouter(
	ctx context.Context,
	cfg *config.Config,
	logger logger.Logger,
	respWriter response.Writer,
	pluginCatalog *plugin.Catalog,
	authenticator auth.Authenticator,
	openIDConfigFetcher auth.OpenIDConfigFetcher,
	serviceCatalog *services.Catalog,
	userSessionManager auth.UserSessionManager,
	signingKeyManager auth.SigningKeyManager,
) (chi.Router, error) {
	resolverState := resolver.State{
		Config:         cfg,
		Logger:         logger,
		ServiceCatalog: serviceCatalog,
	}

	// The connection timeout will use the same timeout as the session refresh token to ensure that a graphql subscription
	// cannot persist longer than the refresh token duration
	maxSubscriptionDuration := time.Duration(cfg.UserSessionRefreshTokenExpirationMinutes) * time.Minute
	graphqlHandler, err := graphql.NewGraphQL(&resolverState, logger, pluginCatalog.GraphqlRateLimitStore, authenticator, cfg.MaxGraphQLComplexity, maxSubscriptionDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize graphql handler %v", err)
	}

	tharsisUIURL, err := url.Parse(cfg.TharsisUIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tharsis ui url: %v", err)
	}

	var tfeHandler http.HandlerFunc
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

		tfeHandler, err = tfe.BuildTFEServiceDiscoveryHandler(ctx, logger, loginIdp, cfg.TFELoginScopes, cfg.TharsisAPIURL, tfeBasePath, openIDConfigFetcher)
		if err != nil {
			return nil, fmt.Errorf("failed to build TFE discovery document handler %v", err)
		}
	}

	// Check if external url uses https to determine if secure cookies should be enabled
	enableSecureCookies := false
	if parsedURL, err := url.Parse(cfg.TharsisAPIURL); err == nil && parsedURL.Scheme == "https" {
		enableSecureCookies = true
	}

	allowedOrigins := strings.Split(cfg.CorsAllowedOrigins, ",")
	for i, part := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(part)
	}

	commonMiddleware := []func(http.Handler) http.Handler{
		cors.Handler(cors.Options{
			AllowedOrigins:   allowedOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", auth.CSRFTokenHeader},
			AllowCredentials: true,
		}),
		middleware.NewRequestIDMiddleware(),
		middleware.PrometheusMiddleware,
		middleware.NewAuthenticationMiddleware(authenticator, respWriter, enableSecureCookies),
		middleware.NewSubjectMiddleware(logger, respWriter),
		middleware.HTTPRateLimiterMiddleware(
			logger,
			respWriter,
			pluginCatalog.HTTPRateLimitStore,
		),
	}

	csrfMiddleware := middleware.NewCSRFMiddleware(respWriter, userSessionManager, logger)

	requireAuthenticatedCallerMiddleware := middleware.NewRequireAuthenticatedCallerMiddleware(logger, respWriter)

	/* Root router */
	router := chi.NewRouter()
	router.Use(commonMiddleware...)

	router.Method("GET", "/swagger/*", httpSwagger.WrapHandler)

	// UI routes - serve at base path with SPA fallback
	uiHandler, err := ui.NewHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize UI handler %v", err)
	}

	if uiHandler != nil {
		logger.Info("Serving UI at base path")
		router.Handle("/*", uiHandler)
	}

	AddRoutes(router, controllers.NewHealthController(
		respWriter,
	))

	AddRoutes(router, controllers.NewOIDCController(
		respWriter,
		signingKeyManager,
	))

	if tfeHandler != nil {
		router.Method("GET", "/.well-known/terraform.json", tfeHandler)
	}

	router.Group(func(r chi.Router) {
		r.Use(csrfMiddleware)
		r.Handle("/graphql", graphqlHandler)
	})

	/* TFE API Router */

	tfeVersionRouter := chi.NewRouter()
	tfeVersionRouter.Use(middleware.NewCommonHeadersMiddleware(map[string]string{
		tfpAPIEndpointHeader: tfpAPIEndpointVersion,
	}))
	tfeVersionRouter.Use(csrfMiddleware)
	router.Mount(fmt.Sprintf("%s%s", tfeBasePath, tfeVersionPath), tfeVersionRouter)

	// Terraform Backend Ping Endpoint
	tfeVersionRouter.MethodFunc("GET", "/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	AddRoutes(tfeVersionRouter, tfecontrollers.NewStateController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		serviceCatalog.WorkspaceService,
		cfg.TharsisAPIURL,
		tfeBasePath+tfeVersionPath,
	))
	AddRoutes(tfeVersionRouter, tfecontrollers.NewRunController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		signingKeyManager,
		serviceCatalog.RunService,
		cfg.TharsisAPIURL,
	))
	AddRoutes(tfeVersionRouter, tfecontrollers.NewOrgController(
		logger,
		respWriter,
		requireAuthenticatedCallerMiddleware,
		serviceCatalog.RunService,
		serviceCatalog.GroupService,
	))
	AddRoutes(tfeVersionRouter, tfecontrollers.NewWorkspaceController(
		logger,
		respWriter,
		serviceCatalog.RunService,
		serviceCatalog.WorkspaceService,
		serviceCatalog.GroupService,
		serviceCatalog.ManagedIdentityService,
		signingKeyManager,
		serviceCatalog.VariableService,
		cfg.TharsisAPIURL,
		tfeBasePath+tfeVersionPath,
	))

	/* V1 Router */

	v1Router := chi.NewRouter()
	router.Mount("/v1", v1Router)

	v1Router.Group(func(r chi.Router) {
		AddRoutes(r, controllers.NewUserSessionController(
			respWriter,
			userSessionManager,
			cfg.UserSessionAccessTokenExpirationMinutes,
			csrfMiddleware,
			enableSecureCookies,
			tharsisUIURL.Hostname(),
			logger,
		))

		AddRoutes(r, controllers.NewVCSController(
			logger,
			respWriter,
			authenticator,
			serviceCatalog.VCSService,
		))
	})

	v1Router.Group(func(r chi.Router) {
		// Add CSRF middleware after the session controller to avoid invoking it for session endpoints
		r.Use(csrfMiddleware)

		AddRoutes(r, controllers.NewRunController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			serviceCatalog.RunService,
		))
		AddRoutes(r, controllers.NewJobController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			signingKeyManager,
			serviceCatalog.JobService,
		))
		AddRoutes(r, controllers.NewServiceAccountController(
			logger,
			respWriter,
			serviceCatalog.ServiceAccountService,
		))
		AddRoutes(r, controllers.NewProviderRegistryController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			serviceCatalog.TerraformProviderRegistryService,
		))
		AddRoutes(r, controllers.NewModuleRegistryController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			serviceCatalog.TerraformModuleRegistryService,
			cfg.ModuleRegistryMaxUploadSize,
		))
		AddRoutes(r, controllers.NewSCIMController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			serviceCatalog.UserService,
			serviceCatalog.TeamService,
			serviceCatalog.SCIMService,
		))
		AddRoutes(r, controllers.NewProviderMirrorController(
			logger,
			respWriter,
			requireAuthenticatedCallerMiddleware,
			serviceCatalog.TerraformProviderMirrorService,
		))
	})

	return router, nil
}

// AddRoutes adds the controllers routes to the path
func AddRoutes(router chi.Router, controller controllers.Controller) {
	router.Group(func(groupRouter chi.Router) {
		controller.RegisterRoutes(groupRouter)
	})
}
