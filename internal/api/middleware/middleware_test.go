package middleware

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsGraphqlSubscriptionRequest(t *testing.T) {
	type testCase struct {
		name         string
		method       string
		path         string
		upgrade      string
		connection   string
		subprotocols []string
		expect       bool
	}

	testCases := []testCase{
		{
			name:         "graphql-transport-ws subprotocol (graphql-ws client)",
			subprotocols: []string{"graphql-transport-ws"},
			expect:       true,
		},
		{
			name:         "legacy graphql-ws subprotocol is rejected",
			subprotocols: []string{"graphql-ws"},
			expect:       false,
		},
		{
			name:         "subprotocol match is case sensitive (not negotiated by the upgrader)",
			subprotocols: []string{"GraphQL-Transport-WS"},
			expect:       false,
		},
		{
			name:         "comma separated subprotocol list",
			subprotocols: []string{"other, graphql-transport-ws"},
			expect:       true,
		},
		{
			name:         "only the first Sec-Websocket-Protocol header is read",
			subprotocols: []string{"other", "graphql-transport-ws"},
			expect:       false,
		},
		{
			name:         "unsupported subprotocol",
			subprotocols: []string{"other"},
			expect:       false,
		},
		{
			name:   "no subprotocol",
			expect: false,
		},
		{
			name:         "not a websocket upgrade",
			upgrade:      "-",
			subprotocols: []string{"graphql-transport-ws"},
			expect:       false,
		},
		{
			name:         "connection header is not upgrade",
			connection:   "keep-alive",
			subprotocols: []string{"graphql-transport-ws"},
			expect:       false,
		},
		{
			name:         "wrong path",
			path:         "/v1/graphql",
			subprotocols: []string{"graphql-transport-ws"},
			expect:       false,
		},
		{
			name:         "not a GET request",
			method:       http.MethodPost,
			subprotocols: []string{"graphql-transport-ws"},
			expect:       false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			method, path, upgrade, connection := http.MethodGet, "/graphql", "websocket", "Upgrade"
			if test.method != "" {
				method = test.method
			}
			if test.path != "" {
				path = test.path
			}
			if test.upgrade == "-" {
				upgrade = ""
			}
			if test.connection != "" {
				connection = test.connection
			}

			r, err := http.NewRequest(method, path, nil)
			assert.NoError(t, err)
			if upgrade != "" {
				r.Header.Set("Upgrade", upgrade)
			}
			r.Header.Set("Connection", connection)
			for _, p := range test.subprotocols {
				r.Header.Add("Sec-Websocket-Protocol", p)
			}

			assert.Equal(t, test.expect, isGraphqlSubscriptionRequest(r))
		})
	}
}
