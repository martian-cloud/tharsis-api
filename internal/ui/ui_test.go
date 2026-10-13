//go:build !noui

package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestServeHTTPSetsSecurityHeaders(t *testing.T) {
	handler := &Handler{
		uiFS: fstest.MapFS{
			"index.html":    {Data: []byte("<html></html>")},
			"assets/app.js": {Data: []byte("console.log('hi')")},
		},
	}

	testCases := []struct {
		name string
		path string
	}{
		{name: "static file", path: "/assets/app.js"},
		{name: "spa fallback", path: "/groups/some-group"},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, test.path, nil))

			assert.Equal(t, http.StatusOK, rec.Code)
			for k, v := range securityHeaders {
				assert.Equal(t, v, rec.Header().Get(k), k)
			}
		})
	}
}
