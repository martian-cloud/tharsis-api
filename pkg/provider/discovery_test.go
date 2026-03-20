package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverTFEServices(t *testing.T) {
	type testCase struct {
		name               string
		responseBody       string
		responseStatus     int
		endpoint           string
		endpointSuffix     string
		expectError        bool
		expectServiceURL   string
		useServerURLPrefix bool
	}

	testCases := []testCase{
		{
			name:             "successful discovery",
			responseBody:     `{"providers.v1": "https://example.com/providers/v1"}`,
			responseStatus:   http.StatusOK,
			expectServiceURL: "https://example.com/providers/v1",
		},
		{
			name:           "non-200 status code",
			responseBody:   `{"error": "not found"}`,
			responseStatus: http.StatusNotFound,
			expectError:    true,
		},
		{
			name:           "invalid JSON",
			responseBody:   `{invalid json}`,
			responseStatus: http.StatusOK,
			expectError:    true,
		},
		{
			name:             "endpoint with path and query",
			endpointSuffix:   "/some/path?query=param",
			responseBody:     `{"providers.v1": "https://example.com/providers/v1"}`,
			responseStatus:   http.StatusOK,
			expectServiceURL: "https://example.com/providers/v1",
		},
		{
			name:             "endpoint without scheme",
			responseBody:     `{"providers.v1": "https://example.com/providers/v1"}`,
			responseStatus:   http.StatusOK,
			expectServiceURL: "https://example.com/providers/v1",
		},
		{
			name:        "empty endpoint",
			endpoint:    "",
			expectError: true,
		},
		{
			name:        "whitespace endpoint",
			endpoint:    "   ",
			expectError: true,
		},
		{
			name:        "invalid scheme",
			endpoint:    "ftp://example.com",
			expectError: true,
		},
		{
			name:        "no host",
			endpoint:    "https://",
			expectError: true,
		},
		{
			name:             "non-string service value",
			responseBody:     `{"providers.v1": "https://example.com/providers/v1", "other": 123}`,
			responseStatus:   http.StatusOK,
			expectServiceURL: "https://example.com/providers/v1",
		},
		{
			name:               "relative URL in discovery",
			responseBody:       `{"providers.v1": "/v1/providers"}`,
			responseStatus:     http.StatusOK,
			useServerURLPrefix: true,
			expectServiceURL:   "/v1/providers",
		},
		{
			name:               "relative URL with endpoint path",
			endpointSuffix:     "/api/v2",
			responseBody:       `{"providers.v1": "/v1/providers"}`,
			responseStatus:     http.StatusOK,
			useServerURLPrefix: true,
			expectServiceURL:   "/v1/providers",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Validation errors don't need a server
			if test.endpoint != "" && test.expectError {
				discoverer := NewServiceDiscoverer(http.DefaultClient)
				_, err := discoverer.DiscoverTFEServices(t.Context(), test.endpoint)
				require.Error(t, err)
				return
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, discoveryPath, r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)

				w.WriteHeader(test.responseStatus)
				w.Write([]byte(test.responseBody))
			}))
			defer server.Close()

			endpoint := server.URL + test.endpointSuffix

			discoverer := NewServiceDiscoverer(server.Client())
			discovered, err := discoverer.DiscoverTFEServices(t.Context(), endpoint)

			if test.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, discovered)

			serviceURL, ok := discovered.Services[ProvidersServiceID]
			require.True(t, ok)

			expectedURL := test.expectServiceURL
			if test.useServerURLPrefix {
				expectedURL = server.URL + test.expectServiceURL
			}
			assert.Equal(t, expectedURL, serviceURL.String())
		})
	}
}
