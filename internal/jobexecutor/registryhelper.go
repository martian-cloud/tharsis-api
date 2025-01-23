package jobexecutor

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
)

const (
	xTerraformGet = "x-terraform-get"
)

// resolveModuleSource returns the final pre-signed URL for a module source.
func resolveModuleSource(moduleSource string, moduleVersion string, hostTokenMap map[string]string) (string, error) {
	// Separate the pieces of the source module string.
	// These are equal to the fields of Terraform's addrs.ModuleSourceRegistry.PackageAddr.
	parts := strings.Split(moduleSource, "/")
	if len(parts) != 4 {
		return "", fmt.Errorf("failed to parse module source into necessary 4 parts: %s", moduleSource)
	}
	host := parts[0]
	namespace := parts[1]
	sourcePath := parts[2]
	targetSystem := parts[3]

	// Get the auth token for the specified host.
	wantVar, err := module.BuildTokenEnvVar(host)
	if err != nil {
		return "", fmt.Errorf("unable to find an authorization token for host %s; invalid host %v", host, err)
	}

	var token *string
	if t, ok := hostTokenMap[wantVar]; ok {
		// Use token if one is provided.
		token = &t
	}

	// Create an HTTP client to use for the next requests:
	httpClient := tharsishttp.NewHTTPClient()

	// Visit the 'well-known' URL for the server in question:
	registryURL, err := module.GetModuleRegistryEndpointForHost(httpClient, host)
	if err != nil {
		return "", err
	}

	// Relative reference based from the above:
	moreRefURL, err := url.Parse(strings.Join([]string{namespace, sourcePath, targetSystem, moduleVersion, "download"}, "/"))
	if err != nil {
		return "", fmt.Errorf("failed to parse relative reference for leading URL: %v", err)
	}

	// Visit the URL to get the pre-authorized URL for the desired version:
	preSignedURL, err := getPreSignedURL(*httpClient, token, registryURL.ResolveReference(moreRefURL))
	if err != nil {
		return "", err
	}

	return preSignedURL, nil
}

// getPreSignedURL returns a string of the pre-signed URL to download the actual module content
// for example, https://gitlab.com/api/v4/packages/terraform/modules/v1/mygroup/module-001/aws/0.0.1/download
func getPreSignedURL(httpClient http.Client, token *string, registryURL *url.URL) (string, error) {
	downloadURLString := registryURL.String()

	req, err := http.NewRequest(http.MethodGet, downloadURLString, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate the download GET request: %s", downloadURLString)
	}

	if token != nil {
		req.Header.Set("AUTHORIZATION", fmt.Sprintf("Bearer %s", *token))
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to visit download URL %s: %v", downloadURLString, err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		envVar, bErr := module.BuildTokenEnvVar(registryURL.Host)
		if bErr != nil {
			return "", bErr
		}

		if token != nil {
			// Since we were able to make the request with a token we can assume the host is correct but token is bad.
			return "", fmt.Errorf("token in environment variable '%s' is apparently not authorized to access this module", envVar)
		}
		// Required token environment variable was not provided.
		return "", fmt.Errorf("missing required environment variable '%s' for host %s", envVar, registryURL.Host)
	}
	if resp.StatusCode != http.StatusNoContent {
		return "", fmt.Errorf("not-ok status from download URL: %s: %s", downloadURLString, resp.Status)
	}

	resultPathQuery := resp.Header.Get(xTerraformGet)
	if resultPathQuery == "" {
		return "", fmt.Errorf("failed to get final URL from download URL: %s", downloadURLString)
	}

	// Generate the final (relative) reference URL.
	resultRefURL, err := url.Parse(resultPathQuery)
	if err != nil {
		return "", fmt.Errorf("failed to parse final URL path and query: %s", resultPathQuery)
	}

	// Resolve to the final URL.
	finalURL := registryURL.ResolveReference(resultRefURL)

	return finalURL.String(), nil
}
