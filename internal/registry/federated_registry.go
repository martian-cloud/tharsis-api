// Package registry converses with a remote federated registry to download a module version object and attestations.
package registry

//go:generate go tool mockery --name FederatedRegistryClient --inpackage --case underscore
//go:generate go tool mockery --name remoteClient --inpackage --case underscore

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	tharsishttp "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/http"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/module"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace/utils"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client/token"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

const (
	maxAttestationPages = 10
)

// remoteClient wraps the gRPC calls needed to communicate with a remote Tharsis instance.
type remoteClient interface {
	GetTerraformModuleVersionByID(ctx context.Context, req *pb.GetTerraformModuleVersionByIDRequest) (*pb.TerraformModuleVersion, error)
	GetTerraformModuleAttestations(ctx context.Context, req *pb.GetTerraformModuleAttestationsRequest) (*pb.GetTerraformModuleAttestationsResponse, error)
	Close() error
}

// grpcRemoteClient implements remoteClient using a GRPCClient.
type grpcRemoteClient struct {
	client *client.GRPCClient
}

func (c *grpcRemoteClient) GetTerraformModuleVersionByID(ctx context.Context, req *pb.GetTerraformModuleVersionByIDRequest) (*pb.TerraformModuleVersion, error) {
	return c.client.TerraformModulesClient.GetTerraformModuleVersionByID(ctx, req)
}

func (c *grpcRemoteClient) GetTerraformModuleAttestations(ctx context.Context, req *pb.GetTerraformModuleAttestationsRequest) (*pb.GetTerraformModuleAttestationsResponse, error) {
	return c.client.TerraformModulesClient.GetTerraformModuleAttestations(ctx, req)
}

func (c *grpcRemoteClient) Close() error {
	return c.client.Close()
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

// FederatedRegistryClient communicates via gRPC with the federated registry.
type FederatedRegistryClient interface {
	GetModuleVersion(ctx context.Context, input *GetModuleVersionInput) (*pb.TerraformModuleVersion, error)
	GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) ([]*pb.TerraformModuleAttestation, error)
}

type federatedRegistryClient struct {
	httpClient          *http.Client
	identityProvider    auth.SigningKeyManager
	logger              logger.Logger
	version             string
	remoteClientBuilder func(ctx context.Context, cfg *client.GRPCClientConfig) (remoteClient, error)
	endpointResolver    func(httpClient *http.Client, host string) (*url.URL, error)
}

// NewFederatedRegistryClient returns a new FederatedRegistryClient registry.
func NewFederatedRegistryClient(logger logger.Logger, identityProvider auth.SigningKeyManager, version string) FederatedRegistryClient {
	return &federatedRegistryClient{
		httpClient:          tharsishttp.NewHTTPClient(),
		identityProvider:    identityProvider,
		remoteClientBuilder: remoteClientBuilder,
		endpointResolver:    module.GetModuleRegistryEndpointForHost,
		version:             version,
		logger:              logger,
	}
}

// GetModuleVersion fetches the module version object from the federated registry server.
func (r *federatedRegistryClient) GetModuleVersion(ctx context.Context, input *GetModuleVersionInput) (*pb.TerraformModuleVersion, error) {
	rc, err := r.initializeRemoteClient(ctx, input.FederatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gRPC client")
	}
	defer rc.Close()

	return rc.GetTerraformModuleVersionByID(ctx, &pb.GetTerraformModuleVersionByIDRequest{
		Id: trn.TypeTerraformModuleVersion.Build(input.ModuleNamespace, input.ModuleName, input.ModuleSystem, input.ModuleVersion),
	})
}

// GetModuleAttestations returns the attestations from the federated registry.
func (r *federatedRegistryClient) GetModuleAttestations(ctx context.Context, input *GetModuleAttestationsInput) ([]*pb.TerraformModuleAttestation, error) {
	rc, err := r.initializeRemoteClient(ctx, input.FederatedRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gRPC client")
	}
	defer rc.Close()

	response := []*pb.TerraformModuleAttestation{}

	count := 0
	var cursor *string
	for {
		limit := int32(100)
		sort := pb.TerraformModuleAttestationSortableField_CREATED_AT_DESC
		req := &pb.GetTerraformModuleAttestationsRequest{
			ModuleId: input.ModuleVersionID,
			Digest:   &input.ModuleDigest,
			Sort:     &sort,
			PaginationOptions: &pb.PaginationOptions{
				First: &limit,
				After: cursor,
			},
		}

		page, err := rc.GetTerraformModuleAttestations(ctx, req)
		if err != nil {
			return nil, err
		}

		response = append(response, page.Attestations...)

		if !page.PageInfo.HasNextPage {
			break
		}

		cursor = page.PageInfo.EndCursor

		count++
		if count > maxAttestationPages {
			return nil, errors.New("too many pages of attestations")
		}
	}

	return response, nil
}

// initializeRemoteClient creates a remote client for the federated registry.
func (r *federatedRegistryClient) initializeRemoteClient(ctx context.Context, federatedRegistry *models.FederatedRegistry) (remoteClient, error) {
	federatedToken, err := NewFederatedRegistryToken(ctx, &FederatedRegistryTokenInput{
		FederatedRegistry: federatedRegistry,
		IdentityProvider:  r.identityProvider,
	})
	if err != nil {
		return nil, err
	}

	apiEndpoint, err := r.resolveAPIEndpoint(federatedRegistry.Hostname)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get module api endpoint for service discovery host %s", federatedRegistry.Hostname)
	}

	tokenResolver, err := token.NewStatic(func() (string, error) {
		return federatedToken, nil
	})
	if err != nil {
		return nil, err
	}

	return r.remoteClientBuilder(ctx, &client.GRPCClientConfig{
		HTTPEndpoint:  apiEndpoint,
		TokenResolver: tokenResolver,
		UserAgent:     client.BuildUserAgent("tharsis-federated-registry", r.version),
		Logger:        r.logger.Slog(),
	})
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

func remoteClientBuilder(ctx context.Context, cfg *client.GRPCClientConfig) (remoteClient, error) {
	grpcClient, err := client.NewGRPCClient(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create remote grpc client")
	}

	return &grpcRemoteClient{
		client: grpcClient,
	}, nil
}

// FederatedRegistryTokenInput is the input for generating a federated registry token.
type FederatedRegistryTokenInput struct {
	FederatedRegistry *models.FederatedRegistry
	IdentityProvider  auth.SigningKeyManager
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
