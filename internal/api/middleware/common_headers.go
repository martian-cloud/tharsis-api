// Package middleware package
package middleware

import (
	"net/http"
)

var _ http.ResponseWriter = (*customResponseWriter)(nil)

type customResponseWriter struct {
	w             http.ResponseWriter
	commonHeaders map[string]string
}

func (s customResponseWriter) WriteHeader(code int) {
	for k, v := range s.commonHeaders {
		s.Header().Set(k, v)
	}
	s.w.WriteHeader(code)
}

func (s customResponseWriter) Write(b []byte) (int, error) {
	return s.w.Write(b)
}

func (s customResponseWriter) Header() http.Header {
	return s.w.Header()
}

// NewCommonHeadersMiddleware creates an instance of CommonHeadersMiddleware
func NewCommonHeadersMiddleware(
	headers map[string]string,
) Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&customResponseWriter{
				w:             w,
				commonHeaders: headers,
			}, r)
		})
	}
}
