package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSourceIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string][]string
		expectIP   string
	}{
		{
			name:       "no proxy headers uses the connection address",
			remoteAddr: "10.0.0.5:4321",
			expectIP:   "10.0.0.5",
		},
		{
			name:       "IPv6 loopback is normalized",
			remoteAddr: "[::1]:4321",
			expectIP:   "127.0.0.1",
		},
		{
			name:       "single X-Forwarded-For entry is the client",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"203.0.113.7"}},
			expectIP:   "203.0.113.7",
		},
		{
			// The load balancer appends the address it saw to the client-supplied header.
			name:       "client-supplied X-Forwarded-For entries are ignored",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"1.2.3.4, 5.6.7.8, 203.0.113.7"}},
			expectIP:   "203.0.113.7",
		},
		{
			name:       "the last of several X-Forwarded-For lines is used",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"1.2.3.4", "5.6.7.8, 203.0.113.7"}},
			expectIP:   "203.0.113.7",
		},
		{
			// Load balancers pass Forwarded through untouched, so it is entirely client-controlled.
			name:       "Forwarded is ignored",
			remoteAddr: "10.0.0.5:4321",
			headers: map[string][]string{
				"Forwarded":       {"for=1.2.3.4"},
				"X-Forwarded-For": {"203.0.113.7"},
			},
			expectIP: "203.0.113.7",
		},
		{
			name:       "Forwarded alone falls back to the connection address",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"Forwarded": {"for=1.2.3.4"}},
			expectIP:   "10.0.0.5",
		},
		{
			name:       "an empty last X-Forwarded-For entry falls back to the connection address",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"1.2.3.4, "}},
			expectIP:   "10.0.0.5",
		},
		{
			// A per-connection source port would otherwise give every request a fresh bucket.
			name:       "a port on the X-Forwarded-For entry is dropped",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"1.2.3.4, 203.0.113.7:51234"}},
			expectIP:   "203.0.113.7",
		},
		{
			name:       "a port on a bracketed IPv6 X-Forwarded-For entry is dropped",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"[2001:db8::7]:51234"}},
			expectIP:   "2001:db8::7",
		},
		{
			name:       "a bare IPv6 X-Forwarded-For entry is unchanged",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"2001:db8::7"}},
			expectIP:   "2001:db8::7",
		},
		{
			name:       "a bracketed IPv6 X-Forwarded-For entry without a port is unbracketed",
			remoteAddr: "10.0.0.5:4321",
			headers:    map[string][]string{"X-Forwarded-For": {"[2001:db8::7]"}},
			expectIP:   "2001:db8::7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/v1/sessions", nil)
			req.RemoteAddr = test.remoteAddr
			for k, values := range test.headers {
				for _, v := range values {
					req.Header.Add(k, v)
				}
			}

			ip, err := getSourceIP(req)
			require.NoError(t, err)
			assert.Equal(t, test.expectIP, ip)
		})
	}
}
