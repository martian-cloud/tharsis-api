package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/resourcelimit"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ResourceLimitServer implements the ResourceLimits gRPC service.
type ResourceLimitServer struct {
	pb.UnimplementedResourceLimitsServer
	service resourcelimit.Service
}

// NewResourceLimitServer returns an instance of ResourceLimitServer.
func NewResourceLimitServer(catalog *services.Catalog) *ResourceLimitServer {
	return &ResourceLimitServer{service: catalog.ResourceLimitService}
}

// GetResourceLimits returns a list of ResourceLimits.
func (s *ResourceLimitServer) GetResourceLimits(ctx context.Context, _ *emptypb.Empty) (*pb.GetResourceLimitsResponse, error) {
	result, err := s.service.GetResourceLimits(ctx)
	if err != nil {
		return nil, err
	}

	pbLimits := make([]*pb.ResourceLimit, len(result))
	for ix := range result {
		pbLimits[ix] = toPBResourceLimit(&result[ix])
	}

	return &pb.GetResourceLimitsResponse{ResourceLimits: pbLimits}, nil
}

// UpdateResourceLimit returns the updated ResourceLimit.
func (s *ResourceLimitServer) UpdateResourceLimit(ctx context.Context, req *pb.UpdateResourceLimitRequest) (*pb.ResourceLimit, error) {
	input := &resourcelimit.UpdateResourceLimitInput{
		Name:  req.Name,
		Value: int(req.Value),
	}

	if req.Version != nil {
		v := int(*req.Version)
		input.MetadataVersion = &v
	}

	updated, err := s.service.UpdateResourceLimit(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBResourceLimit(updated), nil
}

func toPBResourceLimit(limit *models.ResourceLimit) *pb.ResourceLimit {
	return &pb.ResourceLimit{
		Metadata: toPBMetadata(&limit.Metadata, types.ResourceLimitModelType),
		Name:     limit.Name,
		Value:    int32(limit.Value),
	}
}
