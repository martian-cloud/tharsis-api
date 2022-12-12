package module

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	https            = "https" // could not find a net/http-supplied constant
	wellKnownPath    = "/.well-known/terraform.json"
	tfTokenVarPrefix = "TF_TOKEN_"
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
		return nil, fmt.Errorf("failed to visit well-known URL: %s", urlString)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not-ok status from well-known URL: %s: %s", urlString, resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of well-known URL: %s", urlString)
	}

	var items struct {
		ModulesV1 string `json:"modules.v1"`
	}
	err = json.Unmarshal(body, &items)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body of well-known URL: %s: %s", urlString, body)
	}

	moduleURL, err := url.Parse(items.ModulesV1)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module registry path %s: %v", items.ModulesV1, err)
	}

	if moduleURL.Host == "" {
		moduleURL.Host = host
	}

	if moduleURL.Scheme == "" {
		moduleURL.Scheme = "https"
	}

	// Remove leading slash if present.
	moduleURL.Path = strings.TrimPrefix(moduleURL.Path, "/")

	// Add trailing slash if not present--needed to make relative reference resolution work.
	if !strings.HasSuffix(moduleURL.Path, "/") {
		moduleURL.Path += "/"
	}

	return moduleURL, nil
}

// TODO: Post-MVP, other character conversions are required.  See here:
// https://www.terraform.io/cli/config/config-file#environment-variable-credentials
// Also, it could be argued this function should be moved to its own module within this package.

// BuildTokenEnvVar builds the environment variable name to supply the authorization token for the specified host.
func BuildTokenEnvVar(host string) string {
	return tfTokenVarPrefix + strings.ReplaceAll(host, ".", "_")
}

// The End.
