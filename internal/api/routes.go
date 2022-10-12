package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/controllers"
)

// RouteBuilder is used to build a Router instance
type RouteBuilder struct {
	baseRouter chi.Router
	v1Router   chi.Router
}

// NewRouteBuilder creates an instance of RouterBuilder
func NewRouteBuilder(middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	router := chi.NewRouter()

	router.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
	}))

	router.Use(middlewares...)

	v1Router := chi.NewRouter()
	router.Mount("/v1", v1Router)

	return &RouteBuilder{baseRouter: router, v1Router: v1Router}
}

// Build returns and instance of a chi.Router with all routes added
func (rb *RouteBuilder) Build() chi.Router {
	return rb.baseRouter
}

// AddBaseHandler adds the handler to the base path
func (rb *RouteBuilder) AddBaseHandler(pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.baseRouter.With(middlewares...).Handle(pattern, handler)
	return rb
}

// AddBaseHandlerFunc adds a handler function to the base router for a specific pattern
func (rb *RouteBuilder) AddBaseHandlerFunc(method string, pattern string, handler http.HandlerFunc) *RouteBuilder {
	rb.baseRouter.Method(method, pattern, handler)
	return rb
}

// AddBaseRoutes adds the controllers routes to the base path
func (rb *RouteBuilder) AddBaseRoutes(controller controllers.Controller) *RouteBuilder {
	controller.RegisterRoutes(rb.baseRouter)
	return rb
}

// AddV1Routes adds the controllers routes to the /v1 path
func (rb *RouteBuilder) AddV1Routes(controller controllers.Controller) *RouteBuilder {
	// Use Group to create fresh middleware stack
	rb.v1Router.Group(func(groupRouter chi.Router) {
		controller.RegisterRoutes(groupRouter)
	})
	return rb
}
