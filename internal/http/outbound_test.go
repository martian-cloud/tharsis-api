package http

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutboundPolicy_CheckAddr(t *testing.T) {
	policy := NewOutboundPolicy("https://tharsis.example.com", "https://login.internal.example.com/realms/tharsis")

	tests := []struct {
		name  string
		host  string
		addr  string
		allow bool
	}{
		// The special-purpose ranges are ssrf's; these cases cover how the policy uses them.
		{name: "public IPv4", host: "registry.terraform.io", addr: "151.101.1.1", allow: true},
		{name: "public IPv6", host: "registry.terraform.io", addr: "2a04:4e42::1", allow: true},

		{name: "loopback", host: "evil.example.net", addr: "127.0.0.1"},
		{name: "IPv4-mapped loopback", host: "evil.example.net", addr: "::ffff:127.0.0.1"},
		{name: "private address", host: "evil.example.net", addr: "10.0.0.5"},
		{name: "carrier-grade NAT", host: "evil.example.net", addr: "100.64.0.1"},
		{name: "IPv6 unique local", host: "evil.example.net", addr: "fd12::1"},
		{name: "NAT64 of a public address", host: "registry.terraform.io", addr: "64:ff9b::9765:101", allow: true},
		{name: "NAT64 of loopback", host: "evil.example.net", addr: "64:ff9b::7f00:1"},
		{name: "NAT64 of a private address", host: "evil.example.net", addr: "64:ff9b::a00:5"},
		{name: "NAT64 of cloud metadata on an allowed host", host: "tharsis.example.com", addr: "64:ff9b::a9fe:a9fe"},
		{name: "NAT64 of a private address on an allowed host", host: "tharsis.example.com", addr: "64:ff9b::a00:9", allow: true},
		{name: "local-use NAT64 prefix", host: "registry.terraform.io", addr: "64:ff9b:1::9765:101"},
		{name: "6to4", host: "registry.terraform.io", addr: "2002:7f00:1::1"},
		{name: "Teredo", host: "registry.terraform.io", addr: "2001:0:4136:e378:8000:63bf:3fff:fdd2"},
		{name: "IPv6 loopback", host: "evil.example.net", addr: "::1"},

		{name: "the API's own host may reach a private address", host: "tharsis.example.com", addr: "10.0.0.9", allow: true},
		{name: "a configured issuer may reach a private address", host: "login.internal.example.com", addr: "10.0.0.5", allow: true},
		{name: "an allowed host may reach loopback", host: "tharsis.example.com", addr: "127.0.0.1", allow: true},
		{name: "an allowed host may reach IPv6 loopback", host: "tharsis.example.com", addr: "::1", allow: true},
		{name: "an allowed host may reach carrier-grade NAT", host: "tharsis.example.com", addr: "100.64.0.1", allow: true},
		{name: "an allowed host may reach benchmarking addresses", host: "tharsis.example.com", addr: "198.18.0.1", allow: true},
		{name: "an allowed host may reach IPv6 unique local", host: "tharsis.example.com", addr: "fd12::1", allow: true},
		{name: "a subdomain of an allowed host is not allowed", host: "vcs.login.internal.example.com", addr: "10.0.0.5"},

		{name: "cloud metadata", host: "evil.example.net", addr: "169.254.169.254"},
		{name: "cloud metadata IPv6", host: "evil.example.net", addr: "fd00:ec2::254"},
		{name: "cloud metadata on an allowed host", host: "tharsis.example.com", addr: "169.254.169.254"},
		{name: "cloud metadata IPv6 on an allowed host", host: "tharsis.example.com", addr: "fd00:ec2::254"},
		{name: "link-local on an allowed host", host: "tharsis.example.com", addr: "fe80::1"},
		{name: "this network on an allowed host", host: "tharsis.example.com", addr: "0.0.0.1"},
		{name: "multicast on an allowed host", host: "tharsis.example.com", addr: "224.0.0.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := policy.checkAddr(test.host, netip.MustParseAddr(test.addr))
			if test.allow {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

func TestOutboundPolicy_HTTPClient(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	targetURL, err := url.Parse(target.URL)
	require.NoError(t, err)

	t.Run("loopback is refused by default", func(t *testing.T) {
		policy := NewOutboundPolicy()

		_, err = policy.HTTPClient().Get(target.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refused")
	})

	t.Run("the API's own URL is reachable", func(t *testing.T) {
		// Local development serves the API on localhost, and it fetches its own discovery document.
		policy := NewOutboundPolicy("http://localhost:" + targetURL.Port())

		resp, err := policy.HTTPClient().Get("http://localhost:" + targetURL.Port())
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("a redirect to a refused address is refused", func(t *testing.T) {
		// The check runs on every connection, so an allowed host can't redirect elsewhere.
		redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL, http.StatusFound)
		}))
		defer redirector.Close()
		redirectorURL, err := url.Parse(redirector.URL)
		require.NoError(t, err)

		// Only the hostname "localhost" is allowed; the redirect goes to the 127.0.0.1 literal.
		policy := NewOutboundPolicy("http://localhost")

		_, err = policy.HTTPClient().Get("http://localhost:" + redirectorURL.Port())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "refused")
	})
}

// Every caller shares one client, and so one connection pool.
func TestOutboundPolicy_HTTPClientIsShared(t *testing.T) {
	policy := NewOutboundPolicy()
	assert.Same(t, policy.HTTPClient(), policy.HTTPClient())
}
