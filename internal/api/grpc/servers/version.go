package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/version"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VersionServer implements functionality to get the API version.
type VersionServer struct {
	pb.UnimplementedVersionServer
	versionService version.Service
}

// NewVersionServer returns an instance of VersionServer.
func NewVersionServer(versionService version.Service) *VersionServer {
	return &VersionServer{
		versionService: versionService,
	}
}

// GetVersion returns info about the API and component versions.
func (s *VersionServer) GetVersion(ctx context.Context, _ *emptypb.Empty) (*pb.GetVersionResponse, error) {
	versionInfo, err := s.versionService.GetCurrentVersion(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetVersionResponse{
		Version:            versionInfo.Version,
		DbMigrationVersion: versionInfo.DBMigrationVersion,
		DbMigrationDirty:   versionInfo.DBMigrationDirty,
		BuildTimestamp:     timestamppb.New(versionInfo.BuildTimestamp),
	}, nil
}
