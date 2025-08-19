package middleware

import (
	"net/http"
	"strings"
)

// Handler is used for returning middleware functions
type Handler func(next http.Handler) http.Handler

func isGraphqlSubscriptionRequest(r *http.Request) bool {
	return r.URL.Path == "/graphql" &&
		strings.EqualFold(r.Method, "GET") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket") &&
		strings.EqualFold(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(r.Header.Get("Sec-Websocket-Protocol"), "graphql-ws")
}
