package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/graph-gophers/dataloader"
	"github.com/graph-gophers/graphql-go"
	grapherrors "github.com/graph-gophers/graphql-go/errors"
	"github.com/graph-gophers/graphql-go/trace/otel"

	complexity "gitlab.com/infor-cloud/martian-cloud/tharsis/graphql-query-complexity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/schema"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/ratelimitstore"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// maxQueryNestedLevels defines the maximum number of nested levels allowed in a GraphQL query
	maxQueryNestedLevels = 20
)

// fieldOverrides is initialized map passed into GetQueryComplexity
var fieldOverrides = map[string]int{
	"readme":                   1,
	"runnerAvailabilityStatus": 5,
}

type queryComplexityResult struct {
	Throttled          bool `json:"throttled"`
	RequestedQueryCost int  `json:"requestedQueryCost"`
	MaxQueryCost       int  `json:"maxQueryCost"`
	Remaining          int  `json:"remaining"`
}

type contextGenerator struct {
	resolverState *resolver.State
	loaders       *loader.Collection
	disableCache  bool
}

func (c *contextGenerator) BuildContext(ctx context.Context, _ *http.Request) (context.Context, error) {
	// Build context for subscriptions
	ctx = c.resolverState.Attach(ctx)

	options := []dataloader.Option{}
	if c.disableCache {
		options = append(options, dataloader.WithCache(&dataloader.NoCache{}))
	}
	ctx = c.loaders.Attach(ctx, options...)
	return ctx, nil
}

// The GraphQL handler handles GraphQL API requests over HTTP.
type GraphQL struct {
	Logger      logger.Logger
	handlerFunc http.HandlerFunc
}

// NewGraphQL creates a new GraphQL instance
func NewGraphQL(
	resolverState *resolver.State,
	logger logger.Logger,
	ratelimitStore ratelimitstore.Store,
	maxGraphqlComplexity int,
	authenticator *auth.Authenticator,
) (*GraphQL, error) {
	schemaStr, err := schema.String()
	if err != nil {
		return nil, fmt.Errorf("failed to create graphql schema %v", err)
	}

	loaderCollection := loader.NewCollection()
	resolver.RegisterGroupLoader(loaderCollection)
	resolver.RegisterWorkspaceLoader(loaderCollection)
	resolver.RegisterWorkspaceAssessmentLoader(loaderCollection)
	resolver.RegisterApplyLoader(loaderCollection)
	resolver.RegisterPlanLoader(loaderCollection)
	resolver.RegisterUserLoader(loaderCollection)
	resolver.RegisterServiceAccountLoader(loaderCollection)
	resolver.RegisterConfigurationVersionLoader(loaderCollection)
	resolver.RegisterStateVersionLoader(loaderCollection)
	resolver.RegisterRunLoader(loaderCollection)
	resolver.RegisterJobLoader(loaderCollection)
	resolver.RegisterTeamLoader(loaderCollection)
	resolver.RegisterTerraformProviderLoader(loaderCollection)
	resolver.RegisterTerraformProviderVersionLoader(loaderCollection)
	resolver.RegisterTerraformModuleLoader(loaderCollection)
	resolver.RegisterTerraformModuleVersionLoader(loaderCollection)
	resolver.RegisterGPGKeyLoader(loaderCollection)
	resolver.RegisterManagedIdentityLoader(loaderCollection)
	resolver.RegisterManagedIdentityAccessRuleLoader(loaderCollection)
	resolver.RegisterNamespaceMembershipLoader(loaderCollection)
	resolver.RegisterNamespaceVariableLoader(loaderCollection)
	resolver.RegisterVCSProviderLoader(loaderCollection)
	resolver.RegisterVCSEventLoader(loaderCollection)
	resolver.RegisterRoleLoader(loaderCollection)
	resolver.RegisterRunnerLoader(loaderCollection)
	resolver.RegisterTerraformProviderVersionMirrorLoader(loaderCollection)
	resolver.RegisterRunnerSessionLogStreamLoader(loaderCollection)
	resolver.RegisterJobLogStreamLoader(loaderCollection)
	resolver.RegisterRunStateVersionLoader(loaderCollection)
	resolver.RegisterFederatedRegistryLoader(loaderCollection)

	schema := graphql.MustParseSchema(schemaStr, resolver.NewRootResolver(), graphql.UseFieldResolvers(),
		graphql.Tracer(&otel.Tracer{
			Tracer: tracer,
		}),
		graphql.SubscribeResolverTimeout(time.Second*10),
	)

	httpHandler := httpHandler{
		schema:         schema,
		logger:         logger,
		rateLimitStore: ratelimitStore,
		ctxGenerator: &contextGenerator{
			resolverState: resolverState,
			loaders:       loaderCollection,
		},
		maxGraphqlComplexity: maxGraphqlComplexity,
		authenticator:        authenticator,
	}

	return &GraphQL{
		Logger: logger,
		handlerFunc: newSubscriptionHandler(
			&httpHandler,
			schema,
			authenticator,
			resolverState,
			loaderCollection,
		),
	}, nil
}

func (h *GraphQL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate the request.
	if ok := isSupported(r.Method); !ok {
		respond(w, errorJSON("only POST or GET requests are supported"), http.StatusMethodNotAllowed)
		return
	}

	h.handlerFunc(w, r)
}

type httpHandler struct {
	schema               *graphql.Schema
	logger               logger.Logger
	ctxGenerator         *contextGenerator
	rateLimitStore       ratelimitstore.Store
	authenticator        *auth.Authenticator
	maxGraphqlComplexity int
}

var (
	rateLimitExceededCount   = metric.NewCounter("rate_limit_exceeded_count", "Amount of times rate limit exceeded.")
	queryExecutionTime       = metric.NewHistogram("query_execution_time", "Amount of time a query took to execute.", 1, 4, 6)
	queryComplexityHistogram = metric.NewHistogram("query_complexity", "Query complexity.", 1, 5, 10)
)

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	caller := auth.GetCaller(r.Context())

	parentCtx := r.Context()
	if caller != nil {
		parentCtx = auth.WithCaller(parentCtx, caller)
	}
	ctx, err := h.ctxGenerator.BuildContext(parentCtx, r)
	if err != nil {
		respond(w, errorJSON(err.Error()), http.StatusInternalServerError)
		return
	}

	subject := auth.GetSubject(r.Context())
	if subject == nil {
		respond(w, errorJSON("no subject string in context"), http.StatusInternalServerError)
		return
	}

	req, err := parse(r)
	if err != nil {
		respond(w, errorJSON(err.Error()), http.StatusBadRequest)
		return
	}

	n := len(req.queries)
	if n == 0 {
		respond(w, errorJSON("no queries to execute"), http.StatusBadRequest)
		return
	}

	var (
		responses = make([]*graphql.Response, n) // Allocate a slice large enough for all responses.
		wg        sync.WaitGroup                 // Use the WaitGroup to wait for all executions to finish.
	)

	wg.Add(n)
	start := time.Now()

	for i, q := range req.queries {
		// Loop through the parsed queries from the request.
		// These queries are executed in separate goroutines so they process in parallel.
		go func(i int, q query) {
			defer wg.Done()
			var res *graphql.Response

			// Rate limit query
			queryComplexity, qcErr := h.calculateQueryComplexity(ctx, q, *subject)
			if qcErr != nil {
				h.logger.Errorf("An error occurred while checking graphql query complexity; %v", qcErr)
				err := errors.New(
					"invalid graphql query: "+strings.TrimPrefix(qcErr.Error(), "graphql: syntax error: "),
					errors.WithErrorCode(errors.EInvalid),
				)
				responses[i] = &graphql.Response{Errors: []*grapherrors.QueryError{{Err: err, Message: errors.ErrorMessage(err)}}}
				return
			}

			if !queryComplexity.Throttled {
				res = h.schema.Exec(ctx, q.Query, q.OpName, q.Variables)
				// Expand errors when it is possible for a resolver to return
				// more than one error (for example, a list resolver).
				res.Errors = expandResolverErrors(res.Errors)
			} else {
				rateLimitExceededCount.Inc()
				err := errors.New(
					"max query complexity exceeded",
					errors.WithErrorCode(errors.ETooManyRequests),
				)
				res = &graphql.Response{Errors: []*grapherrors.QueryError{{Err: err, Message: errors.ErrorMessage(err)}}}
			}

			if res.Extensions == nil {
				res.Extensions = map[string]interface{}{}
			}
			// Add query cost extension
			res.Extensions["cost"] = queryComplexity

			responses[i] = res
		}(i, q)
	}

	wg.Wait()
	duration := time.Since(start)
	queryExecutionTime.Observe(float64(duration.Milliseconds()))

	// Add extensions to errors
	for _, response := range responses {
		for _, e := range response.Errors {
			if e != nil && e.Err != nil {
				if !errors.IsContextCanceledError(e.Err) {
					switch errors.ErrorCode(e.Err) {
					case errors.EInternal:
						h.logger.Errorf("Unexpected error occurred: %s", e.Err.Error())
						// Avoid exposing sensitive error messages
						e.Message = errors.InternalErrorMessage
					case errors.EUnauthorized:
						h.logger.Infof("Unauthorized request from subject: %s", *subject)
					}
				}

				e.Extensions = getErrExtensions(e.Err)
			}
		}
	}

	var resp []byte
	if req.isBatch {
		resp, err = json.Marshal(responses)
	} else if len(responses) > 0 {
		resp, err = json.Marshal(responses[0])
	}

	if err != nil {
		respond(w, errorJSON("server error"), http.StatusInternalServerError)
		return
	}
	respond(w, resp, http.StatusOK)
}

func (h *httpHandler) calculateQueryComplexity(ctx context.Context, q query, subject string) (*queryComplexityResult, error) {
	// calculate query complexity
	complexity, err := complexity.GetQueryComplexity(q.Query, q.Variables, fieldOverrides, maxQueryNestedLevels)
	if err != nil {
		return nil, err
	}
	queryComplexityHistogram.Observe(float64(complexity))

	// Max Complexity of 0 disables rate limiting
	if h.maxGraphqlComplexity == 0 {
		return &queryComplexityResult{
			Throttled:          false,
			RequestedQueryCost: complexity,
			MaxQueryCost:       h.maxGraphqlComplexity,
			Remaining:          0,
		}, nil
	}

	// TakeMany determines if the query needs to be rate limited
	_, remaining, _, ok, err := h.rateLimitStore.TakeMany(ctx, "graphql-"+subject, uint64(complexity))
	if err != nil {
		return nil, err
	}

	return &queryComplexityResult{
		Throttled:          !ok,
		RequestedQueryCost: complexity,
		MaxQueryCost:       h.maxGraphqlComplexity,
		Remaining:          int(remaining),
	}, nil
}

// A request represents an HTTP request to the GraphQL endpoint.
// A request can have a single query or a batch of requests with one or more queries.
// It is important to distinguish between a single query request and a batch request with a single query.
// The shape of the response will differ in both cases.
type request struct {
	queries []query
	isBatch bool
}

// A query represents a single GraphQL query.
type query struct {
	Variables map[string]interface{} `json:"variables"`
	OpName    string                 `json:"operationName"`
	Query     string                 `json:"query"`
}

func respond(w http.ResponseWriter, body []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}

func isSupported(method string) bool {
	return method == "POST" || method == "GET"
}

func errorJSON(msg string) []byte {
	buf := bytes.Buffer{}
	fmt.Fprintf(&buf, `{"error": "%s"}`, msg)
	return buf.Bytes()
}
