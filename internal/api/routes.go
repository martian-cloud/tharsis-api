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
	subRouters map[string]*RouteBuilder
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

	return &RouteBuilder{baseRouter: router, subRouters: make(map[string]*RouteBuilder)}
}

// Router returns and instance of a chi.Router with all routes added
func (rb *RouteBuilder) Router() chi.Router {
	return rb.baseRouter
}

// SubRouteBuilder returns a RouteBuilder under the given subpath
func (rb *RouteBuilder) SubRouteBuilder(subPath string) *RouteBuilder {
	if subPath == "" {
		return rb
	}
	return rb.subRouters[subPath]
}

// AddHandler adds the handler to the path
func (rb *RouteBuilder) AddHandler(pattern string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	rb.baseRouter.With(middlewares...).Handle(pattern, handler)
	return rb
}

// AddHandlerFunc adds a handler function to the router for a specific pattern
func (rb *RouteBuilder) AddHandlerFunc(method string, pattern string, handler http.HandlerFunc) *RouteBuilder {
	rb.baseRouter.Method(method, pattern, handler)
	return rb
}

// AddRoutes adds the controllers routes to the path
func (rb *RouteBuilder) AddRoutes(controller controllers.Controller) *RouteBuilder {
	rb.baseRouter.Group(func(groupRouter chi.Router) {
		controller.RegisterRoutes(groupRouter)
	})

	return rb
}

// WithSubRouter mounts a router under the current RouteBuilder with the given subpath
func (rb *RouteBuilder) WithSubRouter(subPath string, middlewares ...func(http.Handler) http.Handler) *RouteBuilder {
	subRouteBuilder := NewRouteBuilder(middlewares...)

	rb.baseRouter.Mount(subPath, subRouteBuilder.Router())
	rb.subRouters[subPath] = subRouteBuilder

	return rb
}
