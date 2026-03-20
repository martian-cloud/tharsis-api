package provider

import (
	context "context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// The well-known document to look for.
	discoveryPath = "/.well-known/terraform.json"
)

// ServiceID is the key for the service to return the URL for from the well-known document.
type ServiceID string

// ServiceID consts.
const (
	ProvidersServiceID ServiceID = "providers.v1"
	TFEServiceID       ServiceID = "tfe.v2"
)

// TFEServices contains discovered service URLs from a Terraform registry discovery document.
type TFEServices struct {
	Services map[ServiceID]*url.URL
}

//go:generate go tool mockery --name ServiceDiscoverer --inpackage --case underscore

// ServiceDiscoverer is an interface for discovering service URLs.
// This is a custom implementation of Terraform's disco package that supports
// localhost environments and discovers multiple services in a single request
// from the /.well-known/terraform.json endpoint.
type ServiceDiscoverer interface {
	DiscoverTFEServices(ctx context.Context, endpoint string) (*TFEServices, error)
}

type discoverer struct {
	httpClient *http.Client
}

// NewServiceDiscoverer returns a new ServiceDiscoverer.
func NewServiceDiscoverer(httpClient *http.Client) ServiceDiscoverer {
	return &discoverer{httpClient: httpClient}
}

// DiscoverTFEServices discovers the available services at the provided endpoint.
func (d *discoverer) DiscoverTFEServices(ctx context.Context, endpoint string) (*TFEServices, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	// Prepend https:// if no scheme is present.
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}

	endpointURL, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Only allow http and https schemes.
	if endpointURL.Scheme != "http" && endpointURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme: %s (only http and https are allowed)", endpointURL.Scheme)
	}

	// Replace the path with the discovery path.
	endpointURL.Path = discoveryPath
	endpointURL.RawQuery = ""

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't build request: %w", err)
	}

	response, err := d.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("couldn't perform request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		return nil, fmt.Errorf("received status code %d from well-known URL: %s", response.StatusCode, string(body))
	}

	var discoveredServices map[string]any
	if err = json.NewDecoder(response.Body).Decode(&discoveredServices); err != nil {
		return nil, fmt.Errorf("failed to decode TFE discovery document: %w", err)
	}

	// Extract service URLs from the discovery document.
	discovered := make(map[ServiceID]*url.URL)

	for serviceID, serviceValue := range discoveredServices {
		val, ok := serviceValue.(string)
		if !ok {
			// Skip non-string values.
			continue
		}

		// Parse the service URL.
		serviceURL, err := url.Parse(val)
		if err != nil {
			// Skip invalid URLs.
			continue
		}

		// Resolve relative URLs against the endpoint. Absolute URLs are returned unchanged.
		if serviceURL.Scheme == "" || serviceURL.Host == "" {
			serviceURL = endpointURL.ResolveReference(serviceURL)
		}

		discovered[ServiceID(serviceID)] = serviceURL
	}

	return &TFEServices{Services: discovered}, nil
}
