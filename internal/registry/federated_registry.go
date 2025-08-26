// Package registry converses with a remote federated registry to download a module version object and attestations.
package registry

//go:generate go tool mockery --name FederatedRegistryClient --inpackage --case underscore
//go:generate go tool mockery --name sdkClient --inpackage --case underscore

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	sdk "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	sdkauth "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	maxAttestationPages = 10
)

type sdkClient interface {
	GetModuleVersion(ctx context.Context, input *types.GetTerraformModuleVersionInput) (*types.TerraformModuleVersion, error)
	GetModuleAttestations(ctx context.Context, input *types.GetTerraformModuleAttestationsInput) (*types.GetTerraformModuleAttestationsOutput, error)
}

type sdkClientImpl struct {
	client *sdk.Client
}

func (c *sdkClientImpl) GetModuleVersion(ctx context.Context, input *types.GetTerraformModuleVersionInput) (*types.TerraformModuleVersion, error) {
	return c.client.TerraformModuleVersion.GetModuleVersion(ctx, input)
}

func (c *sdkClientImpl) GetModuleAttestations(ctx context.Context, input *types.GetTerraformModuleAttestationsInput) (*types.GetTerraformModuleAttestationsOutput, error) {
	return c.client.TerraformModuleAttestation.GetModuleAttestations(ctx, input)
}

// GetModuleVersionInput is the input for getting a module version.
type GetModuleVersionInput struct {
	FederatedRegistry *models.FederatedRegistry
	ModuleNamespace   string
	ModuleName        string
	ModuleSystem      string
	ModuleVersion     string
}

// GetModuleAttestationsInput is the input for getting module attestations.
type GetModuleAttestationsInput struct {
	FederatedRegistry *models.FederatedRegistry
	ModuleVersionID   string
	ModuleDigest      string
}

// FederatedRegistryClient communicates via HTTP with the federated registry.
type FederatedRegistryClient interface {
	GetModuleVersion(ctx context.Context, input *GetModuleVersionInput) (*types.TerraformModuleVersion, error)
	GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) ([]*types.TerraformModuleAttestation, error)
}

type federatedRegistryClient struct {
	httpClient       *http.Client
	identityProvider auth.IdentityProvider
	sdkClientBuilder func(cfg *config.Config) (sdkClient, error)
	endpointResolver func(httpClient *http.Client, host string) (*url.URL, error)
}

// NewFederatedRegistryClient returns a new FederatedRegistryClient registry.
func NewFederatedRegistryClient(identityProvider auth.IdentityProvider) FederatedRegistryClient {
	return &federatedRegistryClient{
		httpClient:       tharsishttp.NewHTTPClient(),
		identityProvider: identityProvider,
		sdkClientBuilder: sdkClientBuilder,
		endpointResolver: module.GetModuleRegistryEndpointForHost,
	}
}

// GetModuleVersion fetches the module version object from the federated registry server.
func (r *federatedRegistryClient) GetModuleVersion(ctx context.Context, input *GetModuleVersionInput) (*types.TerraformModuleVersion, error) {
	client, err := r.initializeTharsisClient(ctx, input.FederatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Tharsis client")
	}

	response, err := client.GetModuleVersion(ctx, &types.GetTerraformModuleVersionInput{
		ModulePath: ptr.String(fmt.Sprintf("%s/%s/%s", input.ModuleNamespace, input.ModuleName, input.ModuleSystem)),
		Version:    &input.ModuleVersion,
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}

// GetAttestations returns the attestations as SDK types.
func (r *federatedRegistryClient) GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) ([]*types.TerraformModuleAttestation, error) {
	client, err := r.initializeTharsisClient(ctx, input.FederatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Tharsis client")
	}

	response := []*types.TerraformModuleAttestation{}

	count := 0

	var cursor *string
	for {
		toSort := types.TerraformModuleAttestationSortableFieldCreatedAtDesc
		toLimit := int32(100)
		page, err := client.GetModuleAttestations(ctx,
			&types.GetTerraformModuleAttestationsInput{
				Filter: &types.TerraformModuleAttestationFilter{
					TerraformModuleVersionID: &input.ModuleVersionID,
					Digest:                   &input.ModuleDigest,
				},
				Sort: &toSort,
				PaginationOptions: &types.PaginationOptions{
					Limit:  &toLimit,
					Cursor: cursor,
				},
			})
		if err != nil {
			return nil, err
		}

		for _, attestation := range page.ModuleAttestations {
			attestation := attestation
			response = append(response, &attestation)
		}

		if !page.PageInfo.HasNextPage {
			break
		}

		cursor = &page.PageInfo.Cursor

		count++
		if count > maxAttestationPages {
			return nil, errors.New("too many pages of attestations")
		}
	}

	return response, nil
}

// initializeClient opens the Tharsis SDK client for business.
func (r *federatedRegistryClient) initializeTharsisClient(ctx context.Context, federatedRegistry *models.FederatedRegistry) (sdkClient, error) {
	token, err := NewFederatedRegistryToken(ctx, &FederatedRegistryTokenInput{
		FederatedRegistry: federatedRegistry,
		IdentityProvider:  r.identityProvider,
	})
	if err != nil {
		return nil, err
	}

	staticTokenProvider, err := sdkauth.NewStaticTokenProvider(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create static token provider for the remote API: %v")
	}

	realAPIBaseURL, err := r.resolveAPIEndpoint(federatedRegistry.Hostname)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get module api endpoint for service discovery host %s", federatedRegistry.Hostname)
	}

	cfg, err := config.Load(config.WithEndpoint(realAPIBaseURL), config.WithTokenProvider(staticTokenProvider), config.WithHTTPClient(r.httpClient))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load the config for the remote SDK: %v")
	}

	return r.sdkClientBuilder(cfg)
}

// resolveAPIEndpoint fetches the service discovery document from the server and returns the module API URL as a string.
func (r *federatedRegistryClient) resolveAPIEndpoint(host string) (string, error) {
	// Visit the 'well-known' URL for the server in question:
	moduleRegistryURL, err := r.endpointResolver(r.httpClient, host)
	if err != nil {
		return "", err
	}

	// Clear path since we only need the base URL
	moduleRegistryURL.Path = ""

	return moduleRegistryURL.String(), nil
}

func sdkClientBuilder(cfg *config.Config) (sdkClient, error) {
	client, err := sdk.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the remote SDK client: %v")
	}

	return &sdkClientImpl{
		client: client,
	}, nil
}

// FederatedRegistryTokenInput is the input for generating a federated registry token.
type FederatedRegistryTokenInput struct {
	FederatedRegistry *models.FederatedRegistry
	IdentityProvider  auth.IdentityProvider
}

// NewFederatedRegistryToken generates a new federated registry token.
func NewFederatedRegistryToken(ctx context.Context, input *FederatedRegistryTokenInput) (string, error) {
	expiration := time.Now().Add(time.Minute)
	token, err := input.IdentityProvider.GenerateToken(ctx, &auth.TokenInput{
		Expiration: &expiration,
		Subject:    input.FederatedRegistry.GetGlobalID(),
		Audience:   input.FederatedRegistry.Audience,
		Claims: map[string]string{
			"type": auth.FederatedRegistryTokenType,
		},
	})
	if err != nil {
		return "", err
	}

	return string(token), nil
}

// GetFederatedRegistriesInput is the input for getting federated registries.
type GetFederatedRegistriesInput struct {
	DBClient  *db.Client
	GroupPath string
	Hostname  *string
}

// GetFederatedRegistries returns the federated registries for the specified group path.
func GetFederatedRegistries(ctx context.Context, input *GetFederatedRegistriesInput) ([]*models.FederatedRegistry, error) {
	// Get the federated registries for the workspace associated with the job.
	federatedRegistries, err := input.DBClient.FederatedRegistries.GetFederatedRegistries(ctx,
		&db.GetFederatedRegistriesInput{
			Filter: &db.FederatedRegistryFilter{
				Hostname:   input.Hostname,
				GroupPaths: utils.ExpandPath(input.GroupPath),
			},
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get federated registries")
	}

	groupIDs := []string{}
	for _, registry := range federatedRegistries.FederatedRegistries {
		groupIDs = append(groupIDs, registry.GroupID)
	}

	if len(groupIDs) > 1 {
		// Get groups
		groups, err := input.DBClient.Groups.GetGroups(ctx, &db.GetGroupsInput{
			Filter: &db.GroupFilter{
				GroupIDs: groupIDs,
			},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get groups")
		}
		if len(groups.Groups) != len(groupIDs) {
			return nil, errors.New(
				"cannot create tokens since some groups have been deleted",
				errors.WithErrorCode(errors.ENotFound),
			)
		}
		namespaceMap := map[string]string{}
		for _, group := range groups.Groups {
			namespaceMap[group.Metadata.ID] = group.FullPath
		}

		hostMap := map[string]*models.FederatedRegistry{}
		for _, registry := range federatedRegistries.FederatedRegistries {
			existing, ok := hostMap[registry.Hostname]
			if !ok {
				hostMap[registry.Hostname] = registry
			} else {
				namespacePath := namespaceMap[registry.GroupID]
				existingNamespacePath := namespaceMap[existing.GroupID]
				if utils.IsDescendantOfPath(namespacePath, existingNamespacePath) {
					hostMap[registry.Hostname] = registry
				}
			}
		}

		response := []*models.FederatedRegistry{}
		for _, federatedRegistry := range hostMap {
			response = append(response, federatedRegistry)
		}
		return response, nil
	}

	return federatedRegistries.FederatedRegistries, nil
}
