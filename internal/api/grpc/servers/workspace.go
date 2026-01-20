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
	"google.golang.org/protobuf/types/known/emptypb"
)

// WorkspaceServer embeds the UnimplementedWorkspacesServer.
type WorkspaceServer struct {
	pb.UnimplementedWorkspacesServer
	serviceCatalog *services.Catalog
}

// NewWorkspaceServer returns an instance of WorkspaceServer.
func NewWorkspaceServer(serviceCatalog *services.Catalog) *WorkspaceServer {
	return &WorkspaceServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetWorkspaceByID returns a Workspace by an ID.
func (s *WorkspaceServer) GetWorkspaceByID(ctx context.Context, req *pb.GetWorkspaceByIDRequest) (*pb.Workspace, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	workspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBWorkspace(workspace), nil
}

// GetWorkspaces returns a paginated list of Workspaces.
func (s *WorkspaceServer) GetWorkspaces(ctx context.Context, req *pb.GetWorkspacesRequest) (*pb.GetWorkspacesResponse, error) {
	sort := db.WorkspaceSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	labelFilters := make([]db.WorkspaceLabelFilter, 0, len(req.LabelFilters))
	for key, value := range req.LabelFilters {
		labelFilters = append(labelFilters, db.WorkspaceLabelFilter{
			Key:   key,
			Value: value,
		})
	}

	var groupID *string
	if req.GroupId != nil {
		id, err := s.serviceCatalog.FetchModelID(ctx, *req.GroupId)
		if err != nil {
			return nil, err
		}
		groupID = &id
	}

	var assignedManagedIdentityID *string
	if req.AssignedManagedIdentityId != nil {
		id, err := s.serviceCatalog.FetchModelID(ctx, *req.AssignedManagedIdentityId)
		if err != nil {
			return nil, err
		}
		assignedManagedIdentityID = &id
	}

	input := &workspace.GetWorkspacesInput{
		Search:                    req.Search,
		Sort:                      &sort,
		PaginationOptions:         paginationOpts,
		GroupID:                   groupID,
		AssignedManagedIdentityID: assignedManagedIdentityID,
		WorkspacePath:             req.WorkspacePath,
		LabelFilters:              labelFilters,
	}

	result, err := s.serviceCatalog.WorkspaceService.GetWorkspaces(ctx, input)
	if err != nil {
		return nil, err
	}

	workspaces := result.Workspaces

	pbWorkspaces := make([]*pb.Workspace, len(workspaces))
	for ix := range workspaces {
		pbWorkspaces[ix] = toPBWorkspace(&workspaces[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(workspaces) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&workspaces[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&workspaces[len(workspaces)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetWorkspacesResponse{
		PageInfo:   pageInfo,
		Workspaces: pbWorkspaces,
	}, nil
}

// CreateWorkspace creates a new Workspace.
func (s *WorkspaceServer) CreateWorkspace(ctx context.Context, req *pb.CreateWorkspaceRequest) (*pb.Workspace, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	var runnerTags []string
	if req.RunnerTags != nil {
		runnerTags = req.RunnerTags.GetTags()
	}

	toCreate := &models.Workspace{
		Name:               req.Name,
		Description:        req.Description,
		GroupID:            groupID,
		TerraformVersion:   req.TerraformVersion,
		MaxJobDuration:     req.MaxJobDuration,
		PreventDestroyPlan: req.PreventDestroyPlan,
		RunnerTags:         runnerTags,
		Labels:             req.Labels,
	}

	if req.DriftDetectionEnabled != nil {
		if !req.DriftDetectionEnabled.GetInherit() {
			enabled := req.DriftDetectionEnabled.GetEnabled()
			toCreate.EnableDriftDetection = &enabled
		}
	}

	createdWorkspace, err := s.serviceCatalog.WorkspaceService.CreateWorkspace(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBWorkspace(createdWorkspace), nil
}

// UpdateWorkspace returns the updated Workspace.
func (s *WorkspaceServer) UpdateWorkspace(ctx context.Context, req *pb.UpdateWorkspaceRequest) (*pb.Workspace, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotWorkspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotWorkspace.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		gotWorkspace.Description = *req.Description
	}

	if req.TerraformVersion != nil {
		gotWorkspace.TerraformVersion = *req.TerraformVersion
	}

	if req.MaxJobDuration != nil {
		gotWorkspace.MaxJobDuration = req.MaxJobDuration
	}

	if req.PreventDestroyPlan != nil {
		gotWorkspace.PreventDestroyPlan = *req.PreventDestroyPlan
	}

	if req.RunnerTags != nil {
		gotWorkspace.RunnerTags = req.RunnerTags.GetTags()
	}

	if req.DriftDetectionEnabled != nil {
		if !req.DriftDetectionEnabled.GetInherit() {
			enabled := req.DriftDetectionEnabled.GetEnabled()
			gotWorkspace.EnableDriftDetection = &enabled
		}
	}

	if len(req.Labels) > 0 {
		gotWorkspace.Labels = req.Labels
	}

	updatedWorkspace, err := s.serviceCatalog.WorkspaceService.UpdateWorkspace(ctx, gotWorkspace)
	if err != nil {
		return nil, err
	}

	return toPBWorkspace(updatedWorkspace), nil
}

// DeleteWorkspace deletes a Workspace.
func (s *WorkspaceServer) DeleteWorkspace(ctx context.Context, req *pb.DeleteWorkspaceRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotWorkspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotWorkspace.Metadata.Version = int(*req.Version)
	}

	if err := s.serviceCatalog.WorkspaceService.DeleteWorkspace(ctx, gotWorkspace, req.GetForce()); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// LockWorkspace locks a Workspace.
func (s *WorkspaceServer) LockWorkspace(ctx context.Context, req *pb.LockWorkspaceRequest) (*pb.Workspace, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	gotWorkspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
	}

	lockedWorkspace, err := s.serviceCatalog.WorkspaceService.LockWorkspace(ctx, gotWorkspace)
	if err != nil {
		return nil, err
	}

	return toPBWorkspace(lockedWorkspace), nil
}

// UnlockWorkspace unlocks a Workspace.
func (s *WorkspaceServer) UnlockWorkspace(ctx context.Context, req *pb.UnlockWorkspaceRequest) (*pb.Workspace, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	gotWorkspace, ok := model.(*models.Workspace)
	if !ok {
		return nil, errors.New("workspace with id %s not found", req.WorkspaceId, errors.WithErrorCode(errors.ENotFound))
	}

	unlockedWorkspace, err := s.serviceCatalog.WorkspaceService.UnlockWorkspace(ctx, gotWorkspace)
	if err != nil {
		return nil, err
	}

	return toPBWorkspace(unlockedWorkspace), nil
}

// MigrateWorkspace moves a Workspace to a different group.
func (s *WorkspaceServer) MigrateWorkspace(ctx context.Context, req *pb.MigrateWorkspaceRequest) (*pb.Workspace, error) {
	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return nil, err
	}

	newGroupID, err := s.serviceCatalog.FetchModelID(ctx, req.NewGroupId)
	if err != nil {
		return nil, err
	}

	migratedWorkspace, err := s.serviceCatalog.WorkspaceService.MigrateWorkspace(ctx, workspaceID, newGroupID)
	if err != nil {
		return nil, err
	}

	return toPBWorkspace(migratedWorkspace), nil
}

// SubscribeToWorkspaceEvents subscribes to workspace events.
func (s *WorkspaceServer) SubscribeToWorkspaceEvents(req *pb.SubscribeToWorkspaceEventsRequest, stream pb.Workspaces_SubscribeToWorkspaceEventsServer) error {
	ctx := stream.Context()

	workspaceID, err := s.serviceCatalog.FetchModelID(ctx, req.WorkspaceId)
	if err != nil {
		return err
	}

	eventChan, err := s.serviceCatalog.WorkspaceService.SubscribeToWorkspaceEvents(ctx, &workspace.EventSubscriptionOptions{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return err
	}

	for event := range eventChan {
		pbEvent := &pb.WorkspaceEvent{
			Action:    event.Action,
			Workspace: toPBWorkspace(&event.Workspace),
		}

		if err := stream.Send(pbEvent); err != nil {
			return err
		}
	}

	return nil
}

// toPBWorkspace converts from Workspace model to ProtoBuf model.
func toPBWorkspace(w *models.Workspace) *pb.Workspace {
	var currentJobID string
	if w.CurrentJobID != "" {
		currentJobID = gid.ToGlobalID(types.JobModelType, w.CurrentJobID)
	}

	var currentStateVersionID string
	if w.CurrentStateVersionID != "" {
		currentStateVersionID = gid.ToGlobalID(types.StateVersionModelType, w.CurrentStateVersionID)
	}

	return &pb.Workspace{
		Metadata:              toPBMetadata(&w.Metadata, types.WorkspaceModelType),
		Name:                  w.Name,
		Description:           w.Description,
		GroupId:               gid.ToGlobalID(types.GroupModelType, w.GroupID),
		FullPath:              w.FullPath,
		CreatedBy:             w.CreatedBy,
		Locked:                w.Locked,
		DirtyState:            w.DirtyState,
		CurrentJobId:          currentJobID,
		CurrentStateVersionId: currentStateVersionID,
		Labels:                w.Labels,
	}
}
