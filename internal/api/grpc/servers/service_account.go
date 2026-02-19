// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/aws/smithy-go/ptr"
)

// ServiceAccountServer embeds the UnimplementedServiceAccountsServer.
type ServiceAccountServer struct {
	pb.UnimplementedServiceAccountsServer
	serviceCatalog *services.Catalog
}

// NewServiceAccountServer returns an instance of ServiceAccountServer.
func NewServiceAccountServer(serviceCatalog *services.Catalog) *ServiceAccountServer {
	return &ServiceAccountServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetServiceAccountByID returns a ServiceAccount by an ID.
func (s *ServiceAccountServer) GetServiceAccountByID(ctx context.Context, req *pb.GetServiceAccountByIDRequest) (*pb.ServiceAccount, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	serviceAccount, ok := model.(*models.ServiceAccount)
	if !ok {
		return nil, errors.New("service account with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBServiceAccount(serviceAccount), nil
}

// GetServiceAccounts returns a paginated list of ServiceAccounts.
func (s *ServiceAccountServer) GetServiceAccounts(ctx context.Context, req *pb.GetServiceAccountsRequest) (*pb.GetServiceAccountsResponse, error) {
	sort := db.ServiceAccountSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &serviceaccount.GetServiceAccountsInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		NamespacePath:     req.NamespacePath,
		IncludeInherited:  req.IncludeInherited,
	}

	if req.RunnerId != nil {
		runnerID, rErr := s.serviceCatalog.FetchModelID(ctx, *req.RunnerId)
		if rErr != nil {
			return nil, rErr
		}
		input.RunnerID = &runnerID
	}

	result, err := s.serviceCatalog.ServiceAccountService.GetServiceAccounts(ctx, input)
	if err != nil {
		return nil, err
	}

	serviceAccounts := result.ServiceAccounts

	pbServiceAccounts := make([]*pb.ServiceAccount, len(serviceAccounts))
	for ix := range serviceAccounts {
		pbServiceAccounts[ix] = toPBServiceAccount(&serviceAccounts[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(serviceAccounts) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&serviceAccounts[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&serviceAccounts[len(serviceAccounts)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetServiceAccountsResponse{
		PageInfo:        pageInfo,
		ServiceAccounts: pbServiceAccounts,
	}, nil
}

// CreateServiceAccount creates a new ServiceAccount.
func (s *ServiceAccountServer) CreateServiceAccount(ctx context.Context, req *pb.CreateServiceAccountRequest) (*pb.ServiceAccountResponse, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	input := &serviceaccount.CreateServiceAccountInput{
		Name:                    req.Name,
		Description:             req.Description,
		GroupID:                 groupID,
		OIDCTrustPolicies:       fromPBOIDCTrustPolicies(req.OidcTrustPolicies),
		EnableClientCredentials: req.EnableClientCredentials,
	}

	if req.ClientSecretExpiresAt != nil {
		input.ClientSecretExpiresAt = ptr.Time(req.ClientSecretExpiresAt.AsTime())
	}

	response, err := s.serviceCatalog.ServiceAccountService.CreateServiceAccount(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.ServiceAccountResponse{
		ServiceAccount: toPBServiceAccount(response.ServiceAccount),
		ClientSecret:   response.ClientSecret,
	}, nil
}

// UpdateServiceAccount returns the updated ServiceAccount.
func (s *ServiceAccountServer) UpdateServiceAccount(ctx context.Context, req *pb.UpdateServiceAccountRequest) (*pb.ServiceAccountResponse, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	updateInput := &serviceaccount.UpdateServiceAccountInput{
		ID:                      id,
		EnableClientCredentials: req.EnableClientCredentials,
	}

	if req.ClientSecretExpiresAt != nil {
		updateInput.ClientSecretExpiresAt = ptr.Time(req.ClientSecretExpiresAt.AsTime())
	}

	if req.Version != nil {
		v := int(*req.Version)
		updateInput.MetadataVersion = &v
	}

	if req.Description != nil {
		updateInput.Description = req.Description
	}

	if len(req.OidcTrustPolicies) > 0 {
		updateInput.OIDCTrustPolicies = fromPBOIDCTrustPolicies(req.OidcTrustPolicies)
	}

	response, err := s.serviceCatalog.ServiceAccountService.UpdateServiceAccount(ctx, updateInput)
	if err != nil {
		return nil, err
	}

	return &pb.ServiceAccountResponse{
		ServiceAccount: toPBServiceAccount(response.ServiceAccount),
		ClientSecret:   response.ClientSecret,
	}, nil
}

// DeleteServiceAccount deletes a ServiceAccount.
func (s *ServiceAccountServer) DeleteServiceAccount(ctx context.Context, req *pb.DeleteServiceAccountRequest) (*emptypb.Empty, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	deleteInput := &serviceaccount.DeleteServiceAccountInput{ID: id}

	if req.Version != nil {
		v := int(*req.Version)
		deleteInput.MetadataVersion = &v
	}

	if err := s.serviceCatalog.ServiceAccountService.DeleteServiceAccount(ctx, deleteInput); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateOIDCToken creates a token for a ServiceAccount using OIDC token exchange.
func (s *ServiceAccountServer) CreateOIDCToken(ctx context.Context, req *pb.CreateOIDCTokenRequest) (*pb.CreateTokenResponse, error) {
	input := &serviceaccount.CreateOIDCTokenInput{
		ServiceAccountPublicID: req.ServiceAccountId,
		Token:                  []byte(req.Token),
	}

	result, err := s.serviceCatalog.ServiceAccountService.CreateOIDCToken(ctx, input)
	if err != nil {
		return nil, err
	}

	return &pb.CreateTokenResponse{
		Token:     string(result.Token),
		ExpiresIn: result.ExpiresIn,
	}, nil
}

// CreateClientCredentialsToken creates a token using client credentials.
func (s *ServiceAccountServer) CreateClientCredentialsToken(ctx context.Context, req *pb.CreateClientCredentialsTokenRequest) (*pb.CreateTokenResponse, error) {
	result, err := s.serviceCatalog.ServiceAccountService.CreateClientCredentialsToken(ctx, &serviceaccount.CreateClientCredentialsTokenInput{
		ClientID:     req.ClientId,
		ClientSecret: req.ClientSecret,
	})
	if err != nil {
		return nil, err
	}

	return &pb.CreateTokenResponse{
		Token:     string(result.Token),
		ExpiresIn: result.ExpiresIn,
	}, nil
}

// toPBServiceAccount converts from ServiceAccount model to ProtoBuf model.
func toPBServiceAccount(sa *models.ServiceAccount) *pb.ServiceAccount {
	result := &pb.ServiceAccount{
		Metadata:                 toPBMetadata(&sa.Metadata, types.ServiceAccountModelType),
		Name:                     sa.Name,
		Description:              sa.Description,
		GroupId:                  gid.ToGlobalID(types.GroupModelType, sa.GroupID),
		CreatedBy:                sa.CreatedBy,
		OidcTrustPolicies:        toPBOIDCTrustPolicies(sa.OIDCTrustPolicies),
		ClientCredentialsEnabled: sa.ClientCredentialsEnabled(),
	}

	if sa.ClientSecretExpiresAt != nil {
		result.ClientSecretExpiresAt = timestamppb.New(*sa.ClientSecretExpiresAt)
	}

	return result
}

// toPBOIDCTrustPolicies converts from model OIDC trust policies to ProtoBuf.
func toPBOIDCTrustPolicies(policies []models.OIDCTrustPolicy) []*pb.OIDCTrustPolicy {
	pbPolicies := make([]*pb.OIDCTrustPolicy, len(policies))
	for i, policy := range policies {
		boundClaimType := pb.BoundClaimsType(pb.BoundClaimsType_value[string(policy.BoundClaimsType)])

		pbPolicies[i] = &pb.OIDCTrustPolicy{
			Issuer:          policy.Issuer,
			BoundClaimsType: &boundClaimType,
			BoundClaims:     policy.BoundClaims,
		}
	}
	return pbPolicies
}

// fromPBOIDCTrustPolicies converts from ProtoBuf OIDC trust policies to model.
func fromPBOIDCTrustPolicies(pbPolicies []*pb.OIDCTrustPolicy) []models.OIDCTrustPolicy {
	policies := make([]models.OIDCTrustPolicy, len(pbPolicies))
	for i, pbPolicy := range pbPolicies {
		policies[i] = models.OIDCTrustPolicy{
			Issuer:          pbPolicy.Issuer,
			BoundClaimsType: models.BoundClaimsType(strings.ToLower(pbPolicy.GetBoundClaimsType().String())),
			BoundClaims:     pbPolicy.BoundClaims,
		}
	}
	return policies
}
