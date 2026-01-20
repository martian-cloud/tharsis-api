// Package servers implements the gRPC servers.
package servers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/team"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/protobuf/types/known/emptypb"
)

// TeamServer embeds the UnimplementedTeamsServer.
type TeamServer struct {
	pb.UnimplementedTeamsServer
	serviceCatalog *services.Catalog
}

// NewTeamServer returns an instance of TeamServer.
func NewTeamServer(serviceCatalog *services.Catalog) *TeamServer {
	return &TeamServer{
		serviceCatalog: serviceCatalog,
	}
}

// GetTeamByID returns a Team by an ID.
func (s *TeamServer) GetTeamByID(ctx context.Context, req *pb.GetTeamByIDRequest) (*pb.Team, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	team, ok := model.(*models.Team)
	if !ok {
		return nil, errors.New("team with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	return toPBTeam(team), nil
}

// GetTeams returns a paginated list of Teams.
func (s *TeamServer) GetTeams(ctx context.Context, req *pb.GetTeamsRequest) (*pb.GetTeamsResponse, error) {
	sort := db.TeamSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &team.GetTeamsInput{
		Sort:              &sort,
		PaginationOptions: paginationOpts,
		TeamNamePrefix:    req.TeamNamePrefix,
	}

	if req.UserId != nil {
		userID, uErr := s.serviceCatalog.FetchModelID(ctx, *req.UserId)
		if uErr != nil {
			return nil, uErr
		}
		input.UserID = &userID
	}

	result, err := s.serviceCatalog.TeamService.GetTeams(ctx, input)
	if err != nil {
		return nil, err
	}

	teams := result.Teams

	pbTeams := make([]*pb.Team, len(teams))
	for ix := range teams {
		pbTeams[ix] = toPBTeam(&teams[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(teams) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&teams[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&teams[len(teams)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTeamsResponse{
		PageInfo: pageInfo,
		Teams:    pbTeams,
	}, nil
}

// CreateTeam creates a new Team.
func (s *TeamServer) CreateTeam(ctx context.Context, req *pb.CreateTeamRequest) (*pb.Team, error) {
	input := &team.CreateTeamInput{
		Name:        req.Name,
		Description: req.Description,
	}

	createdTeam, err := s.serviceCatalog.TeamService.CreateTeam(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTeam(createdTeam), nil
}

// UpdateTeam returns the updated Team.
func (s *TeamServer) UpdateTeam(ctx context.Context, req *pb.UpdateTeamRequest) (*pb.Team, error) {
	id, err := s.serviceCatalog.FetchModelID(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	input := &team.UpdateTeamInput{
		ID: id,
	}

	if req.Version != nil {
		version := int(*req.Version)
		input.MetadataVersion = &version
	}

	if req.Description != nil {
		input.Description = req.Description
	}

	updatedTeam, err := s.serviceCatalog.TeamService.UpdateTeam(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTeam(updatedTeam), nil
}

// DeleteTeam deletes a Team.
func (s *TeamServer) DeleteTeam(ctx context.Context, req *pb.DeleteTeamRequest) (*emptypb.Empty, error) {
	model, err := s.serviceCatalog.FetchModel(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	gotTeam, ok := model.(*models.Team)
	if !ok {
		return nil, errors.New("team with id %s not found", req.Id, errors.WithErrorCode(errors.ENotFound))
	}

	if req.Version != nil {
		gotTeam.Metadata.Version = int(*req.Version)
	}

	input := &team.DeleteTeamInput{
		Team: gotTeam,
	}

	if err := s.serviceCatalog.TeamService.DeleteTeam(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetTeamMember returns a team member.
func (s *TeamServer) GetTeamMember(ctx context.Context, req *pb.GetTeamMemberRequest) (*pb.TeamMember, error) {
	teamMember, err := s.serviceCatalog.TeamService.GetTeamMember(ctx, req.Username, req.TeamName)
	if err != nil {
		return nil, err
	}

	return toPBTeamMember(teamMember), nil
}

// GetTeamMembers returns a paginated list of team members.
func (s *TeamServer) GetTeamMembers(ctx context.Context, req *pb.GetTeamMembersRequest) (*pb.GetTeamMembersResponse, error) {
	sort := db.TeamMemberSortableField(req.GetSort().String())

	paginationOpts, err := fromPBPaginationOptions(req.GetPaginationOptions())
	if err != nil {
		return nil, err
	}

	input := &team.GetTeamMembersInput{
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

	if req.TeamId != nil {
		teamID, tErr := s.serviceCatalog.FetchModelID(ctx, *req.TeamId)
		if tErr != nil {
			return nil, tErr
		}
		input.TeamID = &teamID
	}

	result, err := s.serviceCatalog.TeamService.GetTeamMembers(ctx, input)
	if err != nil {
		return nil, err
	}

	teamMembers := result.TeamMembers

	pbTeamMembers := make([]*pb.TeamMember, len(teamMembers))
	for ix := range teamMembers {
		pbTeamMembers[ix] = toPBTeamMember(&teamMembers[ix])
	}

	pageInfo := &pb.PageInfo{
		HasNextPage:     result.PageInfo.HasNextPage,
		HasPreviousPage: result.PageInfo.HasPreviousPage,
		TotalCount:      result.PageInfo.TotalCount,
	}

	if len(teamMembers) > 0 {
		pageInfo.StartCursor, err = result.PageInfo.Cursor(&teamMembers[0])
		if err != nil {
			return nil, err
		}

		pageInfo.EndCursor, err = result.PageInfo.Cursor(&teamMembers[len(teamMembers)-1])
		if err != nil {
			return nil, err
		}
	}

	return &pb.GetTeamMembersResponse{
		PageInfo:    pageInfo,
		TeamMembers: pbTeamMembers,
	}, nil
}

// AddUserToTeam adds a user to a team.
func (s *TeamServer) AddUserToTeam(ctx context.Context, req *pb.AddUserToTeamRequest) (*pb.TeamMember, error) {
	input := &team.AddUserToTeamInput{
		Username:     req.Username,
		TeamName:     req.TeamName,
		IsMaintainer: req.IsMaintainer,
	}

	teamMember, err := s.serviceCatalog.TeamService.AddUserToTeam(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTeamMember(teamMember), nil
}

// UpdateTeamMember updates a team member.
func (s *TeamServer) UpdateTeamMember(ctx context.Context, req *pb.UpdateTeamMemberRequest) (*pb.TeamMember, error) {
	input := &team.UpdateTeamMemberInput{
		Username:     req.Username,
		TeamName:     req.TeamName,
		IsMaintainer: req.IsMaintainer,
	}

	if req.Version != nil {
		version := int(*req.Version)
		input.MetadataVersion = &version
	}

	teamMember, err := s.serviceCatalog.TeamService.UpdateTeamMember(ctx, input)
	if err != nil {
		return nil, err
	}

	return toPBTeamMember(teamMember), nil
}

// RemoveUserFromTeam removes a user from a team.
func (s *TeamServer) RemoveUserFromTeam(ctx context.Context, req *pb.RemoveUserFromTeamRequest) (*emptypb.Empty, error) {
	teamMember, err := s.serviceCatalog.TeamService.GetTeamMember(ctx, req.Username, req.TeamName)
	if err != nil {
		return nil, err
	}

	input := &team.RemoveUserFromTeamInput{
		TeamMember: teamMember,
	}

	if err := s.serviceCatalog.TeamService.RemoveUserFromTeam(ctx, input); err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// toPBTeam converts from Team model to ProtoBuf model.
func toPBTeam(t *models.Team) *pb.Team {
	return &pb.Team{
		Metadata:       toPBMetadata(&t.Metadata, types.TeamModelType),
		Name:           t.Name,
		Description:    t.Description,
		ScimExternalId: t.SCIMExternalID,
	}
}

// toPBTeamMember converts from TeamMember model to ProtoBuf model.
func toPBTeamMember(tm *models.TeamMember) *pb.TeamMember {
	return &pb.TeamMember{
		Metadata:     toPBMetadata(&tm.Metadata, types.TeamMemberModelType),
		UserId:       gid.ToGlobalID(types.UserModelType, tm.UserID),
		TeamId:       gid.ToGlobalID(types.TeamModelType, tm.TeamID),
		IsMaintainer: tm.IsMaintainer,
	}
}
