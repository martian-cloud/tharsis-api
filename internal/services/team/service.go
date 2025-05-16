// Package team package
package team

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/activityevent"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"go.opentelemetry.io/otel/trace"
)

// GetTeamsInput is the input for querying a list of teams
type GetTeamsInput struct {
	// Sort specifies the field to sort on and direction
	Sort *db.TeamSortableField
	// PaginationOptions supports cursor based pagination
	PaginationOptions *pagination.Options
	// TeamNamePrefix filters team list by teamName prefix
	TeamNamePrefix *string
	// UserID filters the team list by user ID
	UserID *string
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

// CreateTeamInput is the input for creating a team.
type CreateTeamInput struct {
	Name        string
	Description string
}

// UpdateTeamInput is the input for updating a team.
type UpdateTeamInput struct {
	MetadataVersion *int
	Description     *string
	ID              string
}

// DeleteTeamInput is the input for deleting a team.
type DeleteTeamInput struct {
	Team *models.Team
}

// AddUserToTeamInput is the input for adding a new team member.
type AddUserToTeamInput struct {
	TeamName     string
	Username     string
	IsMaintainer bool
}

// UpdateTeamMemberInput is the input for updating a team member.
type UpdateTeamMemberInput struct {
	MetadataVersion *int
	TeamName        string
	Username        string
	IsMaintainer    bool
}

// RemoveUserFromTeamInput is the input for deleting a team member.
type RemoveUserFromTeamInput struct {
	TeamMember *models.TeamMember
}

// Service implements all team related functionality
type Service interface {
	GetTeamByID(ctx context.Context, id string) (*models.Team, error)
	GetTeamByTRN(ctx context.Context, trn string) (*models.Team, error)
	GetTeamsByIDs(ctx context.Context, idList []string) ([]models.Team, error)
	GetTeams(ctx context.Context, input *GetTeamsInput) (*db.TeamsResult, error)
	CreateTeam(ctx context.Context, input *CreateTeamInput) (*models.Team, error)
	UpdateTeam(ctx context.Context, input *UpdateTeamInput) (*models.Team, error)
	DeleteTeam(ctx context.Context, input *DeleteTeamInput) error
	GetTeamMember(ctx context.Context, username, teamName string) (*models.TeamMember, error)
	GetTeamMembers(ctx context.Context, input *db.GetTeamMembersInput) (*db.TeamMembersResult, error)
	AddUserToTeam(ctx context.Context, input *AddUserToTeamInput) (*models.TeamMember, error)
	UpdateTeamMember(ctx context.Context, input *UpdateTeamMemberInput) (*models.TeamMember, error)
	RemoveUserFromTeam(ctx context.Context, input *RemoveUserFromTeamInput) error
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

// Methods for teams.

func (s *service) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByID")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, id)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by id")
		return nil, err
	}

	// Returned team pointer will never be nil if error is nil.
	if team == nil {
		tracing.RecordError(span, nil, "team not found")
		return nil, errors.New(
			"team with id %s not found", id,
			errors.WithErrorCode(errors.ENotFound))
	}

	return team, nil
}

func (s *service) GetTeamByTRN(ctx context.Context, trn string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByTRN")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByTRN(ctx, trn)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by trn")
		return nil, err
	}

	// Returned team pointer will never be nil if error is nil.
	if team == nil {
		tracing.RecordError(span, nil, "team not found")
		return nil, errors.New(
			"team with trn %s not found", trn,
			errors.WithErrorCode(errors.ENotFound))
	}

	return team, nil
}

func (s *service) GetTeamsByIDs(ctx context.Context, idList []string) ([]models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamsByIDs")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	resp, err := s.dbClient.Teams.GetTeams(ctx, &db.GetTeamsInput{Filter: &db.TeamFilter{TeamIDs: idList}})
	if err != nil {
		tracing.RecordError(span, err, "failed to get teams")
		return nil, err
	}

	return resp.Teams, nil
}

func (s *service) GetTeams(ctx context.Context, input *GetTeamsInput) (*db.TeamsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeams")
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
			UserID:         input.UserID,
		},
	}

	return s.dbClient.Teams.GetTeams(ctx, &dbInput)
}

func (s *service) CreateTeam(ctx context.Context, input *CreateTeamInput) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Team has not yet been created, so it cannot have an ID.
	if err = caller.RequirePermission(ctx, models.CreateTeamPermission); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	toCreate := &models.Team{
		Name:        input.Name,
		Description: input.Description,
	}

	// Validate model
	if err = toCreate.Validate(); err != nil {
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

	createdTeam, err := s.dbClient.Teams.CreateTeam(txContext, toCreate)
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

func (s *service) UpdateTeam(ctx context.Context, input *UpdateTeamInput) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if team == nil {
		return nil, errors.New("team with id %s not found", input.ID, errors.WithErrorCode(errors.ENotFound))
	}

	err = caller.RequirePermission(ctx, models.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Update fields.
	if input.Description != nil {
		team.Description = *input.Description
	}

	if input.MetadataVersion != nil {
		team.Metadata.Version = *input.MetadataVersion
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

func (s *service) DeleteTeam(ctx context.Context, input *DeleteTeamInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteTeamPermission, auth.WithTeamID(input.Team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	if err = s.dbClient.Teams.DeleteTeam(ctx, input.Team); err != nil {
		tracing.RecordError(span, err, "failed to delete team")
		return err
	}

	s.logger.Infow("Deleted a team.",
		"caller", caller.GetSubject(),
		"teamName", input.Team.Name,
		"teamID", input.Team.Metadata.ID,
	)
	return nil
}

// Methods for team members.

func (s *service) GetTeamMember(ctx context.Context, username, teamName string) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamMember")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	user, err := s.getUserByUsername(ctx, span, username)
	if err != nil {
		return nil, err
	}

	team, err := s.getTeamByName(ctx, span, teamName)
	if err != nil {
		return nil, err
	}

	teamMember, err := s.dbClient.TeamMembers.GetTeamMember(ctx, user.Metadata.ID, team.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team member")
		return nil, err
	}

	if teamMember == nil {
		return nil, errors.New(
			"Team member with username %s and team name %s not found", username, teamName,
			errors.WithErrorCode(errors.ENotFound))
	}

	return teamMember, nil
}

func (s *service) GetTeamMembers(ctx context.Context, input *db.GetTeamMembersInput) (*db.TeamMembersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamMembers")
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

func (s *service) AddUserToTeam(ctx context.Context, input *AddUserToTeamInput) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.AddUserToTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.getTeamByName(ctx, span, input.TeamName)
	if err != nil {
		return nil, err
	}

	user, err := s.getUserByUsername(ctx, span, input.Username)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

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

	toAdd := &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: input.IsMaintainer,
	}

	addedTeamMember, err := s.dbClient.TeamMembers.AddUserToTeam(txContext, toAdd)
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
				UserID:     &user.Metadata.ID,
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

	s.logger.Infow("Created a new team member.",
		"caller", caller.GetSubject(),
		"userID", user.Metadata.ID,
		"teamID", team.Metadata.ID,
		"isMaintainer", input.IsMaintainer,
	)

	return addedTeamMember, nil
}

func (s *service) UpdateTeamMember(ctx context.Context, input *UpdateTeamMemberInput) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateTeamMember")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	team, err := s.getTeamByName(ctx, span, input.TeamName)
	if err != nil {
		return nil, err
	}

	user, err := s.getUserByUsername(ctx, span, input.Username)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.UpdateTeamPermission, auth.WithTeamID(team.Metadata.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	teamMember, err := s.dbClient.TeamMembers.GetTeamMember(ctx, user.Metadata.ID, team.Metadata.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team member")
		return nil, err
	}

	if teamMember == nil {
		tracing.RecordError(span, nil, "team member does not exist")
		return nil, errors.New("user %s is not a member of team %s", user.Username, team.Name, errors.WithErrorCode(errors.ENotFound))
	}

	if input.MetadataVersion != nil {
		teamMember.Metadata.Version = *input.MetadataVersion
	}

	teamMember.IsMaintainer = input.IsMaintainer

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

	updatedTeamMember, err := s.dbClient.TeamMembers.UpdateTeamMember(txContext, teamMember)
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
				UserID:     &user.Metadata.ID,
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

	s.logger.Infow("Updated a team member.",
		"caller", caller.GetSubject(),
		"userID", user.Metadata.ID,
		"teamID", team.Metadata.ID,
		"isMaintainer", input.IsMaintainer,
	)

	return updatedTeamMember, nil
}

func (s *service) RemoveUserFromTeam(ctx context.Context, input *RemoveUserFromTeamInput) error {
	ctx, span := tracer.Start(ctx, "svc.RemoveUserFromTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTeamPermission, auth.WithTeamID(input.TeamMember.TeamID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

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

	err = s.dbClient.TeamMembers.RemoveUserFromTeam(txContext, input.TeamMember)
	if err != nil {
		tracing.RecordError(span, err, "failed to remove user from team")
		return err
	}

	if _, err = s.activityService.CreateActivityEvent(txContext,
		&activityevent.CreateActivityEventInput{
			Action:     models.ActionRemoveMember,
			TargetType: models.TargetTeam,
			TargetID:   input.TeamMember.TeamID,
			Payload: &models.ActivityEventRemoveTeamMemberPayload{
				UserID: &input.TeamMember.UserID,
			},
		}); err != nil {
		tracing.RecordError(span, err, "failed to create activity event")
		return err
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit transaction")
		return err
	}

	s.logger.Infow("Deleted a team member.",
		"caller", caller.GetSubject(),
		"userID", input.TeamMember.UserID,
		"teamID", input.TeamMember.TeamID,
		"isMaintainer", input.TeamMember.IsMaintainer,
	)

	return nil
}

func (s *service) getTeamByName(ctx context.Context, span trace.Span, name string) (*models.Team, error) {
	team, err := s.dbClient.Teams.GetTeamByTRN(ctx, types.TeamModelType.BuildTRN(name))
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by TRN")
		return nil, err
	}

	if team == nil {
		tracing.RecordError(span, nil, "team not found")
		return nil, errors.New("team with name %s not found", name, errors.WithErrorCode(errors.ENotFound))
	}

	return team, nil
}

func (s *service) getUserByUsername(ctx context.Context, span trace.Span, username string) (*models.User, error) {
	user, err := s.dbClient.Users.GetUserByTRN(ctx, types.UserModelType.BuildTRN(username))
	if err != nil {
		tracing.RecordError(span, err, "failed to user by TRN")
		return nil, err
	}

	if user == nil {
		tracing.RecordError(span, nil, "user not found")
		return nil, errors.New("user with username %s not found", username, errors.WithErrorCode(errors.ENotFound))
	}

	return user, nil
}
