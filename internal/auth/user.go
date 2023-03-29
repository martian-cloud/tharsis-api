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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	terrors "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
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
	return &UserCaller{User: user, authorizer: authorizer, dbClient: dbClient}
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

// RequireAccessToNamespace will return an error if the caller doesn't have the specified access level
func (u *UserCaller) RequireAccessToNamespace(ctx context.Context, namespacePath string, accessLevel models.Role) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireAccessToNamespace(ctx, namespacePath, accessLevel)
}

// RequireAccessToGroup will return an error if the caller doesn't have the required access level on the specified group
func (u *UserCaller) RequireAccessToGroup(ctx context.Context, groupID string, accessLevel models.Role) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireAccessToGroup(ctx, groupID, accessLevel)
}

// RequireAccessToInheritedGroupResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (u *UserCaller) RequireAccessToInheritedGroupResource(ctx context.Context, groupID string) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireAccessToInheritedGroupResource(ctx, groupID)
}

// RequireAccessToInheritedNamespaceResource will return an error if the caller doesn't have viewer access on any namespace within the namespace hierarchy
func (u *UserCaller) RequireAccessToInheritedNamespaceResource(ctx context.Context, namespace string) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireAccessToInheritedNamespaceResource(ctx, namespace)
}

// RequireAccessToWorkspace will return an error if the caller doesn't have the required access level on the specified workspace
func (u *UserCaller) RequireAccessToWorkspace(ctx context.Context, workspaceID string, accessLevel models.Role) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireAccessToWorkspace(ctx, workspaceID, accessLevel)
}

// RequireViewerAccessToGroups will return an error if the caller doesn't have viewer access to all the specified groups
func (u *UserCaller) RequireViewerAccessToGroups(ctx context.Context, groups []models.Group) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireViewerAccessToGroups(ctx, groups)
}

// RequireViewerAccessToWorkspaces will return an error if the caller doesn't have viewer access on the specified workspace
func (u *UserCaller) RequireViewerAccessToWorkspaces(ctx context.Context, workspaces []models.Workspace) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireViewerAccessToWorkspaces(ctx, workspaces)
}

// RequireViewerAccessToNamespaces will return an error if the caller doesn't have viewer access to the specified list of namespaces
func (u *UserCaller) RequireViewerAccessToNamespaces(ctx context.Context, namespaces []string) error {
	if u.User.Admin {
		return nil
	}
	return u.authorizer.RequireViewerAccessToNamespaces(ctx, namespaces)
}

// RequireRunWriteAccess will return an error if the caller doesn't have permission to update run state
func (u *UserCaller) RequireRunWriteAccess(ctx context.Context, runID string) error {
	// Return authorization error because users don't have run write access
	return authorizationError(ctx, false)
}

// RequirePlanWriteAccess will return an error if the caller doesn't have permission to update plan state
func (u *UserCaller) RequirePlanWriteAccess(ctx context.Context, planID string) error {
	// Return authorization error because users don't have plan write access
	return authorizationError(ctx, false)
}

// RequireApplyWriteAccess will return an error if the caller doesn't have permission to update apply state
func (u *UserCaller) RequireApplyWriteAccess(ctx context.Context, applyID string) error {
	// Return authorization error because users don't have apply write access
	return authorizationError(ctx, false)
}

// RequireJobWriteAccess will return an error if the caller doesn't have permission to update the state of the specified job
func (u *UserCaller) RequireJobWriteAccess(ctx context.Context, jobID string) error {
	// Return authorization error because users accounts don't have job write access
	return authorizationError(ctx, false)
}

// RequireRunnerAccess will return an error if the caller is not allowed to claim a job as the specified runner
func (u *UserCaller) RequireRunnerAccess(ctx context.Context, runnerID string) error {
	// Return authorization error because users don't have runner access
	return authorizationError(ctx, false)
}


// RequireTeamCreateAccess will return an error if the specified access is not allowed to the indicated team.
// For now, only admins are allowed to create a team.
// Eventually, org admins and SCIM will be allowed to create and delete teams.
func (u *UserCaller) RequireTeamCreateAccess(ctx context.Context) error {
	if u.User.Admin {
		return nil
	}

	// All users have viewer access to teams.
	return authorizationError(ctx, true)
}

// RequireTeamUpdateAccess will return an error if the specified access is not allowed to the indicated team.
func (u *UserCaller) RequireTeamUpdateAccess(ctx context.Context, teamID string) error {
	if u.User.Admin {
		return nil
	}

	teamMember, err := u.dbClient.TeamMembers.GetTeamMember(ctx, u.User.Metadata.ID, teamID)
	if err != nil {
		return err
	}

	// Allow access only if caller is a team member and is a maintainer.
	if teamMember != nil && teamMember.IsMaintainer {
		return nil
	}

	// All others are denied.  Viewer access is available to everyone.
	return authorizationError(ctx, true)
}

// RequireTeamDeleteAccess will return an error if the specified access is not allowed to the indicated team.
// For now, only admins are allowed to delete a team.
// Eventually, org admins and SCIM will be allowed to create and delete teams.
func (u *UserCaller) RequireTeamDeleteAccess(ctx context.Context, teamID string) error {
	if u.User.Admin {
		return nil
	}

	return authorizationError(ctx, false)
}

// RequireUserCreateAccess will return an error if the specified caller is not allowed to create users.
func (u *UserCaller) RequireUserCreateAccess(ctx context.Context) error {
	if u.User.Admin {
		return nil
	}

	// All others are denied.  Viewer access is available to everyone.
	return authorizationError(ctx, true)
}

// RequireUserUpdateAccess will return an error if the specified caller is not allowed to update a user.
func (u *UserCaller) RequireUserUpdateAccess(ctx context.Context, userID string) error {
	if u.User.Admin {
		return nil
	}

	// All others are denied.  Viewer access is available to everyone.
	return authorizationError(ctx, true)
}

// RequireUserDeleteAccess will return an error if the specified caller is not allowed to delete a user.
func (u *UserCaller) RequireUserDeleteAccess(ctx context.Context, userID string) error {
	if u.User.Admin {
		return nil
	}

	return authorizationError(ctx, false)
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
		return nil, terrors.NewError(
			terrors.EUnauthorized,
			"User has been disabled",
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
		return nil, terrors.NewError(
			terrors.EInternal,
			"Failed to get user by external identity",
			terrors.WithErrorErr(err),
		)
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
				return nil, terrors.NewError(
					terrors.EInternal,
					"Failed to link user with external identity",
					terrors.WithErrorErr(err),
				)
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
