// Package scim package
package scim

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/tracing"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// OP is the type of SCIM update operation.
type OP string

// Equal returns true if the two operations are equal.
// Mainly just exists for convenience.
func (o OP) Equal(operation OP) bool {
	return o == operation
}

// Valid update resource operation types.
const (
	replaceOPType OP = "replace"
	addOPType     OP = "add"
	removeOPType  OP = "remove"
)

var (
	// supportedSCIMUserOperations contains the list of supported user update operations.
	supportedSCIMUserOperations = []OP{
		replaceOPType,
		addOPType,
	}

	// supportedSCIMGroupOperations contains the list of supported group (team) update operations.
	supportedSCIMGroupOperations = []OP{
		replaceOPType,
		addOPType,
		removeOPType,
	}
)

// Operation represents a SCIM PATCH operation.
type Operation struct {
	Value interface{} // Value can be multiple types.
	OP    OP
	Path  string
}

// GetSCIMResourceInput is the input for retrieving a SCIM resource.
type GetSCIMResourceInput struct {
	SCIMExternalID string
}

// CreateSCIMUserInput is the input for creating a new SCIM user.
type CreateSCIMUserInput struct {
	Email          string
	SCIMExternalID string
	Active         bool
}

// CreateSCIMGroupInput is the input for creating a new SCIM group.
type CreateSCIMGroupInput struct {
	Name           string
	SCIMExternalID string
}

// UpdateResourceInput is the input for updating a SCIM resource.
type UpdateResourceInput struct {
	ID         string // Metadata ID.
	Operations []Operation
}

// DeleteSCIMResourceInput is the input for deleting a SCIM resource.
type DeleteSCIMResourceInput struct {
	ID string // Metadata ID.
}

// Service encapsulates the logic for interacting with the SCIM service.
type Service interface {
	CreateSCIMToken(ctx context.Context) ([]byte, error)
	GetSCIMUsers(ctx context.Context, input *GetSCIMResourceInput) ([]models.User, error)
	CreateSCIMUser(ctx context.Context, input *CreateSCIMUserInput) (*models.User, error)
	UpdateSCIMUser(ctx context.Context, input *UpdateResourceInput) (*models.User, error)
	DeleteSCIMUser(ctx context.Context, input *DeleteSCIMResourceInput) error
	GetSCIMGroups(ctx context.Context, input *GetSCIMResourceInput) ([]models.Team, error)
	CreateSCIMGroup(ctx context.Context, input *CreateSCIMGroupInput) (*models.Team, error)
	UpdateSCIMGroup(ctx context.Context, input *UpdateResourceInput) (*models.Team, error)
	DeleteSCIMGroup(ctx context.Context, input *DeleteSCIMResourceInput) error
}

type service struct {
	logger   logger.Logger
	dbClient *db.Client
	idp      auth.IdentityProvider
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	idp auth.IdentityProvider,
) Service {
	return &service{
		logger,
		dbClient,
		idp,
	}
}

func (s *service) CreateSCIMToken(ctx context.Context) ([]byte, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateSCIMToken")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	// Verify caller is a user.
	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New(
			"Unsupported caller type, only users are allowed to create SCIM tokens",
			errors.WithErrorCode(errors.EForbidden))
	}

	// Only admins are allows to create SCIM tokens.
	if !userCaller.User.Admin {
		return nil, errors.New(
			"Only system admins can create SCIM tokens",
			errors.WithErrorCode(errors.EForbidden))
	}

	// Transaction is used to avoid invalidating previous token if new one fails creation.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		tracing.RecordError(span, err, "failed to begin DB transaction")
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for CreateSCIMToken: %v", txErr)
		}
	}()

	// Find any previous token, so it can be invalidated (deleted).
	tokens, err := s.dbClient.SCIMTokens.GetTokens(txContext)
	if err != nil {
		tracing.RecordError(span, err, "failed to get tokens to invalidate")
		return nil, err
	}

	// Delete any previous tokens.
	if len(tokens) > 0 {
		for _, token := range tokens {
			tokenCopy := token
			err = s.dbClient.SCIMTokens.DeleteToken(txContext, &tokenCopy)
			if err != nil {
				tracing.RecordError(span, err, "failed to delete any previous tokens")
				return nil, err
			}
		}
	}

	// Generate a token with a UUID claim.
	jwtID := uuid.New().String()
	scimToken, err := s.idp.GenerateToken(txContext, &auth.TokenInput{
		Subject: "scim",
		JwtID:   jwtID,
		Claims: map[string]string{
			"type": auth.SCIMTokenType,
		},
	})
	if err != nil {
		tracing.RecordError(span, err, "failed to generate token")
		return nil, err
	}

	input := &models.SCIMToken{
		Nonce:     jwtID,
		CreatedBy: caller.GetSubject(),
	}

	// Returned models is not needed.
	_, err = s.dbClient.SCIMTokens.CreateToken(txContext, input)
	if err != nil {
		tracing.RecordError(span, err, "failed to create token")
		return nil, err
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		tracing.RecordError(span, err, "failed to commit DB transaction")
		return nil, err
	}

	s.logger.Infow("Created a new SCIM token.",
		"caller", caller.GetSubject(),
	)

	return scimToken, nil
}

func (s *service) GetSCIMUsers(ctx context.Context, input *GetSCIMResourceInput) ([]models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSCIMUsers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic user information.
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	var users []models.User

	// If SCIMExternalID is specified, use that instead.
	if input.SCIMExternalID != "" {
		user, err := s.dbClient.Users.GetUserBySCIMExternalID(ctx, input.SCIMExternalID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get a SCIM user by scimExternalID")
			return nil, errors.Wrap(err, "failed to get a SCIM user by scimExternalID", errors.WithErrorCode(errors.ENotFound))
		}

		// If a user is not found, do not return an error.
		// Per SCIM, return an empty slice.
		if user != nil {
			users = append(users, *user)
		}
	} else {
		// If no filter is specified, get all users that have a SCIMExternalID set.
		input := &db.GetUsersInput{
			Filter: &db.UserFilter{
				SCIMExternalID: true,
			},
		}
		result, err := s.dbClient.Users.GetUsers(ctx, input)
		if err != nil {
			tracing.RecordError(span, err, "failed to get users")
			return nil, err
		}

		users = result.Users
	}

	return users, nil
}

func (s *service) CreateSCIMUser(ctx context.Context, input *CreateSCIMUserInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateSCIMUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, permissions.CreateUserPermission); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	input.Email = strings.ToLower(input.Email)

	// Check if user with the email already exists.
	existingUser, err := s.dbClient.Users.GetUserByEmail(ctx, input.Email)
	if err != nil {
		tracing.RecordError(span, err, "failed to get user by email")
		return nil, err
	}

	var createdUser *models.User

	// User exists, so update it (it was created without SCIM).
	if existingUser != nil {
		existingUser.Active = input.Active
		existingUser.SCIMExternalID = input.SCIMExternalID

		createdUser, err = s.dbClient.Users.UpdateUser(ctx, existingUser)
		if err != nil {
			tracing.RecordError(span, err, "failed to update user")
			return nil, err
		}
	}

	// No such user, so create it.
	if existingUser == nil {
		newUser := &models.User{
			Username:       auth.ParseUsername(input.Email),
			Email:          input.Email,
			SCIMExternalID: input.SCIMExternalID,
			Active:         input.Active,
		}

		createdUser, err = s.dbClient.Users.CreateUser(ctx, newUser)
		if err != nil {
			tracing.RecordError(span, err, "failed to create user")
			return nil, err
		}
	}

	return createdUser, nil
}

func (s *service) UpdateSCIMUser(ctx context.Context, input *UpdateResourceInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateSCIMUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateUserPermission, auth.WithUserID(input.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	updatedUser, err := s.processSCIMUserOperations(ctx, input.Operations, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to process SCIM user operations")
		return nil, err
	}

	updatedUser, err = s.dbClient.Users.UpdateUser(ctx, updatedUser)
	if err != nil {
		tracing.RecordError(span, err, "failed to update user")
		return nil, err
	}

	return updatedUser, nil
}

func (s *service) DeleteSCIMUser(ctx context.Context, input *DeleteSCIMResourceInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteSCIMUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteUserPermission, auth.WithUserID(input.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get user by ID")
		return err
	}

	if user == nil {
		return errors.New(
			"SCIM user with ID %s does not exist", input.ID,
			errors.WithErrorCode(errors.ENotFound))
	}

	return s.dbClient.Users.DeleteUser(ctx, user)
}

func (s *service) GetSCIMGroups(ctx context.Context, input *GetSCIMResourceInput) ([]models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSCIMGroups")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Any authenticated user can view basic scim group information.
	if _, err := auth.AuthorizeCaller(ctx); err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	var teams []models.Team

	// If SCIMExternalID is specified, use that instead.
	if input.SCIMExternalID != "" {
		team, err := s.dbClient.Teams.GetTeamBySCIMExternalID(ctx, input.SCIMExternalID)
		if err != nil {
			tracing.RecordError(span, err, "failed to get a SCIM group by scimExternalID")
			return nil, errors.Wrap(err, "failed to get a SCIM group by scimExternalID", errors.WithErrorCode(errors.ENotFound))
		}

		// If a team is not found, do not return an error.
		// Per SCIM, return an empty slice.
		if team != nil {
			teams = append(teams, *team)
		}
	} else {
		// If no filter is specified, get all teams that have a SCIMExternalID set.
		input := &db.GetTeamsInput{
			Filter: &db.TeamFilter{
				SCIMExternalID: true,
			},
		}
		result, err := s.dbClient.Teams.GetTeams(ctx, input)
		if err != nil {
			tracing.RecordError(span, err, "failed to get teams")
			return nil, err
		}

		teams = result.Teams
	}

	return teams, nil
}

func (s *service) CreateSCIMGroup(ctx context.Context, input *CreateSCIMGroupInput) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateSCIMGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	if err = caller.RequirePermission(ctx, permissions.CreateTeamPermission); err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	// Check if team with same name exists.
	existingTeam, err := s.dbClient.Teams.GetTeamByName(ctx, input.Name)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by name")
		return nil, err
	}

	var createdTeam *models.Team

	// Team exists, so update it.
	if existingTeam != nil {
		existingTeam.SCIMExternalID = input.SCIMExternalID
		createdTeam, err = s.dbClient.Teams.UpdateTeam(ctx, existingTeam)
		if err != nil {
			tracing.RecordError(span, err, "failed to update team")
			return nil, err
		}
	}

	// No such Team, so create it.
	if existingTeam == nil {
		newTeam := &models.Team{
			Name:           input.Name,
			SCIMExternalID: input.SCIMExternalID,
		}

		createdTeam, err = s.dbClient.Teams.CreateTeam(ctx, newTeam)
		if err != nil {
			tracing.RecordError(span, err, "failed to create team")
			return nil, err
		}
	}

	return createdTeam, nil
}

func (s *service) UpdateSCIMGroup(ctx context.Context, input *UpdateResourceInput) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateSCIMGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return nil, err
	}

	err = caller.RequirePermission(ctx, permissions.UpdateTeamPermission, auth.WithTeamID(input.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return nil, err
	}

	updatedTeam, err := s.processSCIMGroupOperations(ctx, input.Operations, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failes to process SCIM group operations")
		return nil, err
	}

	return updatedTeam, nil
}

func (s *service) DeleteSCIMGroup(ctx context.Context, input *DeleteSCIMResourceInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteSCIMGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		tracing.RecordError(span, err, "caller authorization failed")
		return err
	}

	err = caller.RequirePermission(ctx, permissions.DeleteTeamPermission, auth.WithTeamID(input.ID))
	if err != nil {
		tracing.RecordError(span, err, "permission check failed")
		return err
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, input.ID)
	if err != nil {
		tracing.RecordError(span, err, "failed to get team by ID")
		return err
	}

	if team == nil {
		return errors.New(
			"SCIM group with ID %s not found", input.ID,
			errors.WithErrorCode(errors.ENotFound))
	}

	return s.dbClient.Teams.DeleteTeam(ctx, team)
}

// processSCIMUserOperations processes the SCIM PATCH operations,
// and updates the user fields appropriately. Returns an error
// if operation is not supported.
func (s *service) processSCIMUserOperations(ctx context.Context, operations []Operation, userID string) (*models.User, error) {
	user, err := s.dbClient.Users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, errors.New(
			"Failed to get a SCIM user for processing update operations",
			errors.WithErrorCode(errors.EInternal))
	}

	// Process the update operations.
	for _, operation := range operations {
		// Currently, only replacing an existing attribute is supported.
		err := isSCIMUserOperationSupported(operation.OP)
		if err != nil {
			return nil, err
		}

		// Update user fields based on requested attribute name.
		switch operation.Path {
		case "emails[type eq \"work\"].value":
			email, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected value for user emails field: expected string, got %T", operation.Value)
			}
			user.Email = strings.ToLower(email)
		case "externalId":
			externalID, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected value for user externalId field: expected string, got %T", operation.Value)
			}
			user.SCIMExternalID = externalID
		case "active":
			active, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected value for user active field: expected string, got %T", operation.Value)
			}

			user.Active, err = strconv.ParseBool(active)
			if err != nil {
				return nil, fmt.Errorf("failed to parse active boolean value: %w", err)
			}

		// More fields can be added here.

		default:
			return nil, errors.New(
				"Unsupported SCIM user operation path: %s", operation.Path,
				errors.WithErrorCode(errors.EInvalid))
		}
	}

	return user, nil
}

// processGroupOperations processes the SCIM PATCH operations,
// and updates the team fields appropriately. Returns an error
// if operation is not supported.
func (s *service) processSCIMGroupOperations(ctx context.Context, ops []Operation, teamID string) (*models.Team, error) {
	// Get the team, so it can be updated.
	team, err := s.dbClient.Teams.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if team == nil {
		return nil, errors.New(
			"Failed to get SCIM group for processing update operations",
			errors.WithErrorCode(errors.EInternal))
	}

	// Determine if an update is required before beginning the transaction.
	// Without this, if we start a transaction and no update is made then
	// committing the transaction will fail and generate a false EInternal.
	updateRequired, err := s.isSCIMGroupUpdateRequired(ctx, ops, team)
	if err != nil {
		return nil, err
	}

	// Return if no update is required.
	if !updateRequired {
		return team, nil
	}

	// Use a transaction incase one of the db operations fails.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.Errorf("failed to rollback tx for processSCIMGroupOperations: %v", txErr)
		}
	}()

	// Update group fields based on requested attribute name.
	for _, operation := range ops {
		switch operation.Path {
		case "displayName":
			teamName, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for team displayName: expected string, got %T", operation.Value)
			}
			team.Name = teamName
			team, err = s.dbClient.Teams.UpdateTeam(txContext, team)
			if err != nil {
				return nil, err
			}

		case "externalId":
			externalID, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected type for team externalId: expected string, got %T", operation.Value)
			}
			team.SCIMExternalID = externalID
			team, err = s.dbClient.Teams.UpdateTeam(txContext, team)
			if err != nil {
				return nil, err
			}
		case "members":
			op := operation
			err := s.processSCIMGroupMemberOperation(txContext, &op, team)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return team, nil
}

// processSCIMGroupMemberOperation dispatches the appropriate SCIM group
// member function based on the operation value.
func (s *service) processSCIMGroupMemberOperation(ctx context.Context, operation *Operation, team *models.Team) error {
	// A nil operation value with a "remove" as OP
	// means remove ALL team members from the target team.
	if operation.Value == nil && operation.OP.Equal(removeOPType) {
		return s.removeAllSCIMGroupMembers(ctx, team.Metadata.ID)
	}

	return s.addRemoveSCIMGroupMember(ctx, operation, team)
}

// addRemoveSCIMGroupMember process SCIM group (Team) member operations.
// It adds or removes requested SCIM group (Team) members.
func (s *service) addRemoveSCIMGroupMember(ctx context.Context, operation *Operation, team *models.Team) error {
	userID, err := parseSCIMGroupMemberID(operation)
	if err != nil {
		return err
	}

	// Get the user to find their username.
	user, err := s.dbClient.Users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New(
			"scim user with id %s does not exist", userID,
			errors.WithErrorCode(errors.ENotFound))
	}

	// Add a new group (team) member.
	if operation.OP.Equal(addOPType) {
		_, err := s.dbClient.TeamMembers.AddUserToTeam(ctx, &models.TeamMember{
			UserID: user.Metadata.ID,
			TeamID: team.Metadata.ID,
		})
		if err != nil {
			return err
		}
	}

	// Remove a group (team) member.
	if operation.OP.Equal(removeOPType) {
		// Get the team member.
		teamMember, err := s.dbClient.TeamMembers.GetTeamMember(ctx, user.Metadata.ID, team.Metadata.ID)
		if err != nil {
			return err
		}

		if teamMember == nil {
			return errors.New(
				"scim group member %s in SCIM group %s does not exist",
				user.Username,
				team.Name,
				errors.WithErrorCode(errors.ENotFound),
			)
		}

		// Remove team member from team.
		if err = s.dbClient.TeamMembers.RemoveUserFromTeam(ctx, teamMember); err != nil {
			return err
		}
	}

	return nil
}

// removeAllSCIMGroupMembers is a helper function to remove all group (team) members
// from a group (team).
func (s *service) removeAllSCIMGroupMembers(ctx context.Context, teamID string) error {
	input := &db.GetTeamMembersInput{
		Filter: &db.TeamMemberFilter{
			TeamIDs: []string{teamID},
		},
	}
	result, err := s.dbClient.TeamMembers.GetTeamMembers(ctx, input)
	if err != nil {
		return err
	}

	// Remove team members one by one from Team.
	for _, member := range result.TeamMembers {
		memberCopy := member
		if err := s.dbClient.TeamMembers.RemoveUserFromTeam(ctx, &memberCopy); err != nil {
			return err
		}
	}

	return nil
}

// isSCIMGroupUpdateRequired returns true if an SCIM group operation
// contains an update i.e. it contains a SCIM group (team) name change
// or a new SCIM group (team) member etc. It does NOT perform any updates.
func (s *service) isSCIMGroupUpdateRequired(ctx context.Context, operations []Operation, team *models.Team) (bool, error) {
	// Get all team members to see if any require updates.
	input := &db.GetTeamMembersInput{
		Filter: &db.TeamMemberFilter{
			TeamIDs: []string{team.Metadata.ID},
		},
	}

	result, err := s.dbClient.TeamMembers.GetTeamMembers(ctx, input)
	if err != nil {
		return false, err
	}

	for _, operation := range operations {
		// Make sure operation is supported.
		err := isSCIMGroupOperationSupported(operation.OP)
		if err != nil {
			return false, err
		}

		switch operation.Path {
		case "displayName":
			teamName, ok := operation.Value.(string)
			if !ok {
				return false, fmt.Errorf("unexpected type for team displayName: expected string, got %T", operation.Value)
			}

			if team.Name != teamName {
				return true, nil
			}

		case "externalId":
			externalID, ok := operation.Value.(string)
			if !ok {
				return false, fmt.Errorf("unexpected type for team externalId: expected string, got %T", operation.Value)
			}

			if team.SCIMExternalID != externalID {
				return true, nil
			}

		case "members":
			op := operation
			updateRequired, err := isSCIMGroupMemberUpdateRequired(&op, result.TeamMembers)
			if err != nil {
				return false, err
			}

			if updateRequired {
				return true, nil
			}

		default:
			return false, errors.New(
				"Unsupported SCIM group operation path: %s", operation.Path,
				errors.WithErrorCode(errors.EInvalid))
		}
	}

	return false, nil
}

// isSCIMGroupMemberUpdateRequired determines if a SCIM group
// member operation constitutes an update. Only returns true,
// if a new member is being added or an existing one is being
// removed from a SCIM group (team).
func isSCIMGroupMemberUpdateRequired(operation *Operation, members []models.TeamMember) (bool, error) {
	// If a value isn't specified and OP is removeOPType it means remove ALL members.
	// Return true only if there are members in the SCIM group that _can_ be removed.
	if operation.Value == nil && operation.OP.Equal(removeOPType) && len(members) > 0 {
		return true, nil
	}

	userID, err := parseSCIMGroupMemberID(operation)
	if err != nil {
		return false, err
	}

	// Determine if the SCIM group (team) member exists.
	memberExists := false
	for _, member := range members {
		if member.UserID == userID {
			memberExists = true
			break
		}
	}

	// Return true if a new member is being added.
	if !memberExists && operation.OP.Equal(addOPType) {
		return true, nil
	}

	// Return true if an existing member is being removed.
	if memberExists && operation.OP.Equal(removeOPType) {
		return true, nil
	}

	return false, nil
}

// parseSCIMGroupMemberID parses the SCIM group member ID from an operation value.
func parseSCIMGroupMemberID(operation *Operation) (string, error) {
	// Expecting a slice of maps here.
	valueSlice, ok := operation.Value.([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected value type: expected slice, got %T", operation.Value)
	}

	// Get the first element from the slice and make sure its a map.
	firstElement, ok := valueSlice[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected value type: expected map of string, got %T", valueSlice[0])
	}

	// Get the string userID.
	userID, ok := firstElement["value"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected value type: expected string, got %T", firstElement["value"])
	}

	return gid.FromGlobalID(userID), nil
}

// isSCIMUserOperationSupported returns an error if SCIM user
// update operation is not supported.
func isSCIMUserOperationSupported(operation OP) error {
	for _, op := range supportedSCIMUserOperations {
		if op.Equal(operation) {
			return nil
		}
	}

	return errors.New(
		"Unsupported SCIM user operation: %s", operation,
		errors.WithErrorCode(errors.EInvalid))
}

// isSCIMGroupOperationSupported returns an error if SCIM group
// update operation is not supported.
func isSCIMGroupOperationSupported(operation OP) error {
	for _, op := range supportedSCIMGroupOperations {
		if op.Equal(operation) {
			return nil
		}
	}

	return errors.New(
		"Unsupported SCIM group operation: %s", operation,
		errors.WithErrorCode(errors.EInvalid))
}
