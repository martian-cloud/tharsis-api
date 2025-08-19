package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// UserCaller represents a user subject
type UserCaller struct {
	User               *models.User
	UserSessionID      *string
	authorizer         Authorizer
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
	teamListCache      []models.Team // lazy init
}

// NewUserCaller returns a new UserCaller
func NewUserCaller(
	user *models.User,
	authorizer Authorizer,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
	userSessionID *string,
) *UserCaller {
	return &UserCaller{
		User:               user,
		UserSessionID:      userSessionID,
		authorizer:         authorizer,
		dbClient:           dbClient,
		maintenanceMonitor: maintenanceMonitor,
	}
}

// GetSubject returns the subject identifier for this caller
func (u *UserCaller) GetSubject() string {
	return u.User.Email
}

// IsAdmin returns true if the caller is an admin
func (u *UserCaller) IsAdmin() bool {
	return u.User.Admin
}

// UnauthorizedError returns the unauthorized error for this specific caller type
func (u *UserCaller) UnauthorizedError(_ context.Context, hasViewerAccess bool) error {
	// If subject has at least viewer permissions then return 403, if not, return 404
	if hasViewerAccess {
		return terrors.New(
			"user %s is not authorized to perform the requested operation: ensure that the user has been added as a member to the group/workspace with the role required to perform the requested operation",
			u.GetSubject(),
			terrors.WithErrorCode(terrors.EForbidden),
		)
	}

	return terrors.New(
		"either the requested resource does not exist or the user %s is not authorized to perform the requested operation: ensure that the user has been added as a member to the group/workspace with the role required to perform the requested operation",
		u.GetSubject(),
		terrors.WithErrorCode(terrors.ENotFound),
	)
}

// GetNamespaceAccessPolicy returns the namespace access policy for this caller
func (u *UserCaller) GetNamespaceAccessPolicy(ctx context.Context) (*NamespaceAccessPolicy, error) {
	if u.User.Admin {
		return &NamespaceAccessPolicy{AllowAll: true}, nil
	}

	rootNamespaces, err := u.authorizer.GetRootNamespaces(ctx)
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, ns := range rootNamespaces {
		ids = append(ids, ns.ID)
	}

	return &NamespaceAccessPolicy{AllowAll: false, RootNamespaceIDs: ids}, nil
}

// RequirePermission will return an error if the caller doesn't have the specified permissions
func (u *UserCaller) RequirePermission(ctx context.Context, perm models.Permission, checks ...func(*constraints)) error {
	inMaintenance, err := u.maintenanceMonitor.InMaintenanceMode(ctx)
	if err != nil {
		return err
	}

	if inMaintenance && perm.Action != models.ViewAction && perm.Action != models.ViewValueAction {
		// Server is in maintenance mode, only allow view permissions
		return errInMaintenanceMode
	}

	if perm.IsAssignable() && u.User.Admin {
		// User is an admin, so assignable permission can be granted.
		return nil
	}

	if handlerFunc, ok := u.getPermissionHandler(perm); ok {
		return handlerFunc(ctx, &perm, getConstraints(checks...))
	}

	// Lastly, check the authorizer.
	return u.authorizer.RequireAccess(ctx, []models.Permission{perm}, checks...)
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (u *UserCaller) RequireAccessToInheritableResource(ctx context.Context, modelType types.ModelType, checks ...func(*constraints)) error {
	perm := models.Permission{Action: models.ViewAction, ResourceType: modelType.Name()}
	if perm.IsAssignable() && u.User.Admin {
		// User is an admin, so assignable permission can be granted.
		return nil
	}

	return u.authorizer.RequireAccessToInheritableResource(ctx, []types.ModelType{modelType}, checks...)
}

// requireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (u *UserCaller) requireTeamUpdateAccess(ctx context.Context, _ *models.Permission, checks *constraints) error {
	if checks.teamID == nil {
		return errMissingConstraints
	}

	if u.User.Admin {
		return nil
	}

	teamMember, err := u.dbClient.TeamMembers.GetTeamMember(ctx, u.User.Metadata.ID, *checks.teamID)
	if err != nil {
		return err
	}

	// Allow access only if caller is a team member and is a maintainer.
	if teamMember != nil && teamMember.IsMaintainer {
		return nil
	}

	// All others are denied. Viewer access is available to everyone.
	return u.UnauthorizedError(ctx, true)
}

func (u *UserCaller) requireAdmin(ctx context.Context, _ *models.Permission, _ *constraints) error {
	if u.User.Admin {
		return nil
	}

	return u.UnauthorizedError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (u *UserCaller) getPermissionHandler(perm models.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[models.Permission]permissionTypeHandler{
		models.CreateTeamPermission: u.requireAdmin,
		models.DeleteTeamPermission: u.requireAdmin,
		models.CreateUserPermission: u.requireAdmin,
		models.UpdateUserPermission: u.requireAdmin,
		models.DeleteUserPermission: u.requireAdmin,
		models.UpdateTeamPermission: u.requireTeamUpdateAccess,
	}

	handler, ok := handlerMap[perm]
	return handler, ok
}

// GetTeams does lazy initialization of the list of teams for this user caller.
func (u *UserCaller) GetTeams(ctx context.Context) ([]models.Team, error) {
	if u.teamListCache != nil {
		return u.teamListCache, nil
	}

	getResult, err := u.dbClient.Teams.GetTeams(ctx, &db.GetTeamsInput{
		Filter: &db.TeamFilter{
			UserID: &u.User.Metadata.ID,
		},
	})
	if err != nil {
		return nil, err
	}
	result := getResult.Teams

	u.teamListCache = result

	return result, nil
}

type externalIdentity struct {
	ID       string
	Issuer   string
	Username string
	Email    string
}

// UserAuth implements JWT authentication
type UserAuth struct {
	idpMap             map[string]config.IdpConfig
	oidcTokenVerifier  OIDCTokenVerifier
	logger             logger.Logger
	dbClient           *db.Client
	maintenanceMonitor maintenance.Monitor
}

// NewUserAuth creates an instance of UserAuth
func NewUserAuth(
	ctx context.Context,
	identityProviders []config.IdpConfig,
	logger logger.Logger,
	dbClient *db.Client,
	maintenanceMonitor maintenance.Monitor,
	oidcConfigFetcher OpenIDConfigFetcher,
) *UserAuth {
	idpMap := make(map[string]config.IdpConfig)
	issuers := []string{}

	for _, idp := range identityProviders {
		idpMap[idp.IssuerURL] = idp
		issuers = append(issuers, idp.IssuerURL)
	}

	oidcTokenVerifier := NewOIDCTokenVerifier(ctx, issuers, oidcConfigFetcher, true)

	return &UserAuth{idpMap, oidcTokenVerifier, logger, dbClient, maintenanceMonitor}
}

// Use checks if the UserAuth instance can handle the given issuer URL
func (u *UserAuth) Use(token jwt.Token) bool {
	_, ok := u.idpMap[token.Issuer()]
	return ok
}

// GetUsernameClaim returns the username from a JWT token
func (u *UserAuth) GetUsernameClaim(token jwt.Token) (string, error) {
	idp, ok := u.idpMap[token.Issuer()]
	if !ok {
		return "", errors.New("Identity provider not found for token with issuer " + token.Issuer())
	}

	username, ok := token.Get(idp.UsernameClaim)
	if !ok {
		return "", terrors.New("Token with issuer "+token.Issuer()+" is missing "+idp.UsernameClaim+" field", terrors.WithErrorCode(terrors.EUnauthorized))
	}

	return username.(string), nil
}

// Authenticate validates a user JWT and returns a UserCaller
func (u *UserAuth) Authenticate(ctx context.Context, tokenString string, useCache bool) (Caller, error) {
	decodedToken, err := u.oidcTokenVerifier.VerifyToken(ctx, tokenString, []jwt.ValidateOption{})
	if err != nil {
		return nil, terrors.New(errorReason(err), terrors.WithErrorCode(terrors.EUnauthorized))
	}

	issuer := decodedToken.Issuer()
	idp, ok := u.idpMap[issuer]
	if !ok {
		// This should never happen because the token will be invalid if the issuer isn't supported but we'll check just
		// in case
		return nil, fmt.Errorf("identity provider not found for issuer %s", issuer)
	}

	if !slices.Contains(decodedToken.Audience(), idp.ClientID) {
		return nil, terrors.New("token from issuer %s missing required aud %s", issuer, idp.ClientID, terrors.WithErrorCode(terrors.EUnauthorized))
	}

	username, err := u.GetUsernameClaim(decodedToken)
	if err != nil {
		return nil, err
	}

	// Get user from DB, user will be created if this is the first login
	userModel, err := u.getUserWithExternalID(ctx, issuer, decodedToken.Subject())
	if err != nil {
		return nil, err
	}

	if userModel == nil {
		email, ok := decodedToken.Get("email")
		if !ok {
			return nil, terrors.New("email claim missing from token", terrors.WithErrorCode(terrors.EUnauthorized))
		}

		userModel, err = u.createUser(ctx, &externalIdentity{
			Issuer:   issuer,
			ID:       decodedToken.Subject(),
			Username: ParseUsername(username),
			Email:    strings.ToLower(email.(string)),
		})
		if err != nil {
			u.logger.Errorf("Failed to create user in db: %v", err)
			return nil, terrors.Wrap(err, "failed to create user in db")
		}
	}

	// If user is not active (disabled via SCIM), return EUnauthorized.
	if !userModel.Active {
		return nil, terrors.New(
			"User has been disabled",
			terrors.WithErrorCode(terrors.EUnauthorized),
		)
	}

	return NewUserCaller(
		userModel,
		newNamespaceMembershipAuthorizer(u.dbClient, &userModel.Metadata.ID, nil, useCache),
		u.dbClient,
		u.maintenanceMonitor,
		nil,
	), nil
}

func (u *UserAuth) getUserWithExternalID(ctx context.Context, issuer string, externalID string) (*models.User, error) {
	user, err := u.dbClient.Users.GetUserByExternalID(ctx, issuer, externalID)
	if err != nil {
		return nil, terrors.Wrap(err, "failed to get user by external identity")
	}
	return user, nil
}

func (u *UserAuth) createUser(ctx context.Context, identity *externalIdentity) (*models.User, error) {
	// Create user since this if the first time we've seen this user identity
	user, err := u.createUserWithExternalID(ctx, identity)
	if err != nil && terrors.ErrorCode(err) == terrors.EConflict {
		// The conflict error may be due to an existing user that is using the same username or the user for
		// the external identity may already exist but this is the first time we've seen this external ID. If
		// the user's email matches then we can link the external ID to the existing user.

		// Query user by email to check if email matches
		user, err = u.dbClient.Users.GetUserByEmail(ctx, identity.Email)
		if err != nil {
			return nil, err
		}

		if user != nil {
			err = u.dbClient.Users.LinkUserWithExternalID(ctx, identity.Issuer, identity.ID, user.Metadata.ID)
			// Ignore conflict errors since another instance may have already linked the external identity
			if err != nil && terrors.ErrorCode(err) != terrors.EConflict {
				return nil, terrors.Wrap(
					err,
					"failed to link user with external identity")
			}
		} else {
			// User not found with email so we need to create a new user with a number added to their username
			resp, uErr := u.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{Filter: &db.UserFilter{UsernamePrefix: &identity.Username}})
			if uErr != nil {
				return nil, uErr
			}
			newUsername := fmt.Sprintf("%s%d", identity.Username, resp.PageInfo.TotalCount+1)
			// Create user with new username
			user, err = u.createUserWithExternalID(ctx, &externalIdentity{
				ID:       identity.ID,
				Issuer:   identity.Issuer,
				Email:    identity.Email,
				Username: newUsername,
			})
			if err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}

	return user, nil
}

func (u *UserAuth) createUserWithExternalID(ctx context.Context, identity *externalIdentity) (*models.User, error) {
	txContext, err := u.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() {
		if txErr := u.dbClient.Transactions.RollbackTx(txContext); txErr != nil {
			u.logger.Errorf("failed to rollback tx for createUserWithExternalID: %v", txErr)
		}
	}()

	input := &models.User{
		Username: identity.Username,
		Email:    identity.Email,
		Active:   true, // Make sure this user is set as active or auth will fail.
	}

	user, err := u.dbClient.Users.CreateUser(txContext, input)
	if err != nil {
		return nil, err
	}

	err = u.dbClient.Users.LinkUserWithExternalID(txContext, identity.Issuer, identity.ID, user.Metadata.ID)
	if err != nil {
		return nil, err
	}

	if err := u.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, err
	}

	return user, nil
}

// ParseUsername parses the username, if any, from the email.
func ParseUsername(username string) string {
	resp := username
	at := strings.LastIndex(username, "@")
	if at >= 0 {
		// Remove email domain
		resp = username[:at]
	}
	// Convert username to lowercase
	return strings.ToLower(resp)
}
