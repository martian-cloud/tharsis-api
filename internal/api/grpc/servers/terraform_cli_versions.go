package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/cli"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// TerraformCLIVersionsServer embeds the UnimplementedTerraformCLIVersionsServer.
type TerraformCLIVersionsServer struct {
	pb.UnimplementedTerraformCLIVersionsServer
	serviceCatalog *services.Catalog
}

// NewTerraformCLIVersionsServer returns an instance of TerraformCLIVersionsServer.
func NewTerraformCLIVersionsServer(serviceCatalog *services.Catalog) *TerraformCLIVersionsServer {
	return &TerraformCLIVersionsServer{
		serviceCatalog: serviceCatalog,
	}
}

// CreateTerraformCLIDownloadURL creates a download URL for a Terraform CLI binary.
func (s *TerraformCLIVersionsServer) CreateTerraformCLIDownloadURL(ctx context.Context, req *pb.CreateTerraformCLIDownloadURLRequest) (*pb.CreateTerraformCLIDownloadURLResponse, error) {
	downloadURL, err := s.serviceCatalog.CLIService.CreateTerraformCLIDownloadURL(ctx, &cli.TerraformCLIVersionsInput{
		Version:      req.Version,
		OS:           req.Os,
		Architecture: req.Architecture,
	})
	if err != nil {
		return nil, err
	}

	return &pb.CreateTerraformCLIDownloadURLResponse{Url: downloadURL}, nil
}
