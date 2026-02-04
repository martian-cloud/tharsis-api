package provider

import (
	"io"
	"log"
	"net/url"

	svchost "github.com/hashicorp/terraform-svchost"
)

// serviceDiscoverer is an interface for discovering service URLs.
type serviceDiscoverer interface {
	DiscoverServiceURL(host svchost.Hostname, serviceID string) (*url.URL, error)
}

// quietDisco wraps a serviceDiscoverer to silence debug logging from the disco package.
type quietDisco struct {
	inner serviceDiscoverer
}

func (q *quietDisco) DiscoverServiceURL(host svchost.Hostname, serviceID string) (*url.URL, error) {
	original := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(original)
	return q.inner.DiscoverServiceURL(host, serviceID)
}
