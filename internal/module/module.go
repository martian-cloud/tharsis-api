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
func GetModuleRegistryEndpointForHost(httpClient *http.Client, server string) (string, error) {
	url := url.URL{
		Scheme: https,
		Host:   server,
		Path:   wellKnownPath,
	}
	urlString := url.String()
	resp, err := httpClient.Get(urlString)
	if err != nil {
		return "", fmt.Errorf("failed to visit well-known URL: %s", urlString)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("not-ok status from well-known URL: %s: %s", urlString, resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read body of well-known URL: %s", urlString)
	}

	var items struct {
		ModulesV1 string `json:"modules.v1"`
	}
	err = json.Unmarshal(body, &items)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal body of well-known URL: %s: %s", urlString, body)
	}

	return items.ModulesV1, nil
}

// TODO: Post-MVP, other character conversions are required.  See here:
// https://www.terraform.io/cli/config/config-file#environment-variable-credentials
// Also, it could be argued this function should be moved to its own module within this package.

// BuildTokenEnvVar builds the environment variable name to supply the authorization token for the specified host.
func BuildTokenEnvVar(host string) string {
	return tfTokenVarPrefix + strings.ReplaceAll(host, ".", "_")
}

// The End.
