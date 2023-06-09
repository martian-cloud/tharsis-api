// Package team package
package team

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// GetTeamsInput is the input for querying a list of teams
type GetTeamsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TeamSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// TeamNamePrefix filters team list by teamName prefix
	TeamNamePrefix *string
}

// GetTeamMembersInput is the input for querying a list of team members
type GetTeamMembersInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TeamMemberSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// UserID filters the team members by user ID
	UserID *string
	// TeamID filters the team members by user ID
	TeamID *string
}

// Service implements all team related functionality
type Service interface {
	GetTeamByID(ctx context.Context, id string) (*models.Team, error)
	GetTeamByName(ctx context.Context, name string) (*models.Team, error)
	GetTeamsByIDs(ctx context.Context, idList []string) ([]models.Team, error)
	GetTeams(ctx context.Context, input *GetTeamsInput) (*db.TeamsResult, error)
	CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
	UpdateTeam(ctx context.Context, team *models.Team) (*models.Team, error)
	DeleteTeam(ctx context.Context, team *models.Team) error
	GetTeamMember(ctx context.Context, username, teamName string) (*models.TeamMember, error)
	GetTeamMembers(ctx context.Context, input *db.GetTeamMembersInput) (*db.TeamMembersResult, error)
	AddUserToTeam(ctx context.Context, input *models.TeamMember) (*models.TeamMember, error)
	UpdateTeamMember(ctx context.Context, input *models.TeamMember) (*models.TeamMember, error)
	RemoveUserFromTeam(ctx context.Context, input *models.TeamMember) error
}

type service struct {
	logger          logger.Logger
	dbClient        *db.Client
	activityService activityevent.Service
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	activityService activityevent.Service,
) Service {
	return &service{
		logger:          logger,
		dbClient:        dbClient,
		activityService: activityService,
	}
}

//////////////////////////////////////////////////////////////////////////////

// Methods for teams.

func (s *service) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByID")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.getTeamByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by ID")
		return nil, err
	}

	return team, nil
}

func (s *service) GetTeamByName(ctx context.Context, name string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByName")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByName(ctx, name)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by name")
		return nil, err
	}

	if team == nil {
		return nil, errors.New(errors.ENotFound, "Team with name %s not found", name)
	}

	return team, nil
}

func (s *service) GetTeamsByIDs(ctx context.Context, idList []string) ([]models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamsByIDs")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	resp, err := s.dbClient.Teams.GetTeams(ctx, &db.GetTeamsInput{Filter: &db.TeamFilter{TeamIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get teams")
		return nil, err
	}
	teams := resp.Teams

	return teams, nil
}

func (s *service) GetTeams(ctx context.Context, input *GetTeamsInput) (*db.TeamsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeams")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	dbInput := db.GetTeamsInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter: &db.TeamFilter{
			TeamNamePrefix: input.TeamNamePrefix,
		},
	}

	return s.dbClient.Teams.GetTeams(ctx, &dbInput)
}

func (s *service) CreateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Team has not yet been created, so it cannot have an ID.
	if err = caller.RequirePermission(ctx, permissions.CreateTeamPermission); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if err = team.Validate(); err != nil {
		tracing.RecordError(span, err, "failed to validate team model")
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer CreateTeam: %v", txErr)
		}
	}()

	createdTeam, err := s.dbClient.Teams.CreateTeam(txContext, team)
	if err != nil {
		tracing.RecordError(span, err, "failed to create team")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionCreate,
			TargetType: models.TargetTeam,
			TargetID:   createdTeam.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a new team.",
		"caller", caller.GetSubject(),
		"teamName", createdTeam.Name,
		"teamID", createdTeam.Metadata.ID,
	)
	return createdTeam, nil
}

func (s *service) UpdateTeam(ctx context.Context, team *models.Team) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Validate model
	if vErr := team.Validate(); vErr != nil {
		tracing.RecordError(span, vErr, "failed to validate team model")
		return nil, vErr
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateTeam: %v", txErr)
		}
	}()

	updatedTeam, err := s.dbClient.Teams.UpdateTeam(txContext, team)
	if err != nil {
		tracing.RecordError(span, err, "failed to update team")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionUpdate,
			TargetType: models.TargetTeam,
			TargetID:   updatedTeam.Metadata.ID,
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Updated a team.",
		"caller", caller.GetSubject(),
		"teamName", team.Name,
		"teamID", team.Metadata.ID,
	)
	return updatedTeam, nil
}

func (s *service) DeleteTeam(ctx context.Context, team *models.Team) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if err = s.dbClient.Teams.DeleteTeam(ctx, team); err != nil {
		tracing.RecordError(span, err, "failed to delete team")
		return err
	}

	s.logger.Infow("Deleted a team.",
		"caller", caller.GetSubject(),
		"teamName", team.Name,
		"teamID", team.Metadata.ID,
	)
	return nil
}

//////////////////////////////////////////////////////////////////////////////

// Methods for team members.

func (s *service) GetTeamMember(ctx context.Context, username, teamName string) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamMember")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	user, err := s.dbClient.Users.GetUserByUsername(ctx, username)
	if err != nil {
		tracing.RecordError(span, err, "failed to get user by username")
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("user with username %s not found", username)
	}

	team, err := s.GetTeamByName(ctx, teamName)
	if err != nil {
		// This catches both access errors and team not found.
		tracing.RecordError(span, err, "failed to get team by name")
		return nil, err
	}

	teamMember, err := s.dbClient.TeamMembers.GetTeamMember(ctx, user.Metadata.ID, team.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team member")
		return nil, err
	}

	if teamMember == nil {
		return nil, errors.New(
			errors.ENotFound,
			"Team member with username %s and team name %s not found", username, teamName,
		)
	}

	return teamMember, nil
}

func (s *service) GetTeamMembers(ctx context.Context, input *db.GetTeamMembersInput) (*db.TeamMembersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamMembers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Do the query.
	results, err := s.dbClient.TeamMembers.GetTeamMembers(ctx, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team members")
		return nil, err
	}

	// No need to filter the results, because all users can view all teams.

	return results, nil
}

func (s *service) AddUserToTeam(ctx context.Context, input *models.TeamMember) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.AddUserToTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.getTeamByID(ctx, input.TeamID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	s.logger.Infow("Created a new team member.",
		"caller", caller.GetSubject(),
		"userID", input.UserID,
		"teamID", input.TeamID,
		"isMaintainer", input.IsMaintainer,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer AddUserToTeam: %v", txErr)
		}
	}()

	addedTeamMember, err := s.dbClient.TeamMembers.AddUserToTeam(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to add user to team")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionAddMember,
			TargetType: models.TargetTeam,
			TargetID:   team.Metadata.ID,
			Payload: &models.ActivityEventAddTeamMemberPayload{
				UserID:     &input.UserID,
				Maintainer: input.IsMaintainer,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return addedTeamMember, nil
}

func (s *service) UpdateTeamMember(ctx context.Context, input *models.TeamMember) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateTeamMember")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.getTeamByID(ctx, input.TeamID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get am by ID")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	s.logger.Infow("Updated a team member.",
		"caller", caller.GetSubject(),
		"userID", input.UserID,
		"teamID", input.TeamID,
		"isMaintainer", input.IsMaintainer,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer UpdateTeamMember: %v", txErr)
		}
	}()

	updatedTeamMember, err := s.dbClient.TeamMembers.UpdateTeamMember(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to update team member")
		return nil, err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionUpdateMember,
			TargetType: models.TargetTeam,
			TargetID:   team.Metadata.ID,
			Payload: &models.ActivityEventUpdateTeamMemberPayload{
				UserID:     &input.UserID,
				Maintainer: input.IsMaintainer,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	return updatedTeamMember, nil
}

func (s *service) RemoveUserFromTeam(ctx context.Context, input *models.TeamMember) error {
	ctx, span := tracer.Start(ctx, "svc.RemoveUserFromTeam")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	team, err := s.getTeamByID(ctx, input.TeamID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by ID")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	s.logger.Infow("Deleted a team member.",
		"caller", caller.GetSubject(),
		"userID", input.UserID,
		"teamID", input.TeamID,
		"isMaintainer", input.IsMaintainer,
	)

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for service layer RemoveUserFromTeam: %v", txErr)
		}
	}()

	err = s.dbClient.TeamMembers.RemoveUserFromTeam(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to remove user from team")
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionRemoveMember,
			TargetType: models.TargetTeam,
			TargetID:   team.Metadata.ID,
			Payload: &models.ActivityEventRemoveTeamMemberPayload{
				UserID: &input.UserID,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	return s.dbClient.Transactions.CommitTx(txContext)
}

func (s *service) getTeamByID(ctx context.Context, id string) (*models.Team, error) {
	team, err := s.dbClient.Teams.GetTeamByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Returned team pointer will never be nil if error is nil.
	if team == nil {
		return nil, errors.New(
			errors.ENotFound,
			"Team with id %s not found", id,
		)
	}

	return team, nil
}

// The End.
