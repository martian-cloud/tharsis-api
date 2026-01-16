// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/user"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// UserServer embeds the UnimplementedUsersServer.
type UserServer struct {
	pb.UnimplementedUsersServer
	serviceCatalog *services.Catalog
}

// NewUserServer returns an instance of UserServer.
func NewUserServer(serviceCatalog *services.Catalog) *UserServer {
	return &UserServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetUserByID returns a User by an ID.
func (s *UserServer) GetUserByID(ctx context.Context, req *pb.GetUserByIDRequest) (*pb.User, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotUser, ok := model.(*models.User)
	if !ok {
		return nil, errors.New("user with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBUser(gotUser), nil
}

// GetUsers returns a paginated list of Users.
func (s *UserServer) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	sort := db.UserSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &user.GetUsersInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		Search:            req.Search,
	}

	result, err := s.serviceCatalog.UserService.GetUsers(ctx, input)
	if err != nil {
		return nil, err
	}

	users := result.Users

	pbUsers := make([]*pb.User, len(users))
	for ix := range users {
		pbUsers[ix] = toPBUser(&users[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(users) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&users[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&users[len(users)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetUsersResponse{
		Users:    pbUsers,
		PageInfo: pageInfo,
	}, nil
}

// CreateUser creates a new User.
func (s *UserServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	input := &user.CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Admin:    req.Admin,
	}

	createdUser, err := s.serviceCatalog.UserService.CreateUser(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBUser(createdUser), nil
}

// DeleteUser deletes a User.
func (s *UserServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	userID, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	if err := s.serviceCatalog.UserService.DeleteUser(ctx, &user.DeleteUserInput{
		UserID: userID,
	}); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBUser converts from User model to ProtoBuf model.
func toPBUser(u *models.User) *pb.User {
	return &pb.User{
		Metadata:       toPBMetadata(&u.Metadata, types.UserModelType),
		Username:       u.Username,
		Email:          u.Email,
		Admin:          u.Admin,
		Active:         u.Active,
		ScimExternalId: u.SCIMExternalID,
	}
}
