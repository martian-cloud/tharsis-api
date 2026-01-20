// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

// StateVersionServer embeds the UnimplementedStateVersionsServer.
type StateVersionServer struct {
	pb.UnimplementedStateVersionsServer
	serviceCatalog *services.Catalog
}

// NewStateVersionServer returns an instance of StateVersionServer.
func NewStateVersionServer(serviceCatalog *services.Catalog) *StateVersionServer {
	return &StateVersionServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetStateVersionByID returns a StateVersion by an ID.
func (s *StateVersionServer) GetStateVersionByID(ctx context.Context, req *pb.GetStateVersionByIDRequest) (*pb.StateVersion, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	stateVersion, ok := model.(*models.StateVersion)
	if !ok {
		return nil, errors.New("state version with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBStateVersion(stateVersion), nil
}

// GetStateVersions returns a paginated list of StateVersions.
func (s *StateVersionServer) GetStateVersions(ctx context.Context, req *pb.GetStateVersionsRequest) (*pb.GetStateVersionsResponse, error) {
	sort := db.StateVersionSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	model, err := s.serviceCatalog.FetchModel(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	ws, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
	}

	input := &workspace.GetStateVersionsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		Workspace:         ws,
	}

	result, err := s.serviceCatalog.WorkspaceService.GetStateVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	stateVersions := result.StateVersions

	pbStateVersions := make([]*pb.StateVersion, len(stateVersions))
	for ix := range stateVersions {
		pbStateVersions[ix] = toPBStateVersion(&stateVersions[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(stateVersions) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&stateVersions[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&stateVersions[len(stateVersions)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetStateVersionsResponse{
		PageInfo:      pageInfo,
		StateVersions: pbStateVersions,
	}, nil
}

// CreateStateVersion creates a new StateVersion.
func (s *StateVersionServer) CreateStateVersion(ctx context.Context, req *pb.CreateStateVersionRequest) (*pb.StateVersion, error) {
	stateVersion := &models.StateVersion{}

	if req.RunId != nil {
		model, err := s.serviceCatalog.FetchModel(ctx, *req.RunId)
		if err != nil {
			return nil, err
		}
		run, ok := model.(*models.Run)
		if !ok {
			return nil, errors.New("run with id %s not found", *req.RunId, errors.WithErrorCode(errors.ENotFound))
		}
		stateVersion.WorkspaceID = run.WorkspaceID
		stateVersion.RunID = &run.Metadata.ID
	}

	createdStateVersion, err := s.serviceCatalog.WorkspaceService.CreateStateVersion(ctx, stateVersion, req.State)
	if err != nil {
		return nil, err
	}

	return toPBStateVersion(createdStateVersion), nil
}

// toPBStateVersion converts from StateVersion model to ProtoBuf model.
func toPBStateVersion(sv *models.StateVersion) *pb.StateVersion {
	var runID *string
	if sv.RunID != nil {
		id := gid.ToGlobalID(types.RunModelType, *sv.RunID)
		runID = &id
	}

	return &pb.StateVersion{
		Metadata:    toPBMetadata(&sv.Metadata, types.StateVersionModelType),
		WorkspaceId: gid.ToGlobalID(types.WorkspaceModelType, sv.WorkspaceID),
		RunId:       runID,
		CreatedBy:   sv.CreatedBy,
	}
}
