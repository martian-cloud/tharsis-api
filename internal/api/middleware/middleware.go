package middleware

import (
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// Handler is used for returning middleware functions
type Handler func(next http.Handler) http.Handler

func isGraphqlSubscriptionRequest(r *http.Request) bool {
	return r.URL.Path == "/graphql" &&
		strings.EqualFold(r.Method, "GET") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		hasGraphqlSubprotocol(r)
}

// hasGraphqlSubprotocol reports whether the request asks for the GraphQL over
// WebSocket subprotocol "graphql-transport-ws" (used by the graphql-ws client).
// The legacy "graphql-ws" subprotocol is no longer supported.
//
// It uses websocket.Subprotocols and an exact, case-sensitive match so this agrees
// with what the upgrade handler will actually negotiate (gorilla reads only the first
// Sec-Websocket-Protocol header and matches case-sensitively). Otherwise a request
// could skip CSRF here but then fail to upgrade and fall back to the HTTP handler.
func hasGraphqlSubprotocol(r *http.Request) bool {
	for _, protocol := range websocket.Subprotocols(r) {
		if protocol == "graphql-transport-ws" {
			return true
		}
	}
	return false
}
