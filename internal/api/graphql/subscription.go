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
type subscriptionAuthenticator struct {
	tharsisUIOrigin       string
	allowedOrigins        map[string]struct{}
	allowAnyOrigin        bool
	accessTokenCookieName string
}

func newSubscriptionAuthenticator(tharsisUIOrigin string, allowedOrigins []string, accessTokenCookieName string) *subscriptionAuthenticator {
	a := &subscriptionAuthenticator{
		tharsisUIOrigin:       tharsisUIOrigin,
		allowedOrigins:        map[string]struct{}{},
		accessTokenCookieName: accessTokenCookieName,
	}
	for _, origin := range allowedOrigins {
		origin = normalizeOrigin(origin)
		switch origin {
		case "":
		case "*":
			a.allowAnyOrigin = true
		default:
			a.allowedOrigins[origin] = struct{}{}
		}
	}
	return a
}

// normalizeOrigin lowercases an origin and drops a trailing slash, since a configured URL may have
// one and the browser's Origin header never does.
func normalizeOrigin(origin string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(origin), "/"))
}

func (c *subscriptionAuthenticator) isAllowedOrigin(origin string) bool {
	if c.allowAnyOrigin {
		return true
	}
	_, ok := c.allowedOrigins[normalizeOrigin(origin)]
	return ok
}

func (c *subscriptionAuthenticator) BuildContext(ctx context.Context, r *http.Request) (context.Context, error) {
	// subscriptions must always be authenticated
	caller, err := auth.AuthorizeCaller(r.Context())
	if err != nil && r.Header.Get("Origin") == c.tharsisUIOrigin {
		// We only return the error here if the request has an invalid caller and is coming from the Tharsis UI.
		// This allows other clients using the SDK to pass the auth token via the connection params instead of using cookie based authentication
		return nil, err
	}

	if caller != nil {
		// A browser attaches the session cookie to a websocket upgrade from any same-site page, and
		// websockets are not covered by CORS, so a page that can open the socket can read everything
		// sent over it. A connection authenticated by the session cookie is therefore only accepted
		// from an allowed origin. A token in the Authorization header or in the connection params
		// cannot be supplied by a browser on the user's behalf, so those clients need no Origin.
		if cookie, _ := r.Cookie(c.accessTokenCookieName); cookie != nil && !c.isAllowedOrigin(r.Header.Get("Origin")) {
			return nil, errors.New("websocket connection origin is not allowed", errors.WithErrorCode(errors.EForbidden))
		}

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
	tharsisUIOrigin string,
	allowedOrigins []string,
	accessTokenCookieName string,
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
		graphqlws.WithContextGenerator(newSubscriptionAuthenticator(tharsisUIOrigin, allowedOrigins, accessTokenCookieName)),
		graphqlws.WithReadLimit(graphqlSubscriptionMaxMessageSize),
		graphqlws.WithWriteTimeout(graphqlSubscriptionWriteTimeout),
		graphqlws.WithConnectionTimeout(maxConnectionDuration),
		graphqlws.WithConnectionInitHandler(func(ctx context.Context, message json.RawMessage) (context.Context, error) {
			caller := auth.GetCaller(ctx)
			// A caller may already be set if this connection was authenticated using a cookie based user session. If a caller is not set,
			// we'll verify that a valid authorization token is present in the connection params.
			if caller == nil {
				// Verify that the connection params are valid if a caller is not already set
				var params connectionParams
				if err := json.Unmarshal(message, &params); err != nil {
					return nil, errors.New("Failed to decode connection params", errors.WithErrorCode(errors.EInvalid))
				}

				token := params.findToken()

				if token == "" {
					return nil, errors.New("missing authorization token in connection params", errors.WithErrorCode(errors.EUnauthorized))
				}

				caller, err := authenticator.Authenticate(ctx, token, false)
				if err != nil {
					return nil, errors.Wrap(err, "unauthorized", errors.WithErrorCode(errors.EUnauthorized))
				}

				// Add caller to context so that subscriptions will be authenticated
				return auth.WithCaller(ctx, caller), nil
			}

			return ctx, nil
		}),
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
	graphqlSubscriptionCount.Inc()
	return g.schema.Subscribe(ctx, document, operationName, variableValues)
}
