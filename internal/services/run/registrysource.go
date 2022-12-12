package run

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	version "github.com/hashicorp/go-version"
	tfaddrs "github.com/hashicorp/terraform-registry-address"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/moduleregistry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/registry/addrs"
)

// ModuleResolver encapsulates the logic to resolve module source version string(s).
type ModuleResolver interface {
	ResolveModuleVersion(ctx context.Context, moduleSource string, moduleVersion *string, variables []Variable) (*string, error)
}

type moduleResolver struct {
	httpClient         *http.Client
	logger             logger.Logger
	moduleService      moduleregistry.Service
	tharsisAPIEndpoint string
}

// NewModuleResolver returns
func NewModuleResolver(moduleService moduleregistry.Service, httpClient *http.Client, logger logger.Logger, tharsiAPIEndpoint string) ModuleResolver {
	return &moduleResolver{
		moduleService:      moduleService,
		httpClient:         httpClient,
		logger:             logger,
		tharsisAPIEndpoint: tharsiAPIEndpoint,
	}
}

// ResolveModuleVersion parses a module source string.  Then, if necessary,
// it converts it to a go-getter-style source string.
//
// Note: In cases of a registry-style module source, if the module version was not specified
// by the caller, this function returns a pointer to the final module version.
func (m *moduleResolver) ResolveModuleVersion(ctx context.Context, moduleSource string, moduleVersion *string,
	variables []Variable) (*string, error) {

	// Determine if local module.
	if addrs.IsModuleSourceLocal(moduleSource) {
		return nil, fmt.Errorf("local modules are not supported")
	}

	// ParseModuleSource only supports module registry addresses.
	// This will never return an error to the caller, used as a means
	// of fallthrough.
	parsedSource, err := tfaddrs.ParseModuleSource(moduleSource)
	if err != nil {
		// The source string has been validated as already being a non-registry, remote, Go-Getter-type address.
		return nil, addrs.ValidateModuleSourceRemote(moduleSource)
	}

	return m.convertModuleSource(ctx, moduleVersion, parsedSource, variables)
}

// convertModuleSource intends to imitate some of the logic from function installRegistryModule
// from https://github.com/hashicorp/terraform/blob/main/internal/initwd/module_install.go
//
// The sequence of URLs to visit was found by downloading a module that pulls a submodule
// from a Gitlab registry with environment variable TF_LOG to 'trace'.
//
// Note: In cases of a registry-style module source, if the module version was not specified
// by the caller, this function returns a pointer to the final module version.
func (m *moduleResolver) convertModuleSource(ctx context.Context, version *string, sourceModule tfaddrs.Module,
	variables []Variable) (*string, error) {

	// Separate the pieces of sourceModule.
	host := sourceModule.Package.Host.String()
	subdir := sourceModule.Subdir

	// Subdir is not supported.
	if subdir != "" {
		return nil, fmt.Errorf("subdir not supported when reading module from registry")
	}

	// Get the auth token for the specified host.
	var token string
	seeking := module.BuildTokenEnvVar(host)
	for _, variable := range variables {
		if variable.Key == seeking {
			token = *variable.Value
		}
	}

	// Visit the 'well-known' URL for the server in question:
	moduleRegistryURL, err := module.GetModuleRegistryEndpointForHost(m.httpClient, host)
	if err != nil {
		return nil, err
	}

	// Visit the URL to get a list of versions:
	versions, err := m.getVersions(ctx, moduleRegistryURL, token, sourceModule)
	if err != nil {
		return nil, err
	}

	// Get or verify the version.
	chosenVersion, err := getLatestMatchingVersion(versions, version)
	if err != nil {
		return nil, err
	}

	return &chosenVersion, nil
}

// getVersions returns a slice of the versions available on the server
// for example, https://gitlab.com/api/v4/packages/terraform/modules/v1/mygroup/module-001/aws/versions
func (m *moduleResolver) getVersions(ctx context.Context, registryURL *url.URL, token string, sourceModule tfaddrs.Module) (map[string]bool, error) {
	namespace := sourceModule.Package.Namespace
	moduleName := sourceModule.Package.Name
	targetSystem := sourceModule.Package.TargetSystem

	apiURL, err := url.Parse(m.tharsisAPIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL %v", err)
	}

	if registryURL.Host == apiURL.Host {
		module, getModErr := m.moduleService.GetModuleByAddress(ctx, namespace, moduleName, targetSystem)
		if getModErr != nil {
			return nil, getModErr
		}

		statusFilter := models.TerraformModuleVersionStatusUploaded
		versionsResponse, getModVerErr := m.moduleService.GetModuleVersions(ctx, &moduleregistry.GetModuleVersionsInput{
			ModuleID: module.Metadata.ID,
			Status:   &statusFilter,
		})
		if getModVerErr != nil {
			return nil, getModVerErr
		}

		results := map[string]bool{}
		for _, m := range versionsResponse.ModuleVersions {
			results[m.SemanticVersion] = true
		}

		return results, nil
	}

	// Resolve a relative reference from the base URL to the 'versions' path.
	versionsRefURL, err := url.Parse(strings.Join([]string{namespace, moduleName, targetSystem, "versions"}, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse download reference string to URL: %v", err)
	}
	versionsURLString := registryURL.ResolveReference(versionsRefURL).String()

	req, err := http.NewRequest(http.MethodGet, versionsURLString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	if token != "" {
		req.Header.Set("AUTHORIZATION", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to visit versions URL: %s", versionsURLString)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("token in environment variable %s is not authorized to access this module",
			module.BuildTokenEnvVar(registryURL.Host))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("not-ok status from versions URL: %s: %s", versionsURLString, resp.Status)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of versions URL: %s", versionsURLString)
	}

	var unpacked struct {
		Modules []struct {
			Versions []struct {
				Version string `json:"version"`
			} `json:"versions"`
		} `json:"modules"`
	}

	err = json.Unmarshal(body, &unpacked)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body of versions URL: %s: %s", versionsURLString, body)
	}

	results := map[string]bool{}
	for _, m := range unpacked.Modules {
		for _, v := range m.Versions {
			results[v.Version] = true
		}
	}

	return results, nil
}

// getLatestMatchingVersion returns the checked version string for the convenience of the caller.
// If wantVersion is nil, it returns the latest version available.
// Otherwise, it returns the latest version that matches the wanted version constraints.
// However, it prefers an exact match if there is one.
func getLatestMatchingVersion(versions map[string]bool, wantVersion *string) (string, error) {

	// First, check for an exact match of a single specified version.
	if wantVersion != nil {
		_, ok := versions[*wantVersion]
		if ok {
			return *wantVersion, nil
		}
	}

	// Build a slice of constraints from the wanted version range.
	var constraints version.Constraints
	if wantVersion != nil {
		var err error
		constraints, err = version.NewConstraint(*wantVersion)
		if err != nil {
			return "", fmt.Errorf("failed to parse wanted version range string: %s", err)
		}
	}

	// Next, find the latest version that matches a specified range.
	var latestSoFar *version.Version
	for verString := range versions {

		v, err := version.NewVersion(verString)
		if err != nil {
			return "", fmt.Errorf("failed to parse version string: %s", err)
		}

		// A pre-release is always disqualified--unless the earlier first check found an exact match.
		if v.Prerelease() != "" {
			continue
		}

		// If there is a wanted version range, disqualify a non-match.
		if wantVersion != nil {
			if !constraints.Check(v) {
				continue
			}
		}

		if latestSoFar == nil {
			// The first one checked is always the greatest so far.
			latestSoFar = v
		} else {
			// Must compare.
			if v.GreaterThan(latestSoFar) {
				latestSoFar = v
			}
		}

	}

	if latestSoFar == nil {
		if wantVersion == nil {
			return "", fmt.Errorf("no available version found")
		}
		return "", fmt.Errorf("no matching version found")
	}

	return latestSoFar.String(), nil
}
