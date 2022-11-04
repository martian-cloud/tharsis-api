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
	https          = "https" // could not find a net/http-supplied constant
	suffixDownload = "download"
	xTerraformGet  = "x-terraform-get"
)

// resolveModuleSource returns the final pre-signed URL for a module source.
func resolveModuleSource(moduleSource string, moduleVersion string, variables map[string]string) (string, error) {

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
	wantVar := module.BuildTokenEnvVar(host)
	token, ok := variables[wantVar]
	if !ok {
		return "", fmt.Errorf("unable to find an authorization token for host %s; expected environment variable %s",
			host, wantVar)
	}

	// Create an HTTP client to use for the next requests:
	httpClient := tharsishttp.NewHTTPClient()

	// Visit the 'well-known' URL for the server in question:
	apiPath, err := module.GetModuleRegistryEndpointForHost(httpClient, host)
	if err != nil {
		return "", err
	}

	// Build the early leading part of the URL:
	// apiPath has a leading slash.
	earlyLeadingURL := url.URL{
		Scheme: https,
		Host:   host,
		Path:   apiPath,
	}

	// Relative reference based from the above:
	moreRefURL, err := url.Parse(strings.Join([]string{namespace, sourcePath, targetSystem, ""}, "/"))
	if err != nil {
		return "", fmt.Errorf("failed to parse relative reference for leading URL: %v", err)
	}

	// Visit the URL to get the pre-authorized URL for the desired version:
	preSignedURL, err := getPreSignedURL(*httpClient, token, host,
		earlyLeadingURL.ResolveReference(moreRefURL).Path, moduleVersion)
	if err != nil {
		return "", err
	}

	return preSignedURL, nil
}

// getPreSignedURL returns a string of the pre-signed URL to download the actual module content
// for example, https://gitlab.com/api/v4/packages/terraform/modules/v1/mygroup/module-001/aws/0.0.1/download
func getPreSignedURL(httpClient http.Client, token, host, leadingPath, version string) (string, error) {

	// The common base URL, used twice below.  It needs to be the base URL for
	// both the "... download" and final URLs.  Registry protocol documentation,
	// says the value of x-terraform-get may be a relative URL:
	// https://www.terraform.io/internals/module-registry-protocol
	baseURL := url.URL{
		Scheme: https,
		Host:   host,
		Path:   leadingPath,
	}

	// Resolve a relative reference from the base URL to the download path.
	downloadRefString := strings.Join([]string{version, suffixDownload}, "/")
	downloadRefURL, err := url.Parse(downloadRefString)
	if err != nil {
		return "", fmt.Errorf("failed to parse download reference string to URL: %s", downloadRefString)
	}
	downloadURLString := baseURL.ResolveReference(downloadRefURL).String()

	req, err := http.NewRequest(http.MethodGet, downloadURLString, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate the download GET request: %s", downloadURLString)
	}
	req.Header.Set("AUTHORIZATION", fmt.Sprintf("Bearer %s", token))

	resp, err := httpClient.Do(req)
	if err != nil {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", fmt.Errorf("token in environment variable %s is apparently not authorized to access this module",
				module.BuildTokenEnvVar(host))
		}
		return "", fmt.Errorf("failed to visit download URL: %s", downloadURLString)
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
	finalURL := baseURL.ResolveReference(resultRefURL)

	return finalURL.String(), nil
}

// The End.
