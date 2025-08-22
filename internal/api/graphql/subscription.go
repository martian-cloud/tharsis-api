package graphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	graphqlgo "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/graphql-transport-ws/graphqlws"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	graphqlSubscriptionMaxMessageSize = 1024 * 10
	graphqlSubscriptionWriteTimeout   = 10 * time.Second
)

// subscriptionAuthenticator is used to authenticate graphql subscription requests. The BuildContext function will be called
// when the connection is established; therefore, the caller will be added to the context if the request has a valid token.
type subscriptionAuthenticator struct{}

func (c *subscriptionAuthenticator) BuildContext(ctx context.Context, r *http.Request) (context.Context, error) {
	// subscriptions must always be authenticated
	caller, err := auth.AuthorizeCaller(r.Context())
	if err != nil && err != auth.ErrNoCaller {
		// We only return the error here if the request had an invalid caller. If the request
		// does not contain any caller then we continue since this subscription may be using connection
		// params to authenticate the caller.
		return nil, err
	}

	if caller != nil {
		// Add caller to context if it exists
		ctx = auth.WithCaller(ctx, caller)
	}

	return ctx, nil
}

func newSubscriptionHandler(
	httpHandler http.Handler,
	schema *graphqlgo.Schema,
	resolverState *resolver.State,
	loaders *loader.Collection,
	authenticator auth.Authenticator,
	maxConnectionDuration time.Duration,
) http.HandlerFunc {
	return graphqlws.NewHandlerFunc(
		&graphqlSubscriptions{schema: schema, authenticator: authenticator},
		httpHandler,
		graphqlws.WithContextGenerator(&contextGenerator{
			resolverState: resolverState,
			loaders:       loaders,
			// Disable cache for subscriptions since they don't refresh the context per response
			disableCache: true,
		}),
		graphqlws.WithContextGenerator(&subscriptionAuthenticator{}),
		graphqlws.WithReadLimit(graphqlSubscriptionMaxMessageSize),
		graphqlws.WithWriteTimeout(graphqlSubscriptionWriteTimeout),
		graphqlws.WithConnectionTimeout(maxConnectionDuration),
	)
}

type connectionParams struct {
	Authorization string `json:"Authorization"`
}

func (c *connectionParams) findToken() string {
	// Get token from authorization header.
	bearer := c.Authorization
	if len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER" {
		return bearer[7:]
	}
	return ""
}

type graphqlSubscriptions struct {
	schema        *graphqlgo.Schema
	authenticator auth.Authenticator
}

var graphqlSubscriptionCount = metric.NewCounter("graphql_subscription_count", "Amount of GraphQL Subscriptions.")

func (g *graphqlSubscriptions) Subscribe(ctx context.Context, document string, operationName string, variableValues map[string]interface{}) (payloads <-chan interface{}, err error) {
	caller := auth.GetCaller(ctx)
	if caller == nil {
		// Attempt to authenticate this request using connection params
		msg, ok := ctx.Value("Header").(json.RawMessage)
		if !ok {
			return nil, errors.New("Missing Authorization header", errors.WithErrorCode(errors.EUnauthorized))
		}
		var params connectionParams
		if err = json.Unmarshal(msg, &params); err != nil {
			return nil, errors.New("Failed to decode connection params", errors.WithErrorCode(errors.EInvalid))
		}

		caller, err = g.authenticator.Authenticate(ctx, params.findToken(), false)
		if err != nil {
			return nil, errors.Wrap(err, "unauthorized", errors.WithErrorCode(errors.EUnauthorized))
		}

		ctx = auth.WithCaller(ctx, caller)
	}

	graphqlSubscriptionCount.Inc()
	return g.schema.Subscribe(ctx, document, operationName, variableValues)
}
