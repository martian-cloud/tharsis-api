// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/group"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GroupServer embeds the UnimplementedGroupsServer.
type GroupServer struct {
	pb.UnimplementedGroupsServer
	serviceCatalog *services.Catalog
}

// NewGroupServer returns an instance of GroupServer.
func NewGroupServer(serviceCatalog *services.Catalog) *GroupServer {
	return &GroupServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetGroupByID returns a Group by an ID.
func (s *GroupServer) GetGroupByID(ctx context.Context, req *pb.GetGroupByIDRequest) (*pb.Group, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	group, ok := model.(*models.Group)
	if !ok {
		return nil, errors.New("group with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBGroup(group), nil
}

// GetGroups returns a paginated list of Groups.
func (s *GroupServer) GetGroups(ctx context.Context, req *pb.GetGroupsRequest) (*pb.GetGroupsResponse, error) {
	sort := db.GroupSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &group.GetGroupsInput{
		Search:            req.Search,
		Sort:              &sort,
		PaginationOptions: paginationOpts,
	}

	if req.ParentId != nil {
		parentID, pErr := s.serviceCatalog.FetchModelID(ctx, *req.ParentId)
		if pErr != nil {
			return nil, pErr
		}
		input.ParentGroupID = &parentID
	}

	result, err := s.serviceCatalog.GroupService.GetGroups(ctx, input)
	if err != nil {
		return nil, err
	}

	groups := result.Groups

	pbGroups := make([]*pb.Group, len(groups))
	for ix := range groups {
		pbGroups[ix] = toPBGroup(&groups[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(groups) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&groups[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&groups[len(groups)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetGroupsResponse{
		PageInfo: pageInfo,
		Groups:   pbGroups,
	}, nil
}

// CreateGroup creates a new Group.
func (s *GroupServer) CreateGroup(ctx context.Context, req *pb.CreateGroupRequest) (*pb.Group, error) {
	toCreate := &models.Group{
		Name:        req.Name,
		Description: req.Description,
		RunnerTags:  req.RunnerTags,
	}

	if req.DriftDetectionEnabled != nil {
		if !req.DriftDetectionEnabled.GetInherit() {
			enabled := req.DriftDetectionEnabled.GetEnabled()
			toCreate.EnableDriftDetection = &enabled
		}
	}

	if req.ProviderMirrorEnabled != nil {
		if !req.ProviderMirrorEnabled.GetInherit() {
			enabled := req.ProviderMirrorEnabled.GetEnabled()
			toCreate.EnableProviderMirror = &enabled
		}
	}

	if req.ParentId != nil {
		parentID, err := s.serviceCatalog.FetchModelID(ctx, *req.ParentId)
		if err != nil {
			return nil, err
		}

		toCreate.ParentID = parentID
	}

	createdGroup, err := s.serviceCatalog.GroupService.CreateGroup(ctx, toCreate)
	if err != nil {
		return nil, err
	}

	return toPBGroup(createdGroup), nil
}

// UpdateGroup returns the updated Group.
func (s *GroupServer) UpdateGroup(ctx context.Context, req *pb.UpdateGroupRequest) (*pb.Group, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotGroup, ok := model.(*models.Group)
	if !ok {
		return nil, errors.New("group with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotGroup.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		gotGroup.Description = *req.Description
	}

	if len(req.RunnerTags) > 0 {
		gotGroup.RunnerTags = req.RunnerTags
	}

	if req.DriftDetectionEnabled != nil {
		if !req.DriftDetectionEnabled.GetInherit() {
			enabled := req.DriftDetectionEnabled.GetEnabled()
			gotGroup.EnableDriftDetection = &enabled
		}
	}

	if req.ProviderMirrorEnabled != nil {
		if !req.ProviderMirrorEnabled.GetInherit() {
			enabled := req.ProviderMirrorEnabled.GetEnabled()
			gotGroup.EnableProviderMirror = &enabled
		}
	}

	updatedGroup, err := s.serviceCatalog.GroupService.UpdateGroup(ctx, gotGroup)
	if err != nil {
		return nil, err
	}

	return toPBGroup(updatedGroup), nil
}

// DeleteGroup deletes a Group.
func (s *GroupServer) DeleteGroup(ctx context.Context, req *pb.DeleteGroupRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotGroup, ok := model.(*models.Group)
	if !ok {
		return nil, errors.New("group with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotGroup.Metadata.Version = int(*req.Version)
	}

	toDelete := &group.DeleteGroupInput{
		Group: gotGroup,
		Force: req.GetForce(),
	}

	if err := s.serviceCatalog.GroupService.DeleteGroup(ctx, toDelete); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// MigrateGroup migrates a Group to a new parent.
func (s *GroupServer) MigrateGroup(ctx context.Context, req *pb.MigrateGroupRequest) (*pb.Group, error) {
	groupID, err := s.serviceCatalog.FetchModelID(ctx, req.GroupId)
	if err != nil {
		return nil, err
	}

	var newParentID *string
	if req.NewParentId != nil {
		id, pErr := s.serviceCatalog.FetchModelID(ctx, *req.NewParentId)
		if pErr != nil {
			return nil, pErr
		}
		newParentID = &id
	}

	migratedGroup, err := s.serviceCatalog.GroupService.MigrateGroup(ctx, groupID, newParentID)
	if err != nil {
		return nil, err
	}

	return toPBGroup(migratedGroup), nil
}

// toPBGroup converts from Group model to ProtoBuf model.
func toPBGroup(g *models.Group) *pb.Group {
	var parentID string
	if g.ParentID != "" {
		parentID = gid.ToGlobalID(types.GroupModelType, g.ParentID)
	}

	return &pb.Group{
		Metadata:    toPBMetadata(&g.Metadata, types.GroupModelType),
		Name:        g.Name,
		Description: g.Description,
		FullPath:    g.FullPath,
		CreatedBy:   g.CreatedBy,
		ParentId:    parentID,
	}
}
