// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/namespacemembership"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// NamespaceMembershipServer embeds the UnimplementedNamespaceMembershipsServer.
type NamespaceMembershipServer struct {
	pb.UnimplementedNamespaceMembershipsServer
	serviceCatalog *services.Catalog
}

// NewNamespaceMembershipServer returns an instance of NamespaceMembershipServer.
func NewNamespaceMembershipServer(serviceCatalog *services.Catalog) *NamespaceMembershipServer {
	return &NamespaceMembershipServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetNamespaceMembershipByID returns a NamespaceMembership by an ID.
func (s *NamespaceMembershipServer) GetNamespaceMembershipByID(ctx context.Context, req *pb.GetNamespaceMembershipByIDRequest) (*pb.NamespaceMembership, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	membership, ok := model.(*models.NamespaceMembership)
	if !ok {
		return nil, errors.New("namespace membership with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBNamespaceMembership(membership), nil
}

// GetNamespaceMembershipsForNamespace returns memberships for a namespace.
func (s *NamespaceMembershipServer) GetNamespaceMembershipsForNamespace(ctx context.Context, req *pb.GetNamespaceMembershipsForNamespaceRequest) (*pb.GetNamespaceMembershipsForNamespaceResponse, error) {
	memberships, err := s.serviceCatalog.NamespaceMembershipService.GetNamespaceMembershipsForNamespace(ctx, req.NamespacePath)
	if err != nil {
		return nil, err
	}

	pbMemberships := make([]*pb.NamespaceMembership, len(memberships))
	for ix := range memberships {
		pbMemberships[ix] = toPBNamespaceMembership(&memberships[ix])
	}

	return &pb.GetNamespaceMembershipsForNamespaceResponse{
		NamespaceMemberships: pbMemberships,
	}, nil
}

// GetNamespaceMembershipsForSubject returns a paginated list of memberships for a subject.
func (s *NamespaceMembershipServer) GetNamespaceMembershipsForSubject(ctx context.Context, req *pb.GetNamespaceMembershipsForSubjectRequest) (*pb.GetNamespaceMembershipsForSubjectResponse, error) {
	sort := db.NamespaceMembershipSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &namespacemembership.GetNamespaceMembershipsForSubjectInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
	}

	if req.UserId != nil {
		userID, uErr := s.serviceCatalog.FetchModelID(ctx, *req.UserId)
		if uErr != nil {
			return nil, uErr
		}
		input.UserID = &userID
	}

	if req.ServiceAccountId != nil {
		model, saErr := s.serviceCatalog.FetchModel(ctx, *req.ServiceAccountId)
		if saErr != nil {
			return nil, saErr
		}
		sa, ok := model.(*models.ServiceAccount)
		if !ok {
			return nil, errors.New("service account with id %s not found", *req.ServiceAccountId, errors.WithErrorCode(errors.ENotFound))
		}
		input.ServiceAccount = sa
	}

	result, err := s.serviceCatalog.NamespaceMembershipService.GetNamespaceMembershipsForSubject(ctx, input)
	if err != nil {
		return nil, err
	}

	memberships := result.NamespaceMemberships

	pbMemberships := make([]*pb.NamespaceMembership, len(memberships))
	for ix := range memberships {
		pbMemberships[ix] = toPBNamespaceMembership(&memberships[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(memberships) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&memberships[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&memberships[len(memberships)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetNamespaceMembershipsForSubjectResponse{
		NamespaceMemberships: pbMemberships,
		PageInfo:             pageInfo,
	}, nil
}

// CreateNamespaceMembership creates a new NamespaceMembership.
func (s *NamespaceMembershipServer) CreateNamespaceMembership(ctx context.Context, req *pb.CreateNamespaceMembershipRequest) (*pb.NamespaceMembership, error) {
	roleID, err := s.serviceCatalog.FetchModelID(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}

	input := &namespacemembership.CreateNamespaceMembershipInput{
		NamespacePath: req.NamespacePath,
		RoleID:        roleID,
	}

	if req.UserId != nil {
		model, uErr := s.serviceCatalog.FetchModel(ctx, *req.UserId)
		if uErr != nil {
			return nil, uErr
		}
		user, ok := model.(*models.User)
		if !ok {
			return nil, errors.New("user with id %s not found", *req.UserId, errors.WithErrorCode(errors.ENotFound))
		}
		input.User = user
	}

	if req.ServiceAccountId != nil {
		model, saErr := s.serviceCatalog.FetchModel(ctx, *req.ServiceAccountId)
		if saErr != nil {
			return nil, saErr
		}
		sa, ok := model.(*models.ServiceAccount)
		if !ok {
			return nil, errors.New("service account with id %s not found", *req.ServiceAccountId, errors.WithErrorCode(errors.ENotFound))
		}
		input.ServiceAccount = sa
	}

	if req.TeamId != nil {
		model, tErr := s.serviceCatalog.FetchModel(ctx, *req.TeamId)
		if tErr != nil {
			return nil, tErr
		}
		team, ok := model.(*models.Team)
		if !ok {
			return nil, errors.New("team with id %s not found", *req.TeamId, errors.WithErrorCode(errors.ENotFound))
		}
		input.Team = team
	}

	createdMembership, err := s.serviceCatalog.NamespaceMembershipService.CreateNamespaceMembership(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBNamespaceMembership(createdMembership), nil
}

// UpdateNamespaceMembership returns the updated NamespaceMembership.
func (s *NamespaceMembershipServer) UpdateNamespaceMembership(ctx context.Context, req *pb.UpdateNamespaceMembershipRequest) (*pb.NamespaceMembership, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	membership, ok := model.(*models.NamespaceMembership)
	if !ok {
		return nil, errors.New("namespace membership with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		membership.Metadata.Version = int(*req.Version)
	}

	roleID, err := s.serviceCatalog.FetchModelID(ctx, req.RoleId)
	if err != nil {
		return nil, err
	}
	membership.RoleID = roleID

	updatedMembership, err := s.serviceCatalog.NamespaceMembershipService.UpdateNamespaceMembership(ctx, membership)
	if err != nil {
		return nil, err
	}

	return toPBNamespaceMembership(updatedMembership), nil
}

// DeleteNamespaceMembership deletes a NamespaceMembership.
func (s *NamespaceMembershipServer) DeleteNamespaceMembership(ctx context.Context, req *pb.DeleteNamespaceMembershipRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	membership, ok := model.(*models.NamespaceMembership)
	if !ok {
		return nil, errors.New("namespace membership with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		membership.Metadata.Version = int(*req.Version)
	}

	if err := s.serviceCatalog.NamespaceMembershipService.DeleteNamespaceMembership(ctx, membership); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBNamespaceMembership converts from NamespaceMembership model to ProtoBuf model.
func toPBNamespaceMembership(nm *models.NamespaceMembership) *pb.NamespaceMembership {
	pbMembership := &pb.NamespaceMembership{
		Metadata: toPBMetadata(&nm.Metadata, types.NamespaceMembershipModelType),
		RoleId:   gid.ToGlobalID(types.RoleModelType, nm.RoleID),
		Namespace: &pb.MembershipNamespace{
			Id:   nm.Namespace.ID,
			Path: nm.Namespace.Path,
		},
	}

	if nm.Namespace.GroupID != nil {
		id := gid.ToGlobalID(types.GroupModelType, *nm.Namespace.GroupID)
		pbMembership.Namespace.GroupId = &id
	}

	if nm.Namespace.WorkspaceID != nil {
		id := gid.ToGlobalID(types.WorkspaceModelType, *nm.Namespace.WorkspaceID)
		pbMembership.Namespace.WorkspaceId = &id
	}

	if nm.UserID != nil {
		id := gid.ToGlobalID(types.UserModelType, *nm.UserID)
		pbMembership.UserId = &id
	}

	if nm.ServiceAccountID != nil {
		id := gid.ToGlobalID(types.ServiceAccountModelType, *nm.ServiceAccountID)
		pbMembership.ServiceAccountId = &id
	}

	if nm.TeamID != nil {
		id := gid.ToGlobalID(types.TeamModelType, *nm.TeamID)
		pbMembership.TeamId = &id
	}

	return pbMembership
}
