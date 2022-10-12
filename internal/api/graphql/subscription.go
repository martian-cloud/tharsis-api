package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	graphqlgo "github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-transport-ws/graphqlws"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/resolver"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
)

func newSubscriptionHandler(
	httpHandler http.Handler,
	schema *graphqlgo.Schema,
	authenticator *auth.Authenticator,
	resolverState *resolver.State,
	loaders *loader.Collection,
) http.HandlerFunc {
	return graphqlws.NewHandlerFunc(
		&graphqlSubscriptions{schema: schema, authenticator: authenticator},
		httpHandler,
		graphqlws.WithContextGenerator(&contextGenerator{
			resolverState: resolverState,
			loaders:       loaders,
			// Disable cache for subscriptions since they don't refresh the context per response
			disableCache: true,
		}))
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
	authenticator *auth.Authenticator
}

var graphqlSubscriptionCount = metric.NewCounter("graphql_subscription_count", "Amount of GraphQL Subscriptions.")

func (g *graphqlSubscriptions) Subscribe(ctx context.Context, document string, operationName string, variableValues map[string]interface{}) (payloads <-chan interface{}, err error) {
	msg, ok := ctx.Value("Header").(json.RawMessage)
	if !ok {
		return nil, errors.NewError(errors.EUnauthorized, "Missing Authorization header")
	}
	var params connectionParams
	if err = json.Unmarshal(msg, &params); err != nil {
		return nil, errors.NewError(errors.EInvalid, "Failed to decode connection params")
	}

	caller, err := g.authenticator.Authenticate(ctx, params.findToken(), false)
	if err != nil {
		return nil, errors.NewError(errors.EUnauthorized, fmt.Sprintf("Unauthorized: %v", err))
	}

	graphqlSubscriptionCount.Inc()
	return g.schema.Subscribe(auth.WithCaller(ctx, caller), document, operationName, variableValues)
}
