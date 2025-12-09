package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

// WriteHeader overrides the WriteHeader function in http.ResponseWriter
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var totalRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"path", "caller_type"},
)

var responseStatus = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"status"},
)

// PrometheusMiddleware adds basic metrics to a handler
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(rw, r)

		ctx := r.Context()

		var callerType string

		caller := auth.GetCaller(ctx)
		if caller != nil {
			switch caller.(type) {
			case *auth.UserCaller:
				callerType = "user"
			case *auth.ServiceAccountCaller:
				callerType = "service_account"
			case *auth.JobCaller:
				callerType = "job"
			case *auth.SCIMCaller:
				callerType = "scim"
			case *auth.FederatedRegistryCaller:
				callerType = "federated_registry"
			case *auth.VCSWorkspaceLinkCaller:
				callerType = "vcs_workspace_link"
			default:
				callerType = "unknown"
			}
		} else {
			callerType = "anonymous"
		}

		statusCode := rw.Status()

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "<invalid_path>"
		}

		sanitizedPath := strings.ToValidUTF8(routePattern, "<INVALID_UTF_SEQ>")

		responseStatus.WithLabelValues(strconv.Itoa(statusCode)).Inc()
		totalRequests.WithLabelValues(sanitizedPath, callerType).Inc()
	})
}
