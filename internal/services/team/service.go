// Package team package
package team

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
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
	GetTeamMembers(ctx context.Context, input *GetTeamMembersInput) (*db.TeamMembersResult, error)
	AddUserToTeam(ctx context.Context, input *AddUserToTeamInput) (*models.TeamMember, error)
	UpdateTeamMember(ctx context.Context, input *UpdateTeamMemberInput) (*models.TeamMember, error)
	RemoveUserFromTeam(ctx context.Context, input *RemoveUserFromTeamInput) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
) Service {
	return &service{
		logger:   logger,
		dbClient: dbClient,
	}
}

// Methods for teams.

func (s *service) GetTeamByID(ctx context.Context, id string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByID")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team by id", errors.WithSpan(span))
	}

	// Returned team pointer will never be nil if error is nil.
	if team == nil {
		return nil, errors.New(
			"team with id %s not found", id,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return team, nil
}

func (s *service) GetTeamByTRN(ctx context.Context, trn string) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamByTRN")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	team, err := s.dbClient.Teams.GetTeamByTRN(ctx, trn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team by trn", errors.WithSpan(span))
	}

	// Returned team pointer will never be nil if error is nil.
	if team == nil {
		return nil, errors.New(
			"team with trn %s not found", trn,
			errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return team, nil
}

func (s *service) GetTeamsByIDs(ctx context.Context, idList []string) ([]models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamsByIDs")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	resp, err := s.dbClient.Teams.GetTeams(ctx, &db.GetTeamsInput{Filter: &db.TeamFilter{TeamIDs: idList}})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get teams", errors.WithSpan(span))
	}

	return resp.Teams, nil
}

func (s *service) GetTeams(ctx context.Context, input *GetTeamsInput) (*db.TeamsResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeams")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
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
		return nil, err
	}

	// Team has not yet been created, so it cannot have an ID.
	if err = caller.RequirePermission(ctx, models.CreateTeamPermission); err != nil {
		return nil, err
	}

	toCreate := &models.Team{
		Name:        input.Name,
		Description: input.Description,
	}

	// Validate model
	if err = toCreate.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate team model", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer CreateTeam: %v", txErr)
		}
	}()

	createdTeam, err := s.dbClient.Teams.CreateTeam(txContext, toCreate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create team", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			Action:     models.ActionCreate,
			TargetType: models.TargetTeam,
			TargetID:   createdTeam.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new team.",
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
		return nil, errors.Wrap(vErr, "failed to validate team model", errors.WithSpan(span))
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateTeam: %v", txErr)
		}
	}()

	updatedTeam, err := s.dbClient.Teams.UpdateTeam(txContext, team)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update team", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			Action:     models.ActionUpdate,
			TargetType: models.TargetTeam,
			TargetID:   updatedTeam.Metadata.ID,
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a team.",
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
		return err
	}

	err = caller.RequirePermission(ctx, models.DeleteTeamPermission, auth.WithTeamID(input.Team.Metadata.ID))
	if err != nil {
		return err
	}

	if err = s.dbClient.Teams.DeleteTeam(ctx, input.Team); err != nil {
		return errors.Wrap(err, "failed to delete team", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a team.",
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
		return nil, errors.Wrap(err, "failed to get team member", errors.WithSpan(span))
	}

	if teamMember == nil {
		return nil, errors.New(
			"Team member with username %s and team name %s not found", username, teamName,
			errors.WithErrorCode(errors.ENotFound))
	}

	return teamMember, nil
}

func (s *service) GetTeamMembers(ctx context.Context, input *GetTeamMembersInput) (*db.TeamMembersResult, error) {
	ctx, span := tracer.Start(ctx, "svc.GetTeamMembers")
	defer span.End()

	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		return nil, err
	}

	dbInput := &db.GetTeamMembersInput{
		Sort:              input.Sort,
		PaginationOptions: input.PaginationOptions,
		Filter:            &db.TeamMemberFilter{},
	}

	if input.UserID != nil {
		dbInput.Filter.UserID = input.UserID
	}

	if input.TeamID != nil {
		dbInput.Filter.TeamIDs = []string{*input.TeamID}
	}

	// Do the query.
	results, err := s.dbClient.TeamMembers.GetTeamMembers(ctx, dbInput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team members", errors.WithSpan(span))
	}

	// No need to filter the results, because all users can view all teams.

	return results, nil
}

func (s *service) AddUserToTeam(ctx context.Context, input *AddUserToTeamInput) (*models.TeamMember, error) {
	ctx, span := tracer.Start(ctx, "svc.AddUserToTeam")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
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
		return nil, err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer AddUserToTeam: %v", txErr)
		}
	}()

	toAdd := &models.TeamMember{
		UserID:       user.Metadata.ID,
		TeamID:       team.Metadata.ID,
		IsMaintainer: input.IsMaintainer,
	}

	addedTeamMember, err := s.dbClient.TeamMembers.AddUserToTeam(txContext, toAdd)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add user to team", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			Action:     models.ActionAddMember,
			TargetType: models.TargetTeam,
			TargetID:   team.Metadata.ID,
			Payload: &models.ActivityEventAddTeamMemberPayload{
				UserID:     &user.Metadata.ID,
				Maintainer: input.IsMaintainer,
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new team member.",
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
		return nil, err
	}

	teamMember, err := s.dbClient.TeamMembers.GetTeamMember(ctx, user.Metadata.ID, team.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team member", errors.WithSpan(span))
	}

	if teamMember == nil {
		return nil, errors.New("user %s is not a member of team %s", user.Username, team.Name, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	if input.MetadataVersion != nil {
		teamMember.Metadata.Version = *input.MetadataVersion
	}

	teamMember.IsMaintainer = input.IsMaintainer

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer UpdateTeamMember: %v", txErr)
		}
	}()

	updatedTeamMember, err := s.dbClient.TeamMembers.UpdateTeamMember(txContext, teamMember)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update team member", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			Action:     models.ActionUpdateMember,
			TargetType: models.TargetTeam,
			TargetID:   team.Metadata.ID,
			Payload: &models.ActivityEventUpdateTeamMemberPayload{
				UserID:     &user.Metadata.ID,
				Maintainer: input.IsMaintainer,
			},
		}); err != nil {
		return nil, errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a team member.",
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
		return err
	}

	err = caller.RequirePermission(ctx, models.UpdateTeamPermission, auth.WithTeamID(input.TeamMember.TeamID))
	if err != nil {
		return err
	}

	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for service layer RemoveUserFromTeam: %v", txErr)
		}
	}()

	err = s.dbClient.TeamMembers.RemoveUserFromTeam(txContext, input.TeamMember)
	if err != nil {
		return errors.Wrap(err, "failed to remove user from team", errors.WithSpan(span))
	}

	if _, err = activity.CreateActivityEvent(txContext, s.dbClient,
		&activity.CreateActivityEventInput{
			Action:     models.ActionRemoveMember,
			TargetType: models.TargetTeam,
			TargetID:   input.TeamMember.TeamID,
			Payload: &models.ActivityEventRemoveTeamMemberPayload{
				UserID: &input.TeamMember.UserID,
			},
		}); err != nil {
		return errors.Wrap(err, "failed to create activity event", errors.WithSpan(span))
	}

	if err = s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return errors.Wrap(err, "failed to commit transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a team member.",
		"userID", input.TeamMember.UserID,
		"teamID", input.TeamMember.TeamID,
		"isMaintainer", input.TeamMember.IsMaintainer,
	)

	return nil
}

func (s *service) getTeamByName(ctx context.Context, span trace.Span, name string) (*models.Team, error) {
	team, err := s.dbClient.Teams.GetTeamByTRN(ctx, trn.TypeTeam.Build(name))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team by TRN", errors.WithSpan(span))
	}

	if team == nil {
		return nil, errors.New("team with name %s not found", name, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return team, nil
}

func (s *service) getUserByUsername(ctx context.Context, span trace.Span, username string) (*models.User, error) {
	user, err := s.dbClient.Users.GetUserByTRN(ctx, trn.TypeUser.Build(username))
	if err != nil {
		return nil, errors.Wrap(err, "failed to user by TRN", errors.WithSpan(span))
	}

	if user == nil {
		return nil, errors.New("user with username %s not found", username, errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
	}

	return user, nil
}
