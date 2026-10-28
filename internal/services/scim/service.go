// Package scim package
package scim

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
	"go.opentelemetry.io/otel/attribute"
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

// GetSCIMUsersInput is the input for retrieving a SCIM Users.
type GetSCIMUsersInput struct {
	SCIMExternalID *string
	Username       *string
}

// GetSCIMGroupsInput is the input for retrieving a SCIM Groups.
type GetSCIMGroupsInput struct {
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

// CreateSCIMTokenOutput contains the generated secret and its persisted model.
type CreateSCIMTokenOutput struct {
	PlaintextToken []byte
	SCIMToken      *models.SCIMToken
}

// Service encapsulates the logic for interacting with the SCIM service.
type Service interface {
	GetSCIMToken(ctx context.Context) (*models.SCIMToken, error)
	CreateSCIMToken(ctx context.Context, idpIssuerURL string) (*CreateSCIMTokenOutput, error)
	GetSCIMUsers(ctx context.Context, input *GetSCIMUsersInput) ([]models.User, error)
	CreateSCIMUser(ctx context.Context, input *CreateSCIMUserInput) (*models.User, error)
	UpdateSCIMUser(ctx context.Context, input *UpdateResourceInput) (*models.User, error)
	DeleteSCIMUser(ctx context.Context, input *DeleteSCIMResourceInput) error
	GetSCIMGroups(ctx context.Context, input *GetSCIMGroupsInput) ([]models.Team, error)
	CreateSCIMGroup(ctx context.Context, input *CreateSCIMGroupInput) (*models.Team, error)
	UpdateSCIMGroup(ctx context.Context, input *UpdateResourceInput) (*models.Team, error)
	DeleteSCIMGroup(ctx context.Context, input *DeleteSCIMResourceInput) error
}

type service struct {
	logger            logger.Logger
	dbClient          *db.Client
	signingKeyManager auth.SigningKeyManager
	oauthProviders    []config.IdpConfig
}

// NewService creates an instance of Service
func NewService(
	logger logger.Logger,
	dbClient *db.Client,
	signingKeyManager auth.SigningKeyManager,
	oauthProviders []config.IdpConfig,
) Service {
	return &service{
		logger,
		dbClient,
		signingKeyManager,
		oauthProviders,
	}
}

// GetSCIMToken returns the current SCIM token model.
func (s *service) GetSCIMToken(ctx context.Context) (*models.SCIMToken, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSCIMToken")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	if !caller.IsAdminModeActivated(ctx) {
		return nil, errors.New(
			"only admins with admin mode activated can view SCIM token details",
			errors.WithErrorCode(errors.EForbidden))
	}

	tokens, err := s.dbClient.SCIMTokens.GetTokens(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get SCIM tokens", errors.WithSpan(span))
	}

	if len(tokens) == 0 {
		return nil, errors.New(
			"SCIM token not found",
			errors.WithErrorCode(errors.ENotFound),
			errors.WithSpan(span),
		)
	}

	return &tokens[0], nil
}

func (s *service) CreateSCIMToken(ctx context.Context, idpIssuerURL string) (*CreateSCIMTokenOutput, error) {
	ctx, span := tracer.Start(ctx, "svc.CreateSCIMToken")
	span.SetAttributes(attribute.String("idpIssuerURL", idpIssuerURL))
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	// Verify caller is a user.
	userCaller, ok := caller.(*auth.UserCaller)
	if !ok {
		return nil, errors.New(
			"Unsupported caller type, only users are allowed to create SCIM tokens",
			errors.WithErrorCode(errors.EForbidden))
	}

	// Only admins with admin mode activated are allowed to create SCIM tokens.
	if !userCaller.IsAdminModeActivated(ctx) {
		return nil, errors.New(
			"only admins with admin mode activated can create SCIM tokens",
			errors.WithErrorCode(errors.EForbidden))
	}

	if _, err := url.Parse(idpIssuerURL); err != nil {
		return nil, errors.Wrap(err, "failed to parse idpIssuerURL", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// Validate that the idpIssuerURL matches a configured identity provider
	validIssuer := false
	for _, idp := range s.oauthProviders {
		if idp.IssuerURL == idpIssuerURL {
			validIssuer = true
			break
		}
	}
	if !validIssuer {
		return nil, errors.New("identity provider issuer url does not match any configured identity provider", errors.WithErrorCode(errors.EInvalid), errors.WithSpan(span))
	}

	// Transaction is used to avoid invalidating previous token if new one fails creation.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateSCIMToken: %v", txErr)
		}
	}()

	// Find any previous token, so it can be invalidated (deleted).
	tokens, err := s.dbClient.SCIMTokens.GetTokens(txContext)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get tokens to invalidate", errors.WithSpan(span))
	}

	// Delete any previous tokens.
	if len(tokens) > 0 {
		for _, token := range tokens {
			tokenCopy := token
			err = s.dbClient.SCIMTokens.DeleteToken(txContext, &tokenCopy)
			if err != nil {
				return nil, errors.Wrap(err, "failed to delete any previous tokens", errors.WithSpan(span))
			}
		}
	}

	// Generate a token with a UUID claim.
	jwtID := uuid.New().String()
	scimToken, err := s.signingKeyManager.GenerateToken(txContext, &auth.TokenInput{
		Subject:  "scim",
		Audience: "tharsis",
		JwtID:    jwtID,
		Claims: map[string]string{
			"type":                     auth.SCIMTokenType,
			auth.IDPIssuerURLClaimName: idpIssuerURL,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate token", errors.WithSpan(span))
	}

	input := &models.SCIMToken{
		Nonce:     jwtID,
		CreatedBy: caller.GetActor(),
	}

	createdToken, err := s.dbClient.SCIMTokens.CreateToken(txContext, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create token", errors.WithSpan(span))
	}
	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a new SCIM token.")

	return &CreateSCIMTokenOutput{PlaintextToken: scimToken, SCIMToken: createdToken}, nil
}

func (s *service) GetSCIMUsers(ctx context.Context, input *GetSCIMUsersInput) ([]models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSCIMUsers")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Only the IDP's SCIM client may list SCIM users: the results include every provisioned user and
	// their SCIM external ID.
	scimCaller, err := requireSCIMCaller(ctx)
	if err != nil {
		return nil, err
	}

	// A SCIM client only sees the users its own IDP manages.
	issuer := scimCaller.GetIDPIssuerURL()

	users := []models.User{}

	// If SCIMExternalID is specified, use that instead.
	if input.SCIMExternalID != nil && *input.SCIMExternalID != "" {
		user, err := s.dbClient.Users.GetUserBySCIMExternalID(ctx, issuer, *input.SCIMExternalID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a SCIM user by scimExternalID", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}

		// If a user is not found, do not return an error.
		// Per SCIM, return an empty slice.
		if user != nil {
			users = append(users, *user)
		}
	} else if input.Username != nil && *input.Username != "" {
		user, err := s.dbClient.Users.GetUserByTRN(ctx, trn.TypeUser.Build(*input.Username))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a SCIM user by username", errors.WithSpan(span))
		}

		// Only add the user if this IDP manages them.
		if user != nil && scimCaller.Manages(user.SCIMIDPIssuerURL) {
			users = append(users, *user)
		}
	} else {
		// If no filter is specified, get all users this IDP manages.
		input := &db.GetUsersInput{
			Filter: &db.UserFilter{
				SCIMIDPIssuerURL: &issuer,
			},
		}
		result, err := s.dbClient.Users.GetUsers(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get users", errors.WithSpan(span))
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
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ManageSCIMUserPermission); err != nil {
		return nil, err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return nil, errors.New("only a scim caller may create scim users", errors.WithErrorCode(errors.EInternal))
	}
	issuer := scimCaller.GetIDPIssuerURL()

	if err = validateSCIMExternalID(input.SCIMExternalID); err != nil {
		return nil, err
	}

	existingUser, err := s.dbClient.Users.GetUserBySCIMExternalID(ctx, issuer, input.SCIMExternalID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by scim external id", errors.WithSpan(span))
	}

	// If user found by SCIM external ID, return proper SCIM error
	if existingUser != nil {
		return nil, errors.New(
			fmt.Sprintf("user with external id %s already exists", input.SCIMExternalID),
			errors.WithErrorCode(errors.EConflict))
	}

	// Adopt the user who signs in through this IDP with the external ID, if there is one.
	existingUser, err = s.dbClient.Users.GetUserByExternalID(ctx, issuer, input.SCIMExternalID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by external id", errors.WithSpan(span))
	}

	// Otherwise adopt an existing user with the same email.
	if existingUser == nil {
		existingUser, err = s.dbClient.Users.GetUserByEmail(ctx, strings.ToLower(input.Email))
		if err != nil {
			return nil, errors.Wrap(err, "failed to get user by email", errors.WithSpan(span))
		}
	}

	if existingUser != nil {
		return s.adoptSCIMUser(ctx, existingUser, input, issuer)
	}

	// No such user, so create it.
	txContext, err := s.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin DB transaction", errors.WithSpan(span))
	}

	defer func() {
		if txErr := s.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for CreateSCIMUser: %v", txErr)
		}
	}()

	createdUser, err := s.dbClient.Users.CreateUser(txContext, &models.User{
		Username:         auth.ParseUsername(input.Email),
		Email:            strings.ToLower(input.Email),
		SCIMExternalID:   input.SCIMExternalID,
		SCIMIDPIssuerURL: issuer,
		Active:           input.Active,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user", errors.WithSpan(span))
	}

	// Create external identity record for new user
	err = s.dbClient.Users.LinkUserWithExternalID(txContext, issuer, input.SCIMExternalID, createdUser.Metadata.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create external identity for new user", errors.WithSpan(span))
	}

	if err := s.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit DB transaction", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Created a SCIM user.",
		"userID", createdUser.Metadata.ID,
	)

	return createdUser, nil
}

// adoptSCIMUser brings an existing user, created without SCIM, under the management of the IDP's SCIM client.
// Adoption never links the SCIM external ID as a login identity: the SCIM caller chooses that value, so linking
// it would let whoever holds the SCIM token sign in as the existing user.
func (s *service) adoptSCIMUser(ctx context.Context, user *models.User, input *CreateSCIMUserInput, issuer string) (*models.User, error) {
	// Never take a user from another SCIM external ID or another IDP's SCIM client.
	if user.SCIMExternalID != "" || user.SCIMIDPIssuerURL != "" {
		return nil, errors.New(
			fmt.Sprintf("user with email %s is already managed by SCIM", user.Email),
			errors.WithErrorCode(errors.EConflict))
	}

	// Only adopt users who already sign in through this identity provider.
	linked, err := s.dbClient.Users.HasExternalIdentity(ctx, user.Metadata.ID, &issuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check user external identity")
	}

	if !linked {
		return nil, errors.New(
			fmt.Sprintf("user with email %s exists but has not signed in through this identity provider", user.Email),
			errors.WithErrorCode(errors.EConflict))
	}

	user.Active = input.Active
	user.SCIMExternalID = input.SCIMExternalID
	user.SCIMIDPIssuerURL = issuer
	user.Email = strings.ToLower(input.Email)
	// Don't update username - it should never change

	adoptedUser, err := s.dbClient.Users.UpdateUser(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user")
	}

	s.logger.WithContextFields(ctx).Infow("Adopted an existing user as a SCIM user.",
		"userID", adoptedUser.Metadata.ID,
	)

	return adoptedUser, nil
}

func (s *service) UpdateSCIMUser(ctx context.Context, input *UpdateResourceInput) (*models.User, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateSCIMUser")
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ManageSCIMUserPermission, auth.WithUserID(input.ID))
	if err != nil {
		return nil, err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return nil, errors.New("only a scim caller can update a SCIM user", errors.WithErrorCode(errors.EForbidden))
	}

	// Get the current user
	user, err := s.dbClient.Users.GetUserByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get user by ID", errors.WithSpan(span))
	}

	// Only update users the caller's IDP manages, never manually managed users or another IDP's.
	if user == nil || !scimCaller.Manages(user.SCIMIDPIssuerURL) {
		return nil, errors.New(
			"SCIM user not found",
			errors.WithErrorCode(errors.ENotFound))
	}

	// Process the operations to update user fields. Login identities are never linked or unlinked here,
	// since the SCIM caller chooses the externalId value.
	updatedUser, err := s.processSCIMUserOperations(input.Operations, user)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process SCIM user operations", errors.WithSpan(span))
	}

	updatedUser, err = s.dbClient.Users.UpdateUser(ctx, updatedUser)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a SCIM user.",
		"userID", updatedUser.Metadata.ID,
	)

	return updatedUser, nil
}

func (s *service) DeleteSCIMUser(ctx context.Context, input *DeleteSCIMResourceInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteSCIMUser")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, models.ManageSCIMUserPermission, auth.WithUserID(input.ID))
	if err != nil {
		return err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return errors.New("only a scim caller can delete a SCIM user", errors.WithErrorCode(errors.EForbidden))
	}

	user, err := s.dbClient.Users.GetUserByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get user by ID", errors.WithSpan(span))
	}

	// Only delete users the caller's IDP manages, never manually managed users or another IDP's.
	if user == nil || !scimCaller.Manages(user.SCIMIDPIssuerURL) {
		return errors.New(
			"SCIM user with ID %s does not exist", input.ID,
			errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.dbClient.Users.DeleteUser(ctx, user); err != nil {
		return errors.Wrap(err, "failed to delete user", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a SCIM user.",
		"userID", user.Metadata.ID,
	)

	return nil
}

func (s *service) GetSCIMGroups(ctx context.Context, input *GetSCIMGroupsInput) ([]models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.GetSCIMGroups")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	// Only the IDP's SCIM client may list SCIM groups, for the same reason as GetSCIMUsers, and it only
	// sees the teams of its own IDP.
	scimCaller, err := requireSCIMCaller(ctx)
	if err != nil {
		return nil, err
	}
	issuer := scimCaller.GetIDPIssuerURL()

	var teams []models.Team

	// If SCIMExternalID is specified, use that instead.
	if input.SCIMExternalID != "" {
		team, err := s.dbClient.Teams.GetTeamBySCIMExternalID(ctx, issuer, input.SCIMExternalID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get a SCIM group by scimExternalID", errors.WithErrorCode(errors.ENotFound), errors.WithSpan(span))
		}

		// If a team is not found, do not return an error.
		// Per SCIM, return an empty slice.
		if team != nil {
			teams = append(teams, *team)
		}
	} else {
		// If no filter is specified, get all teams this IDP manages.
		input := &db.GetTeamsInput{
			Filter: &db.TeamFilter{
				SCIMIDPIssuerURL: &issuer,
			},
		}
		result, err := s.dbClient.Teams.GetTeams(ctx, input)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get teams", errors.WithSpan(span))
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
		return nil, err
	}

	if err = caller.RequirePermission(ctx, models.ManageSCIMTeamPermission); err != nil {
		return nil, err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return nil, errors.New("caller is not a SCIM caller", errors.WithErrorCode(errors.EForbidden), errors.WithSpan(span))
	}
	issuer := scimCaller.GetIDPIssuerURL()

	if err = validateSCIMExternalID(input.SCIMExternalID); err != nil {
		return nil, err
	}

	// Check if team with same name exists.
	existingTeam, err := s.dbClient.Teams.GetTeamByTRN(ctx, trn.TypeTeam.Build(input.Name))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team by TRN", errors.WithSpan(span))
	}

	var createdTeam *models.Team

	// Never adopt a team this IDP doesn't manage. A manually managed team may already hold namespace
	// memberships that the SCIM caller would gain control of, and another IDP's team belongs to its SCIM client.
	if existingTeam != nil && !scimCaller.Manages(existingTeam.SCIMIDPIssuerURL) {
		return nil, errors.New(
			fmt.Sprintf("team with name %s already exists and is not managed by this identity provider", input.Name),
			errors.WithErrorCode(errors.EConflict))
	}

	// Team is already managed by this IDP's SCIM client, so update it.
	if existingTeam != nil {
		existingTeam.SCIMExternalID = input.SCIMExternalID
		createdTeam, err = s.dbClient.Teams.UpdateTeam(ctx, existingTeam)
		if err != nil {
			return nil, errors.Wrap(err, "failed to update team", errors.WithSpan(span))
		}
	}

	// No such Team, so create it.
	if existingTeam == nil {
		newTeam := &models.Team{
			Name:             input.Name,
			SCIMExternalID:   input.SCIMExternalID,
			SCIMIDPIssuerURL: issuer,
		}

		createdTeam, err = s.dbClient.Teams.CreateTeam(ctx, newTeam)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create team", errors.WithSpan(span))
		}
	}

	s.logger.WithContextFields(ctx).Infow("Created a SCIM group.",
		"teamID", createdTeam.Metadata.ID,
	)

	return createdTeam, nil
}

func (s *service) UpdateSCIMGroup(ctx context.Context, input *UpdateResourceInput) (*models.Team, error) {
	ctx, span := tracer.Start(ctx, "svc.UpdateSCIMGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	err = caller.RequirePermission(ctx, models.ManageSCIMTeamPermission, auth.WithTeamID(input.ID))
	if err != nil {
		return nil, err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return nil, errors.New("only a scim caller can update a SCIM group", errors.WithErrorCode(errors.EForbidden))
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, input.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get team by ID", errors.WithSpan(span))
	}

	// Only update teams the caller's IDP manages, never manually managed teams or another IDP's.
	if team == nil || !scimCaller.Manages(team.SCIMIDPIssuerURL) {
		return nil, errors.New(
			"SCIM group with ID %s not found", input.ID,
			errors.WithErrorCode(errors.ENotFound))
	}

	updatedTeam, err := s.processSCIMGroupOperations(ctx, input.Operations, team)
	if err != nil {
		return nil, errors.Wrap(err, "failes to process SCIM group operations", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Updated a SCIM group.",
		"teamID", updatedTeam.Metadata.ID,
	)

	return updatedTeam, nil
}

func (s *service) DeleteSCIMGroup(ctx context.Context, input *DeleteSCIMResourceInput) error {
	ctx, span := tracer.Start(ctx, "svc.DeleteSCIMGroup")
	// TODO: Consider setting trace/span attributes for the input.
	defer span.End()

	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return err
	}

	err = caller.RequirePermission(ctx, models.ManageSCIMTeamPermission, auth.WithTeamID(input.ID))
	if err != nil {
		return err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return errors.New("only a scim caller can delete a SCIM group", errors.WithErrorCode(errors.EForbidden))
	}

	team, err := s.dbClient.Teams.GetTeamByID(ctx, input.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get team by ID", errors.WithSpan(span))
	}

	// Only delete teams the caller's IDP manages, never manually managed teams or another IDP's.
	if team == nil || !scimCaller.Manages(team.SCIMIDPIssuerURL) {
		return errors.New(
			"SCIM group with ID %s not found", input.ID,
			errors.WithErrorCode(errors.ENotFound))
	}

	if err := s.dbClient.Teams.DeleteTeam(ctx, team); err != nil {
		return errors.Wrap(err, "failed to delete team", errors.WithSpan(span))
	}

	s.logger.WithContextFields(ctx).Infow("Deleted a SCIM group.",
		"teamID", team.Metadata.ID,
	)

	return nil
}

// processSCIMUserOperations processes the SCIM PATCH operations,
// and updates the user fields appropriately. Returns an error
// if operation is not supported.
func (s *service) processSCIMUserOperations(operations []Operation, user *models.User) (*models.User, error) {
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
			// Don't update username - it should never change
		case "externalId":
			externalID, ok := operation.Value.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected value for user externalId field: expected string, got %T", operation.Value)
			}
			if err = validateSCIMExternalID(externalID); err != nil {
				return nil, err
			}
			// The SCIM external ID is the IDP's handle for the user, separate from their login identity:
			// an IDP changes it when its SCIM attribute mapping changes, which needn't change the token
			// claim the user signs in with. The login identity is never relinked here, since the SCIM
			// caller chooses this value and could otherwise sign in as the user.
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
func (s *service) processSCIMGroupOperations(ctx context.Context, ops []Operation, team *models.Team) (*models.Team, error) {
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
			s.logger.WithContextFields(ctx).Errorf("failed to rollback tx for processSCIMGroupOperations: %v", txErr)
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

	// Add a new group (team) member. Only users managed by the team's IDP can be added: anyone else would
	// gain the team's namespace memberships at the SCIM caller's discretion.
	if operation.OP.Equal(addOPType) {
		if user.SCIMIDPIssuerURL != team.SCIMIDPIssuerURL {
			return errors.New(
				"scim user with id %s is not managed by this identity provider", userID,
				errors.WithErrorCode(errors.ENotFound))
		}

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

			if err = validateSCIMExternalID(externalID); err != nil {
				return false, err
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

// validateSCIMExternalID returns an error if a SCIM external ID is empty. A user or team managed by SCIM
// always has one, which keeps its external ID and IDP issuer set together.
func validateSCIMExternalID(externalID string) error {
	if externalID == "" {
		return errors.New("SCIM externalId must not be empty", errors.WithErrorCode(errors.EInvalid))
	}
	return nil
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

// requireSCIMCaller returns the caller if it authenticated with a SCIM token, and an EForbidden error
// for any other authenticated caller.
func requireSCIMCaller(ctx context.Context) (*auth.SCIMCaller, error) {
	caller, err := auth.AuthorizeCaller(ctx)
	if err != nil {
		return nil, err
	}

	scimCaller, ok := caller.(*auth.SCIMCaller)
	if !ok {
		return nil, errors.New("caller is not a SCIM caller", errors.WithErrorCode(errors.EForbidden))
	}

	return scimCaller, nil
}
