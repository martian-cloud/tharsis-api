package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CallerIdentityServer embeds the UnimplementedCallerIdentityServer.
type CallerIdentityServer struct {
	pb.UnimplementedCallerIdentityServer
	serviceCatalog *services.Catalog
}

// NewCallerIdentityServer returns an instance of CallerIdentityServer.
func NewCallerIdentityServer(serviceCatalog *services.Catalog) *CallerIdentityServer {
	return &CallerIdentityServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetCallerIdentity returns information about the authenticated caller.
func (s *CallerIdentityServer) GetCallerIdentity(ctx context.Context, _ *emptypb.Empty) (*pb.GetCallerIdentityResponse, error) {
	var response *pb.GetCallerIdentityResponse

	if err := auth.HandleCaller(
		ctx,
		func(_ context.Context, c *auth.UserCaller) error {
			response = &pb.GetCallerIdentityResponse{
				Caller: &pb.GetCallerIdentityResponse_User{
					User: toPBUser(c.User),
				},
			}
			return nil
		},
		func(ctx context.Context, c *auth.ServiceAccountCaller) error {
			serviceAccount, err := s.serviceCatalog.ServiceAccountService.GetServiceAccountByID(ctx, c.ServiceAccountID)
			if err != nil {
				return err
			}
			response = &pb.GetCallerIdentityResponse{
				Caller: &pb.GetCallerIdentityResponse_ServiceAccount{
					ServiceAccount: toPBServiceAccount(serviceAccount),
				},
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	return response, nil
}
