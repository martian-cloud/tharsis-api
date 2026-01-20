// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/role"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// RoleServer embeds the UnimplementedRolesServer.
type RoleServer struct {
	pb.UnimplementedRolesServer
	serviceCatalog *services.Catalog
}

// NewRoleServer returns an instance of RoleServer.
func NewRoleServer(serviceCatalog *services.Catalog) *RoleServer {
	return &RoleServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetRoleByID returns a Role by an ID.
func (s *RoleServer) GetRoleByID(ctx context.Context, req *pb.GetRoleByIDRequest) (*pb.Role, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotRole, ok := model.(*models.Role)
	if !ok {
		return nil, errors.New("role with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBRole(gotRole), nil
}

// GetRoles returns a paginated list of Roles.
func (s *RoleServer) GetRoles(ctx context.Context, req *pb.GetRolesRequest) (*pb.GetRolesResponse, error) {
	sort := db.RoleSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &role.GetRolesInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		Search:            req.Search,
	}

	result, err := s.serviceCatalog.RoleService.GetRoles(ctx, input)
	if err != nil {
		return nil, err
	}

	roles := result.Roles

	pbRoles := make([]*pb.Role, len(roles))
	for ix := range roles {
		pbRoles[ix] = toPBRole(&roles[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(roles) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&roles[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&roles[len(roles)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetRolesResponse{
		Roles:    pbRoles,
		PageInfo: pageInfo,
	}, nil
}

// GetAvailablePermissions returns the list of available permissions.
func (s *RoleServer) GetAvailablePermissions(ctx context.Context, _ *emptypb.Empty) (*pb.GetAvailablePermissionsResponse, error) {
	permissions, err := s.serviceCatalog.RoleService.GetAvailablePermissions(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetAvailablePermissionsResponse{
		Permissions: permissions,
	}, nil
}

// CreateRole creates a new Role.
func (s *RoleServer) CreateRole(ctx context.Context, req *pb.CreateRoleRequest) (*pb.Role, error) {
	permissions, err := models.ParsePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}

	input := &role.CreateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		Permissions: permissions,
	}

	createdRole, err := s.serviceCatalog.RoleService.CreateRole(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBRole(createdRole), nil
}

// UpdateRole returns the updated Role.
func (s *RoleServer) UpdateRole(ctx context.Context, req *pb.UpdateRoleRequest) (*pb.Role, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotRole, ok := model.(*models.Role)
	if !ok {
		return nil, errors.New("role with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotRole.Metadata.Version = int(*req.Version)
	}

	if req.Description != nil {
		gotRole.Description = *req.Description
	}

	if len(req.Permissions) > 0 {
		permissions, pErr := models.ParsePermissions(req.Permissions)
		if pErr != nil {
			return nil, pErr
		}
		gotRole.SetPermissions(permissions)
	}

	updatedRole, err := s.serviceCatalog.RoleService.UpdateRole(ctx, &role.UpdateRoleInput{
		Role: gotRole,
	})
	if err != nil {
		return nil, err
	}

	return toPBRole(updatedRole), nil
}

// DeleteRole deletes a Role.
func (s *RoleServer) DeleteRole(ctx context.Context, req *pb.DeleteRoleRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotRole, ok := model.(*models.Role)
	if !ok {
		return nil, errors.New("role with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotRole.Metadata.Version = int(*req.Version)
	}

	if err := s.serviceCatalog.RoleService.DeleteRole(ctx, &role.DeleteRoleInput{
		Role: gotRole,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBRole converts from Role model to ProtoBuf model.
func toPBRole(r *models.Role) *pb.Role {
	permissions := r.GetPermissions()
	pbPermissions := make([]string, len(permissions))
	for i, p := range permissions {
		pbPermissions[i] = p.String()
	}

	return &pb.Role{
		Metadata:    toPBMetadata(&r.Metadata, types.RoleModelType),
		Name:        r.Name,
		Description: r.Description,
		CreatedBy:   r.CreatedBy,
		Permissions: pbPermissions,
	}
}
