package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth/permissions"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// UserCaller represents a user subject
type UserCaller struct {
	User          *models.User
	authorizer    Authorizer
	dbClient      *db.Client
	teamListCache []models.Team // lazy init
}

// NewUserCaller returns a new UserCaller
func NewUserCaller(user *models.User, authorizer Authorizer, dbClient *db.Client) *UserCaller {
	return &UserCaller{
		User:       user,
		authorizer: authorizer,
		dbClient:   dbClient,
	}
}

// GetSubject returns the subject identifier for this caller
func (u *UserCaller) GetSubject() string {
	return u.User.Email
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
func (u *UserCaller) RequirePermission(ctx context.Context, perm permissions.Permission, checks ...func(*constraints)) error {
	if perm.IsAssignable() && u.User.Admin {
		// User is an admin, so assignable permission can be granted.
		return nil
	}

	if handlerFunc, ok := u.getPermissionHandler(perm); ok {
		return handlerFunc(ctx, &perm, getConstraints(checks...))
	}

	// Lastly, check the authorizer.
	return u.authorizer.RequireAccess(ctx, []permissions.Permission{perm}, checks...)
}

// RequireAccessToInheritableResource will return an error if caller doesn't have permissions to inherited resources.
func (u *UserCaller) RequireAccessToInheritableResource(ctx context.Context, resourceType permissions.ResourceType, checks ...func(*constraints)) error {
	perm := permissions.Permission{Action: permissions.ViewAction, ResourceType: resourceType}
	if perm.IsAssignable() && u.User.Admin {
		// User is an admin, so assignable permission can be granted.
		return nil
	}

	return u.authorizer.RequireAccessToInheritableResource(ctx, []permissions.ResourceType{resourceType}, checks...)
}

// requireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (u *UserCaller) requireTeamUpdateAccess(ctx context.Context, _ *permissions.Permission, checks *constraints) error {
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
	return authorizationError(ctx, true)
}

func (u *UserCaller) requireAdmin(ctx context.Context, _ *permissions.Permission, _ *constraints) error {
	if u.User.Admin {
		return nil
	}

	return authorizationError(ctx, false)
}

// getPermissionHandler returns a permissionTypeHandler for a given permission.
func (u *UserCaller) getPermissionHandler(perm permissions.Permission) (permissionTypeHandler, bool) {
	handlerMap := map[permissions.Permission]permissionTypeHandler{
		permissions.CreateTeamPermission: u.requireAdmin,
		permissions.DeleteTeamPermission: u.requireAdmin,
		permissions.CreateUserPermission: u.requireAdmin,
		permissions.UpdateUserPermission: u.requireAdmin,
		permissions.DeleteUserPermission: u.requireAdmin,
		permissions.UpdateTeamPermission: u.requireTeamUpdateAccess,
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

// IdentityProviderConfig encompasses the information for an identity provider
type IdentityProviderConfig struct {
	Issuer        string
	ClientID      string
	UsernameClaim string
	JwksURI       string
	TokenEndpoint string
	AuthEndpoint  string
}

type externalIdentity struct {
	ID       string
	Issuer   string
	Username string
	Email    string
}

// UserAuth implements JWT authentication
type UserAuth struct {
	idpMap      map[string]IdentityProviderConfig
	jwkRegistry *jwk.AutoRefresh
	logger      logger.Logger
	dbClient    *db.Client
}

const (
	defaultKeyAlgorithm         = jwa.RS256
	jwtRefreshIntervalInMinutes = 60
)

// NewUserAuth creates an instance of UserAuth
func NewUserAuth(
	ctx context.Context,
	identityProviders []IdentityProviderConfig,
	logger logger.Logger,
	dbClient *db.Client,
) *UserAuth {
	idpMap := make(map[string]IdentityProviderConfig)

	jwkRegistry := jwk.NewAutoRefresh(ctx)

	for _, idp := range identityProviders {
		idpMap[idp.Issuer] = idp

		jwkRegistry.Configure(idp.JwksURI, jwk.WithMinRefreshInterval(jwtRefreshIntervalInMinutes*time.Minute))

		_, err := jwkRegistry.Refresh(ctx, idp.JwksURI)
		if err != nil {
			logger.Errorf("Failed to load keyset for IDP %s: %s", idp.Issuer, err)
		}
	}

	return &UserAuth{idpMap, jwkRegistry, logger, dbClient}
}

func (u *UserAuth) getKey(ctx context.Context, kid string, idp IdentityProviderConfig) (jwk.Key, error) {
	keyset, err := u.jwkRegistry.Fetch(ctx, idp.JwksURI)
	if err != nil {
		return nil, errors.New("Failed to load key set for identity provider " + idp.Issuer)
	}

	key, found := keyset.LookupKeyID(kid)
	if !found {
		// Attempt to refresh the keyset for the IDP because the keys may have been updated
		keyset, err := u.jwkRegistry.Refresh(ctx, idp.JwksURI)
		if err != nil {
			return nil, errors.New("Failed to load key set for identity provider " + idp.Issuer)
		}

		key, found = keyset.LookupKeyID(kid)
		if !found {
			return nil, errors.New("Failed to load key set for identity provider " + idp.Issuer)
		}

		return key, nil
	}

	return key, nil
}

// GetUsernameClaim returns the username from a JWT token
func (u *UserAuth) GetUsernameClaim(token jwt.Token) (string, error) {
	idp, ok := u.idpMap[token.Issuer()]
	if !ok {
		return "", errors.New("Identity provider not found for token with issuer " + token.Issuer())
	}

	username, ok := token.Get(idp.UsernameClaim)
	if !ok {
		return "", errors.New("Token with issuer " + token.Issuer() + " is missing " + idp.UsernameClaim + " field")
	}

	return username.(string), nil
}

// Authenticate validates a user JWT and returns a UserCaller
func (u *UserAuth) Authenticate(ctx context.Context, tokenString string, useCache bool) (*UserCaller, error) {
	tokenBytes := []byte(tokenString)

	// Parse token headers
	msg, err := jws.Parse(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token headers %w", err)
	}

	signatures := msg.Signatures()
	if len(signatures) < 1 {
		return nil, errors.New("token is missing signature")
	}

	// Parse jwt
	decodedToken, err := jwt.Parse(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token %w", err)
	}

	kid := signatures[0].ProtectedHeaders().KeyID()
	idp := u.idpMap[decodedToken.Issuer()]

	key, err := u.getKey(ctx, kid, idp)
	if err != nil {
		return nil, err
	}

	alg := jwa.SignatureAlgorithm(key.Algorithm())
	if alg == "" {
		alg = defaultKeyAlgorithm
	}

	// Verify Token Signature
	if _, vErr := jws.Verify(tokenBytes, alg, key); vErr != nil {
		return nil, vErr
	}

	// Validate claims
	if vErr := jwt.Validate(decodedToken, jwt.WithAudience(idp.ClientID), jwt.WithIssuer(idp.Issuer)); vErr != nil {
		return nil, vErr
	}

	username, err := u.GetUsernameClaim(decodedToken)
	if err != nil {
		return nil, err
	}

	// Get user from DB, user will be created if this is the first login
	userModel, err := u.getUserWithExternalID(ctx, decodedToken.Issuer(), decodedToken.Subject())
	if err != nil {
		return nil, err
	}

	if userModel == nil {
		email, ok := decodedToken.Get("email")
		if !ok {
			return nil, fmt.Errorf("email claim missing from token")
		}

		userModel, err = u.createUser(ctx, &externalIdentity{
			Issuer:   decodedToken.Issuer(),
			ID:       decodedToken.Subject(),
			Username: ParseUsername(username),
			Email:    strings.ToLower(email.(string)),
		})
		if err != nil {
			u.logger.Errorf("Failed to create user in db: %v", err)
			return nil, err
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
