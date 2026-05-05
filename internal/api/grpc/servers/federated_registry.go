package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// FederatedRegistryServer embeds the UnimplementedFederatedRegistriesServer.
type FederatedRegistryServer struct {
	pb.UnimplementedFederatedRegistriesServer
	serviceCatalog *services.Catalog
}

// NewFederatedRegistryServer returns an instance of FederatedRegistryServer.
func NewFederatedRegistryServer(serviceCatalog *services.Catalog) *FederatedRegistryServer {
	return &FederatedRegistryServer{
		serviceCatalog: serviceCatalog,
	}
}

// CreateFederatedRegistryTokens creates federated registry tokens for a job.
func (s *FederatedRegistryServer) CreateFederatedRegistryTokens(ctx context.Context, req *pb.CreateFederatedRegistryTokensRequest) (*pb.CreateFederatedRegistryTokensResponse, error) {
	jobID, err := s.serviceCatalog.FetchModelID(ctx, req.JobId)
	if err != nil {
		return nil, err
	}

	tokens, err := s.serviceCatalog.FederatedRegistryService.CreateFederatedRegistryTokensForJob(ctx, jobID)
	if err != nil {
		return nil, err
	}

	pbTokens := make([]*pb.FederatedRegistryToken, len(tokens))
	for i, t := range tokens {
		pbTokens[i] = &pb.FederatedRegistryToken{
			Token:    t.Token,
			Hostname: t.Hostname,
		}
	}

	return &pb.CreateFederatedRegistryTokensResponse{Tokens: pbTokens}, nil
}
