// Package servers implements the gRPC servers.
package servers

import (
	"context"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/serviceaccount"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
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
func (s *ServiceAccountServer) CreateServiceAccount(ctx context.Context, req *pb.CreateServiceAccountRequest) (*pb.ServiceAccount, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	toCreate := &models.ServiceAccount{
		Name:              req.Name,
		Description:       req.Description,
		GroupID:           groupID,
		OIDCTrustPolicies: fromPBOIDCTrustPolicies(req.OidcTrustPolicies),
	}

	createdServiceAccount, err := s.serviceCatalog.ServiceAccountService.CreateServiceAccount(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBServiceAccount(createdServiceAccount), nil
}

// UpdateServiceAccount returns the updated ServiceAccount.
func (s *ServiceAccountServer) UpdateServiceAccount(ctx context.Context, req *pb.UpdateServiceAccountRequest) (*pb.ServiceAccount, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	serviceAccount, ok := model.(*models.ServiceAccount)
	if !ok {
		return nil, errors.New("service account with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		serviceAccount.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		serviceAccount.Description = *req.Description
	}

	if len(req.OidcTrustPolicies) > 0 {
		serviceAccount.OIDCTrustPolicies = fromPBOIDCTrustPolicies(req.OidcTrustPolicies)
	}

	updatedServiceAccount, err := s.serviceCatalog.ServiceAccountService.UpdateServiceAccount(ctx, serviceAccount)
	if err != nil {
		return nil, err
	}

	return toPBServiceAccount(updatedServiceAccount), nil
}

// DeleteServiceAccount deletes a ServiceAccount.
func (s *ServiceAccountServer) DeleteServiceAccount(ctx context.Context, req *pb.DeleteServiceAccountRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	serviceAccount, ok := model.(*models.ServiceAccount)
	if !ok {
		return nil, errors.New("service account with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		serviceAccount.Metadata.Version = int(*req.Version)
	}

	if err := s.serviceCatalog.ServiceAccountService.DeleteServiceAccount(ctx, serviceAccount); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CreateToken creates a token for a ServiceAccount.
func (s *ServiceAccountServer) CreateToken(ctx context.Context, req *pb.CreateTokenRequest) (*pb.CreateTokenResponse, error) {
	input := &serviceaccount.CreateTokenInput{
		ServiceAccountPublicID: req.ServiceAccountId,
		Token:                  []byte(req.Token),
	}

	result, err := s.serviceCatalog.ServiceAccountService.CreateToken(ctx, input)
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
	return &pb.ServiceAccount{
		Metadata:          toPBMetadata(&sa.Metadata, types.ServiceAccountModelType),
		Name:              sa.Name,
		Description:       sa.Description,
		GroupId:           gid.ToGlobalID(types.GroupModelType, sa.GroupID),
		CreatedBy:         sa.CreatedBy,
		OidcTrustPolicies: toPBOIDCTrustPolicies(sa.OIDCTrustPolicies),
	}
}

// toPBOIDCTrustPolicies converts from model OIDC trust policies to ProtoBuf.
func toPBOIDCTrustPolicies(policies []models.OIDCTrustPolicy) []*pb.OIDCTrustPolicy {
	pbPolicies := make([]*pb.OIDCTrustPolicy, len(policies))
	for i, policy := range policies {
		pbPolicies[i] = &pb.OIDCTrustPolicy{
			Issuer:          policy.Issuer,
			BoundClaimsType: toPBBoundClaimsType(policy.BoundClaimsType),
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

// toPBBoundClaimsType converts from model BoundClaimsType to ProtoBuf.
func toPBBoundClaimsType(claimsType models.BoundClaimsType) *pb.BoundClaimsType {
	val := pb.BoundClaimsType(pb.BoundClaimsType_value[string(claimsType)])
	return &val
}
