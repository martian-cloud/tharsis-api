// Package registry provides common module registry functionality
package registry

//go:generate go tool mockery --name ModuleResolver --inpackage --case underscore
//go:generate go tool mockery --name ModuleRegistrySource --inpackage --case underscore

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/smithy-go/ptr"
	version "github.com/hashicorp/go-version"
	tfaddrs "github.com/hashicorp/terraform-registry-address"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	db "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run/registry/addrs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// TokenGetterFunc is a function that retrieves a token for a given hostname.
type TokenGetterFunc func(ctx context.Context, hostname string) (string, error)

// FederatedRegistryGetterFunc is a function that retrieves a federated registry for a given hostname.
type FederatedRegistryGetterFunc func(ctx context.Context, hostname string) (*models.FederatedRegistry, error)

// ErrRemoteModuleSource is returned when a module source is not a valid registry source.
var ErrRemoteModuleSource = errors.New("remote module source")

// ModuleRegistrySource is an interface for module registry sources.
type ModuleRegistrySource interface {
	Source() string
	Host() string
	Namespace() string
	Name() string
	TargetSystem() string
	IsTharsisModule() bool
	GetAttestations(ctx context.Context, semanticVersion string, moduleDigest string) ([]string, error)
	LocalRegistryModule(ctx context.Context) (*models.TerraformModule, error)
	ResolveDigest(ctx context.Context, version string) ([]byte, error)
	ResolveSemanticVersion(ctx context.Context, wantVersion *string) (string, error)
}

type commonRegistrySource struct {
	source       string
	host         string
	namespace    string
	name         string
	targetSystem string
	registryURL  *url.URL
	httpClient   *http.Client
	dbClient     *db.Client
}

func (m *commonRegistrySource) Source() string {
	return m.source
}

func (m *commonRegistrySource) Host() string {
	return m.host
}

func (m *commonRegistrySource) Namespace() string {
	return m.namespace
}

func (m *commonRegistrySource) Name() string {
	return m.name
}

func (m *commonRegistrySource) TargetSystem() string {
	return m.targetSystem
}

func (m *commonRegistrySource) IsTharsisModule() bool {
	return false
}

func (m *commonRegistrySource) GetAttestations(_ context.Context, _ string, _ string) ([]string, error) {
	return []string{}, nil
}

func (m *commonRegistrySource) LocalRegistryModule(_ context.Context) (*models.TerraformModule, error) {
	return nil, nil
}

func (m *commonRegistrySource) ResolveDigest(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

type localTharsisRegistrySource struct {
	commonRegistrySource
	moduleID string
}

func (m *localTharsisRegistrySource) IsTharsisModule() bool {
	return true
}

func (m *localTharsisRegistrySource) GetAttestations(ctx context.Context, _ string, moduleDigest string) ([]string, error) {
	response := []string{}

	attestations, err := m.dbClient.TerraformModuleAttestations.GetModuleAttestations(ctx, &db.GetModuleAttestationsInput{
		Filter: &db.TerraformModuleAttestationFilter{
			ModuleID: &m.moduleID,
			Digest:   &moduleDigest,
		},
	})
	if err != nil {
		return nil, err
	}

	for _, attestation := range attestations.ModuleAttestations {
		response = append(response, attestation.Data)
	}
	return response, nil
}

func (m *localTharsisRegistrySource) LocalRegistryModule(ctx context.Context) (*models.TerraformModule, error) {
	module, err := m.dbClient.TerraformModules.GetModuleByID(ctx, m.moduleID)
	if err != nil {
		return nil, err
	}
	if module == nil {
		return nil, errors.New("module not found for source %s", m.Source, errors.WithErrorCode(errors.ENotFound))
	}
	return module, nil
}

func (m *localTharsisRegistrySource) ResolveDigest(ctx context.Context, version string) ([]byte, error) {
	versionsResponse, err := m.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
		Filter: &db.TerraformModuleVersionFilter{
			ModuleID:        &m.moduleID,
			SemanticVersion: &version,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(versionsResponse.ModuleVersions) == 0 {
		return nil, errors.New("unable to find the module package for module %s with semantic version %s", m.Source, version)
	}

	return versionsResponse.ModuleVersions[0].SHASum, nil
}

func (m *localTharsisRegistrySource) ResolveSemanticVersion(ctx context.Context, wantVersion *string) (string, error) {
	var versions map[string]bool

	statusFilter := models.TerraformModuleVersionStatusUploaded
	versionsResponse, getModVerErr := m.dbClient.TerraformModuleVersions.GetModuleVersions(ctx, &db.GetModuleVersionsInput{
		Filter: &db.TerraformModuleVersionFilter{
			ModuleID: &m.moduleID,
			Status:   &statusFilter,
		},
	})
	if getModVerErr != nil {
		return "", getModVerErr
	}

	results := map[string]bool{}
	for _, m := range versionsResponse.ModuleVersions {
		results[m.SemanticVersion] = true
	}

	versions = results

	// Get or verify the version.
	chosenVersion, err := getLatestMatchingVersion(versions, wantVersion)
	if err != nil {
		return "", err
	}

	return chosenVersion, nil
}

type federatedTharsisRegistrySource struct {
	commonRegistrySource
	federatedRegistry       *models.FederatedRegistry
	federatedRegistryClient FederatedRegistryClient
	identityProvider        auth.IdentityProvider
}

func (m *federatedTharsisRegistrySource) IsTharsisModule() bool {
	return true
}

func (m *federatedTharsisRegistrySource) GetAttestations(ctx context.Context, semanticVersion string, moduleDigest string) ([]string, error) {
	response := []string{}

	moduleVersion, err := m.federatedRegistryClient.GetModuleVersion(ctx, &GetModuleVersionInput{
		FederatedRegistry: m.federatedRegistry,
		ModuleNamespace:   m.namespace,
		ModuleName:        m.name,
		ModuleSystem:      m.targetSystem,
		ModuleVersion:     semanticVersion,
	})
	if err != nil {
		return nil, err
	}
	attestations, err := m.federatedRegistryClient.GetModuleAttestations(ctx, &GetModuleAttestationsInput{
		FederatedRegistry: m.federatedRegistry,
		ModuleVersionID:   moduleVersion.Metadata.ID,
		ModuleDigest:      moduleDigest,
	})
	if err != nil {
		return nil, err
	}

	for _, attestation := range attestations {
		response = append(response, attestation.Data)
	}

	return response, nil
}

func (m *federatedTharsisRegistrySource) ResolveDigest(ctx context.Context, version string) ([]byte, error) {
	moduleVersion, err := m.federatedRegistryClient.GetModuleVersion(ctx, &GetModuleVersionInput{
		FederatedRegistry: m.federatedRegistry,
		ModuleNamespace:   m.namespace,
		ModuleName:        m.name,
		ModuleSystem:      m.targetSystem,
		ModuleVersion:     version,
	})
	if err != nil {
		return nil, err
	}

	moduleDigest, err := hex.DecodeString(moduleVersion.SHASum)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode federated registry module digest")
	}

	return moduleDigest, nil
}

func (m *federatedTharsisRegistrySource) ResolveSemanticVersion(ctx context.Context, wantVersion *string) (string, error) {
	var versions map[string]bool

	// Build federated registry token
	token, err := NewFederatedRegistryToken(ctx, &FederatedRegistryTokenInput{
		FederatedRegistry: m.federatedRegistry,
		IdentityProvider:  m.identityProvider,
	})
	if err != nil {
		return "", err
	}

	// Visit the URL to get a list of versions:
	results, err := m.getVersionsUsingModuleRegistryProtocol(ctx, token, func(msg string) error {
		return fmt.Errorf(
			"federated registry %s is not configured to allow access to module %s: verify that the federated registry trust policy is configured correctly to allow access: %s",
			m.federatedRegistry.Hostname,
			m.source,
			msg,
		)
	})
	if err != nil {
		return "", err
	}

	versions = results

	// Get or verify the version.
	chosenVersion, err := getLatestMatchingVersion(versions, wantVersion)
	if err != nil {
		return "", err
	}

	return chosenVersion, nil
}

type genericRegistrySource struct {
	commonRegistrySource
	tokenGetter TokenGetterFunc
}

func (m *genericRegistrySource) ResolveSemanticVersion(ctx context.Context, wantVersion *string) (string, error) {
	var versions map[string]bool

	// Get the auth token for the specified host.
	// var token string
	token, err := m.tokenGetter(ctx, m.host)
	if err != nil {
		return "", err
	}

	// Visit the URL to get a list of versions:
	results, err := m.getVersionsUsingModuleRegistryProtocol(ctx, token, func(msg string) error {
		envVar, _ := module.BuildTokenEnvVar(m.registryURL.Host)
		return fmt.Errorf("token in environment variable %s is not authorized to access this module: %s", envVar, msg)
	})
	if err != nil {
		return "", err
	}

	versions = results

	// Get or verify the version.
	chosenVersion, err := getLatestMatchingVersion(versions, wantVersion)
	if err != nil {
		return "", err
	}

	return chosenVersion, nil
}

func (m *commonRegistrySource) getVersionsUsingModuleRegistryProtocol(_ context.Context, token string, handleUnauthorizedFunc func(msg string) error) (map[string]bool, error) {
	namespace := m.namespace
	moduleName := m.name
	targetSystem := m.targetSystem

	// Resolve a relative reference from the base URL to the 'versions' path.
	versionsRefURL, err := url.Parse(strings.Join([]string{namespace, moduleName, targetSystem, "versions"}, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse download reference string to URL: %v", err)
	}
	versionsURLString := m.registryURL.ResolveReference(versionsRefURL).String()

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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body of versions URL: %s", versionsURLString)
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, handleUnauthorizedFunc(string(body))
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get module versions for module source %s: %s: url=%s: status=%s", m.source, string(body), versionsURLString, resp.Status)
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

// ModuleResolver encapsulates the logic to resolve module source version string(s).
type ModuleResolver interface {
	ParseModuleRegistrySource(ctx context.Context, moduleSource string, tokenGetter TokenGetterFunc, federatedRegistryGetter FederatedRegistryGetterFunc) (ModuleRegistrySource, error)
}

type moduleResolver struct {
	dbClient                *db.Client
	httpClient              *http.Client
	federatedRegistryClient FederatedRegistryClient
	logger                  logger.Logger
	tharsisAPIEndpoint      string
	identityProvider        auth.IdentityProvider
	getRegistryEndpoint     func(client *http.Client, host string) (*url.URL, error)
}

// NewModuleResolver returns
func NewModuleResolver(
	dbClient *db.Client,
	httpClient *http.Client,
	federatedRegistryClient FederatedRegistryClient,
	logger logger.Logger,
	tharsiAPIEndpoint string,
	identityProvider auth.IdentityProvider,
) ModuleResolver {
	return newModuleResolver(
		dbClient,
		httpClient,
		federatedRegistryClient,
		logger,
		tharsiAPIEndpoint,
		identityProvider,
		module.GetModuleRegistryEndpointForHost,
	)
}

func newModuleResolver(
	dbClient *db.Client,
	httpClient *http.Client,
	federatedRegistryClient FederatedRegistryClient,
	logger logger.Logger,
	tharsiAPIEndpoint string,
	identityProvider auth.IdentityProvider,
	getRegistryEndpoint func(client *http.Client, host string) (*url.URL, error),
) ModuleResolver {
	return &moduleResolver{
		dbClient:                dbClient,
		httpClient:              httpClient,
		federatedRegistryClient: federatedRegistryClient,
		logger:                  logger,
		tharsisAPIEndpoint:      tharsiAPIEndpoint,
		identityProvider:        identityProvider,
		getRegistryEndpoint:     getRegistryEndpoint,
	}
}

func (m *moduleResolver) ParseModuleRegistrySource(ctx context.Context, moduleSource string, tokenGetter TokenGetterFunc, federatedRegistryGetter FederatedRegistryGetterFunc) (ModuleRegistrySource, error) {
	// Determine if local module.
	if addrs.IsModuleSourceLocal(moduleSource) {
		return nil, fmt.Errorf("local modules are not supported")
	}

	// ParseModuleSource only supports module registry addresses.
	// This will never return an error to the caller, used as a means
	// of fallthrough.
	parsedSource, err := tfaddrs.ParseModuleSource(moduleSource)
	if err != nil {
		if err = addrs.ValidateModuleSourceRemote(moduleSource); err != nil {
			// This is not a valid module source.
			return nil, fmt.Errorf("invalid module source: %w", err)
		}
		// The source string has been validated as already being a non-registry, remote, Go-Getter-type address.
		return nil, ErrRemoteModuleSource
	}

	host := parsedSource.Package.Host.String()

	// Subdir is not supported.
	if parsedSource.Subdir != "" {
		return nil, fmt.Errorf("subdir not supported when reading module from registry")
	}

	// Visit the 'well-known' URL for the server in question:
	moduleRegistryURL, err := m.getRegistryEndpoint(m.httpClient, host)
	if err != nil {
		return nil, err
	}

	source := commonRegistrySource{
		source:       moduleSource,
		host:         host,
		namespace:    parsedSource.Package.Namespace,
		name:         parsedSource.Package.Name,
		targetSystem: parsedSource.Package.TargetSystem,
		registryURL:  moduleRegistryURL,
		httpClient:   m.httpClient,
		dbClient:     m.dbClient,
	}

	apiURL, err := url.Parse(m.tharsisAPIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse API URL %v", err)
	}

	if moduleRegistryURL.Host == apiURL.Host {
		module, getModErr := GetModuleByAddress(ctx, m.dbClient, parsedSource.Package.Namespace, parsedSource.Package.Name, parsedSource.Package.TargetSystem)
		if getModErr != nil {
			return nil, getModErr
		}

		return &localTharsisRegistrySource{
			commonRegistrySource: source,
			moduleID:             module.Metadata.ID,
		}, nil
	}

	// Check if this is a federated registry
	federatedRegistry, err := federatedRegistryGetter(ctx, host)
	if err != nil {
		return nil, err
	}

	if federatedRegistry != nil {
		return &federatedTharsisRegistrySource{
			commonRegistrySource:    source,
			federatedRegistry:       federatedRegistry,
			federatedRegistryClient: m.federatedRegistryClient,
			identityProvider:        m.identityProvider,
		}, nil
	}

	return &genericRegistrySource{
		commonRegistrySource: source,
		tokenGetter:          tokenGetter,
	}, nil
}

// GetModuleByAddress retrieves a module by its namespace, name, and system.
func GetModuleByAddress(ctx context.Context, dbClient *db.Client, namespace string, name string, system string) (*models.TerraformModule, error) {
	rootGroup, err := dbClient.Groups.GetGroupByFullPath(ctx, namespace)
	if err != nil {
		return nil, err
	}

	if rootGroup == nil {
		return nil, errors.New("namespace %s not found", namespace, errors.WithErrorCode(errors.ENotFound))
	}

	moduleResult, err := dbClient.TerraformModules.GetModules(ctx, &db.GetModulesInput{
		PaginationOptions: &pagination.Options{First: ptr.Int32(1)},
		Filter: &db.TerraformModuleFilter{
			RootGroupID: &rootGroup.Metadata.ID,
			Name:        &name,
			System:      &system,
		},
	})
	if err != nil {
		return nil, err
	}

	if len(moduleResult.Modules) == 0 {
		return nil, errors.New("module with name %s and system %s not found in namespace %s", name, system, namespace, errors.WithErrorCode(errors.ENotFound))
	}

	return &moduleResult.Modules[0], nil
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

		if latestSoFar == nil || v.GreaterThan(latestSoFar) {
			latestSoFar = v
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
