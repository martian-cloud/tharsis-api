// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// ConfigurationVersionServer embeds the UnimplementedConfigurationVersionsServer.
type ConfigurationVersionServer struct {
	pb.UnimplementedConfigurationVersionsServer
	serviceCatalog *services.Catalog
}

// NewConfigurationVersionServer returns an instance of ConfigurationVersionServer.
func NewConfigurationVersionServer(serviceCatalog *services.Catalog) *ConfigurationVersionServer {
	return &ConfigurationVersionServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetConfigurationVersionByID returns a ConfigurationVersion by an ID.
func (s *ConfigurationVersionServer) GetConfigurationVersionByID(ctx context.Context, req *pb.GetConfigurationVersionByIDRequest) (*pb.ConfigurationVersion, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	configurationVersion, ok := model.(*models.ConfigurationVersion)
	if !ok {
		return nil, errors.New("configuration version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBConfigurationVersion(configurationVersion), nil
}

// CreateConfigurationVersion creates a new ConfigurationVersion.
func (s *ConfigurationVersionServer) CreateConfigurationVersion(ctx context.Context, req *pb.CreateConfigurationVersionRequest) (*pb.ConfigurationVersion, error) {
	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	toCreate := &workspace.CreateConfigurationVersionInput{
		WorkspaceID: workspaceID,
		Speculative: req.Speculative,
	}

	createdConfigurationVersion, err := s.serviceCatalog.WorkspaceService.CreateConfigurationVersion(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBConfigurationVersion(createdConfigurationVersion), nil
}

// toPBConfigurationVersion converts from ConfigurationVersion model to ProtoBuf model.
func toPBConfigurationVersion(c *models.ConfigurationVersion) *pb.ConfigurationVersion {
	resp := &pb.ConfigurationVersion{
		Metadata:    toPBMetadata(&c.Metadata, types.ConfigurationVersionModelType),
		Status:      string(c.Status),
		Speculative: c.Speculative,
		WorkspaceId: gid.ToGlobalID(types.WorkspaceModelType, c.WorkspaceID),
		CreatedBy:   c.CreatedBy,
	}

	if c.VCSEventID != nil {
		vcsEventID := gid.ToGlobalID(types.VCSEventModelType, *c.VCSEventID)
		resp.VcsEventId = &vcsEventID
	}

	return resp
}
