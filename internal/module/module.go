// Package module package
package module

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	svchost "github.com/hashicorp/terraform-svchost"
)

const (
	https            = "https" // could not find a net/http-supplied constant
	wellKnownPath    = "/.well-known/terraform.json"
	tfTokenVarPrefix = "TF_TOKEN_"
	// maxDiscoveryDocumentSize caps the service discovery document read; the host is user-supplied.
	maxDiscoveryDocumentSize = 1024 * 1024
)

// GetModuleRegistryEndpointForHost returns the API path on the server
// for example, https://somehost.somecompany.com/.well-known/terraform.json
// returns something like {"modules.v1":"/api/v4/packages/terraform/modules/v1/"}
func GetModuleRegistryEndpointForHost(httpClient *http.Client, host string) (*url.URL, error) {
	url := url.URL{
		Scheme: https,
		Host:   host,
		Path:   wellKnownPath,
	}
	urlString := url.String()
	resp, err := httpClient.Get(urlString)
	if err != nil {
		return nil, fmt.Errorf("failed to visit well-known URL: %s: %v", urlString, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not-ok status from well-known URL: %s: %s", urlString, resp.Status)
	}

	body, err := ReadAllLimited(resp.Body, maxDiscoveryDocumentSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of well-known URL: %s: %v", urlString, err)
	}

	var items struct {
		ModulesV1 string `json:"modules.v1"`
	}
	err = json.Unmarshal(body, &items)
	if err != nil {
		// The body is left out: the host is user-supplied, so echoing it would let a caller read
		// whatever the address returned.
		return nil, fmt.Errorf("failed to unmarshal body of well-known URL: %s", urlString)
	}

	moduleURL, err := url.Parse(items.ModulesV1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module registry path %s: %v", items.ModulesV1, err)
	}

	if moduleURL.Host == "" {
		moduleURL.Host = host
	}

	if moduleURL.Scheme == "" {
		moduleURL.Scheme = https
	}

	// Remove leading slash if present.
	moduleURL.Path = strings.TrimPrefix(moduleURL.Path, "/")

	// Add trailing slash if not present--needed to make relative reference resolution work.
	if !strings.HasSuffix(moduleURL.Path, "/") {
		moduleURL.Path += "/"
	}

	return moduleURL, nil
}

// ReadAllLimited reads r to the end, failing if it holds more than limit bytes. io.LimitReader alone
// can't tell the two apart: it reports io.EOF at the limit, so a longer body would come back
// silently truncated. Reading one byte past the limit shows whether there was more.
func ReadAllLimited(r io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds the maximum size of %d bytes", limit)
	}
	return body, nil
}

// BuildTokenEnvVar builds the environment variable name to supply the authorization token for the specified host.
// For reasoning for implementation - https://www.terraform.io/cli/config/config-file#environment-variable-credentials
func BuildTokenEnvVar(host string) (string, error) {
	// Use HashiCorp's svchost package to help us consistently convert from unicode to ASCII using punycode.
	hostname, err := svchost.ForComparison(host)
	if err != nil {
		return "", err
	}

	// Periods must be encoded as underscores
	encHost := strings.ReplaceAll(hostname.String(), ".", "_")

	// Hyphens are usually invalid as variable names, se we encode them as double underscores
	encHost = strings.ReplaceAll(encHost, "-", "__")

	// Build the environment variable by prefixing with `TF_TOKEN_` to the encoded hostname
	return tfTokenVarPrefix + encHost, nil
}
