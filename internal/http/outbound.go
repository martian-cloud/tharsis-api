// Package http restricts the API's outbound HTTP and gRPC connections. Server-side code that
// connects to a user-supplied or configured URL must use an OutboundPolicy.
package http

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"syscall"
	"time"

	"code.dny.dev/ssrf"
)

const httpClientTimeout = 5 * time.Minute

// awsMetadataV6 is the AWS instance metadata endpoint over IPv6. It is inside fc00::/7, which allowed
// hosts may reach, so it is denied explicitly; the IPv4 endpoint is link-local and always denied.
var awsMetadataV6 = netip.MustParseAddr("fd00:ec2::254")

// publicGuardian allows public addresses only. ssrf's deny list comes from the IANA special-purpose
// registries; for IPv6, anything outside global unicast is denied.
var publicGuardian = ssrf.New(ssrf.WithAnyPort())

// allowedHostGuardian also allows the private and loopback ranges, for the policy's allowed hosts.
var allowedHostGuardian = ssrf.New(
	ssrf.WithAnyPort(),
	ssrf.WithAllowedV4Prefixes(
		netip.MustParsePrefix("127.0.0.0/8"),
		netip.MustParsePrefix("10.0.0.0/8"),
		netip.MustParsePrefix("172.16.0.0/12"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("100.64.0.0/10"), // carrier-grade NAT, used for internal services by some clouds
		netip.MustParsePrefix("198.18.0.0/15"), // benchmarking (RFC 2544), sometimes used internally
	),
	ssrf.WithAllowedV6Prefixes(
		netip.MustParsePrefix("::1/128"),
		netip.MustParsePrefix("fc00::/7"), // unique local
	),
)

type dialHostKey struct{}

// OutboundPolicy restricts which addresses the API connects to when a request's destination comes
// from user input (module and provider registries, VCS providers, OIDC issuers, federated
// registries), so those requests can't be pointed at the API's own network.
//
// The check runs when the connection is made, against the resolved IP, so it covers every redirect
// hop and a hostname that resolves differently on a later lookup. Only public addresses are allowed,
// except that the allowed hosts may also reach private and loopback addresses. Link-local (including
// cloud metadata), multicast and other special-purpose addresses are always denied. A NAT64 address
// is checked as the IPv4 address it embeds.
type OutboundPolicy struct {
	hosts  []string
	dialer *net.Dialer
	client *http.Client
}

// NewOutboundPolicy builds a policy that allows the hosts of selfURLs, the API's own URLs and the
// issuers an admin configured, on private networks. The API fetches its own service discovery
// document to resolve its own modules.
func NewOutboundPolicy(selfURLs ...string) *OutboundPolicy {
	p := &OutboundPolicy{}

	for _, selfURL := range selfURLs {
		if host := hostOf(selfURL); host != "" {
			p.hosts = append(p.hosts, host)
		}
	}

	p.dialer = &net.Dialer{
		Timeout:        30 * time.Second,
		KeepAlive:      30 * time.Second,
		ControlContext: p.control,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	// A proxy would be the only address the check could see.
	transport.Proxy = nil
	transport.DialContext = p.DialContext
	p.client = &http.Client{Timeout: httpClientTimeout, Transport: transport}

	return p
}

// DialContext dials addr after checking the resolved IP against the policy. Use it for clients that
// don't go through HTTPClient, such as gRPC; the policy can be passed wherever a dialer is expected.
// The allowed hosts are matched against the hostname in addr, so a gRPC client must hand the dialer
// the hostname (the passthrough resolver, as pkg/client does when given a dialer), not an IP it
// resolved itself.
func (p *OutboundPolicy) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return p.dialer.DialContext(context.WithValue(ctx, dialHostKey{}, strings.ToLower(host)), network, addr)
}

// HTTPClient returns the HTTP client whose connections are restricted by the policy. It is shared,
// so every caller uses one connection pool. It ignores proxy environment variables.
func (p *OutboundPolicy) HTTPClient() *http.Client {
	return p.client
}

// control runs after DNS resolution with the IP actually being connected to.
func (p *OutboundPolicy) control(ctx context.Context, _, address string, _ syscall.RawConn) error {
	ipStr, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return err
	}
	host, _ := ctx.Value(dialHostKey{}).(string)

	if err := p.checkAddr(host, addr); err != nil {
		return fmt.Errorf("connection to %s (%s) refused: %w", host, addr, err)
	}
	return nil
}

func (p *OutboundPolicy) checkAddr(host string, addr netip.Addr) error {
	if addr == awsMetadataV6 {
		return fmt.Errorf("cloud metadata addresses are not allowed")
	}
	// A NAT64 address (RFC 6052, used by DNS64 on IPv6-only networks) reaches the IPv4 address it
	// embeds, so check that address instead. ssrf would otherwise deny every NAT64 address, cutting
	// IPv6-only deployments off from IPv4-only hosts, and allowing the prefix outright would let it reach
	// a loopback or private IPv4 address through the gateway. The local-use prefix 64:ff9b:1::/48 is
	// left denied.
	if ssrf.IPv6NAT64Prefix.Contains(addr) {
		b := addr.As16()
		addr = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
	}
	if p.hostAllowed(host) {
		return allowedHostGuardian.SafeAddr(addr)
	}
	return publicGuardian.SafeAddr(addr)
}

func (p *OutboundPolicy) hostAllowed(host string) bool {
	return host != "" && slices.Contains(p.hosts, host)
}

// hostOf returns the lowercased hostname of a URL, or "" if it has none.
func hostOf(rawURL string) string {
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
