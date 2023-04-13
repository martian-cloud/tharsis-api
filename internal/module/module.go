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
