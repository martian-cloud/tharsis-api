package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/packageregistry"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// PackageServer embeds the UnimplementedPackagesServer.
type PackageServer struct {
	pb.UnimplementedPackagesServer
	serviceCatalog *services.Catalog
}

// NewPackageServer returns an instance of PackageServer.
func NewPackageServer(serviceCatalog *services.Catalog) *PackageServer {
	return &PackageServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetPackageVersion resolves a package source and version constraint to the uploaded package version
// satisfying it. An empty constraint resolves to the latest uploaded version; an exact version is
// itself a valid constraint and resolves to itself.
func (s *PackageServer) GetPackageVersion(ctx context.Context, req *pb.GetPackageVersionRequest) (*pb.PackageVersion, error) {
	packageVersion, err := s.serviceCatalog.PackageService.ResolvePackageVersion(ctx, &packageregistry.ResolvePackageVersionInput{
		PackageSource:     req.Source,
		VersionConstraint: req.VersionConstraint,
	})
	if err != nil {
		return nil, err
	}

	return toPBPackageVersion(packageVersion), nil
}
