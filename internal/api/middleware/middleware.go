package middleware

import "net/http"

// Handler is used for returning middleware functions
type Handler func(next http.Handler) http.Handler
