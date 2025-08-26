package auth

//go:generate go tool mockery --name UserSessionManager --inpackage --case underscore

import (
	"context"
	"fmt"
	"net/http"
	"time"

	goerrors "errors"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

const (
	// SessionIDClaim is the claim name for the session id in the jwt token
	SessionIDClaim = "sid"
	// CSRFTokenHeader is the header used to pass the CSRF token
	CSRFTokenHeader = "X-Csrf-Token"
)

const (
	refreshTokenType       = "user_session_refresh"
	tokenAudience          = "tharsis"
	accessTokenCookieName  = "tharsis_access_token"
	refreshTokenCookieName = "tharsis_refresh_token"
	csrfTokenCookieName    = "tharsis_csrf_token"
	// CookieHostPrefix is the cookie prefix used to restrict the cookie to the exact domain that sets it
	cookieHostPrefix = "__Host-"
)

// CreateSessionResponse is the response for creating a session
type CreateSessionResponse struct {
	AccessToken       string
	RefreshToken      string
	CSRFToken         string
	SessionExpiration time.Time
}

// RefreshSessionResponse is the response for updating a session
type RefreshSessionResponse struct {
	AccessToken       string
	RefreshToken      string
	SessionExpiration time.Time
}

// UserSessionManager interface defines the operations for managing user sessions
type UserSessionManager interface {
	CreateSession(ctx context.Context, token string, userAgent string) (*CreateSessionResponse, error)
	RefreshSession(ctx context.Context, refreshToken string) (*RefreshSessionResponse, error)
	InvalidateSession(ctx context.Context, accessToken, refreshToken string) error
	VerifyCSRFToken(ctx context.Context, requestSessionID string, csrfToken string) error
}

// userSessionManager implements the UserSessionManager interface
type userSessionManager struct {
	dbClient                      *db.Client
	idp                           IdentityProvider
	authenticator                 Authenticator
	logger                        logger.Logger
	accessTokenExpirationMinutes  time.Duration
	refreshTokenExpirationMinutes time.Duration
	maxSessionsPerUser            int
}

// NewUserSessionManager creates a new UserSessionManager instance
func NewUserSessionManager(
	dbClient *db.Client,
	idp IdentityProvider,
	authenticator Authenticator,
	logger logger.Logger,
	accessTokenExpirationMinutes int,
	refreshTokenExpirationMinutes int,
	maxSessionsPerUser int,
) UserSessionManager {
	return &userSessionManager{
		dbClient:                      dbClient,
		idp:                           idp,
		authenticator:                 authenticator,
		logger:                        logger,
		accessTokenExpirationMinutes:  time.Duration(accessTokenExpirationMinutes) * time.Minute,
		refreshTokenExpirationMinutes: time.Duration(refreshTokenExpirationMinutes) * time.Minute,
		maxSessionsPerUser:            maxSessionsPerUser,
	}
}

// CreateSession creates a new user session and returns the access and refresh tokens
func (u *userSessionManager) CreateSession(ctx context.Context, token string, userAgent string) (*CreateSessionResponse, error) {
	// Verify the token and get user information
	caller, err := u.authenticator.Authenticate(ctx, token, false)
	if err != nil {
		return nil, errors.Wrap(err, "oidc token is invalid")
	}

	userCaller, ok := caller.(*UserCaller)
	if !ok {
		return nil, errors.New("invalid caller type", errors.WithErrorCode(errors.EUnauthorized))
	}

	// Create user session
	session := &models.UserSession{
		UserID:         userCaller.User.Metadata.ID,
		RefreshTokenID: uuid.New().String(), // Create uuid for the session
		Expiration:     time.Now().Add(u.refreshTokenExpirationMinutes).UTC(),
		UserAgent:      userAgent,
	}

	txContext, err := u.dbClient.Transactions.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if rollbackErr := u.dbClient.Transactions.RollbackTx(txContext); rollbackErr != nil {
			u.logger.WithContextFields(ctx).Errorf("failed to rollback transaction: %v", rollbackErr)
		}
	}()

	session, err = u.dbClient.UserSessions.CreateUserSession(txContext, session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create user session")
	}

	// Find the oldest session for the user and clean up if necessary
	if err = u.cleanupOldSessions(txContext, session.UserID); err != nil {
		return nil, errors.Wrap(err, "failed to cleanup old user sessions")
	}

	if err = u.dbClient.Transactions.CommitTx(txContext); err != nil {
		return nil, errors.Wrap(err, "failed to commit tx")
	}

	// Generate tokens
	accessToken, err := u.generateAccessToken(ctx, session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate access token for user session")
	}

	refreshToken, err := u.generateRefreshToken(ctx, session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate refresh token for user session")
	}

	csrfToken, err := u.generateCSRFToken(ctx, session)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate csrf token for user session")
	}

	u.logger.WithContextFields(ctx).Infow("New user session created",
		"user_id", userCaller.User.Metadata.ID,
		"session_id", session.Metadata.ID,
		"user_agent", userAgent,
	)

	return &CreateSessionResponse{
		AccessToken:       accessToken,
		RefreshToken:      refreshToken,
		CSRFToken:         csrfToken,
		SessionExpiration: session.Expiration,
	}, nil
}

// RefreshSession validates a refresh token and returns new tokens
func (u *userSessionManager) RefreshSession(ctx context.Context, refreshToken string) (*RefreshSessionResponse, error) {
	// Decode and validate the jwt refresh token
	verifyOutput, err := u.idp.VerifyToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.Wrap(err, "refresh token is invalid", errors.WithErrorCode(errors.EUnauthorized))
	}

	// Verify this is a session refresh token
	if tokenType, ok := verifyOutput.PrivateClaims["type"]; !ok || tokenType != refreshTokenType {
		return nil, errors.New("token is not a refresh token", errors.WithErrorCode(errors.EUnauthorized))
	}

	// At this point we have a valid token, now we can extract the fields to check if a session still exists
	sessionID := gid.FromGlobalID(verifyOutput.PrivateClaims[SessionIDClaim])

	// Get user session from db
	sessionResult, err := u.dbClient.UserSessions.GetUserSessions(ctx, &db.GetUserSessionsInput{
		Filter: &db.UserSessionFilter{
			UserSessionIDs: []string{sessionID},
			RefreshTokenID: ptr.String(verifyOutput.Token.JwtID()),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get user session: %w", err)
	}

	if len(sessionResult.UserSessions) == 0 {
		return nil, errors.New("no user session found for refresh token", errors.WithErrorCode(errors.EUnauthorized))
	}

	// Find session that is not expired
	var validSession *models.UserSession
	for _, s := range sessionResult.UserSessions {
		if !s.IsExpired() {
			validSession = &s
			break
		}
	}

	if validSession == nil {
		return nil, errors.New("no valid user session found for refresh token", errors.WithErrorCode(errors.EUnauthorized))
	}

	// Update session with new refresh token ID
	validSession.RefreshTokenID = uuid.New().String()

	updatedSession, err := u.dbClient.UserSessions.UpdateUserSession(ctx, validSession)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update user session")
	}

	// Generate new tokens
	accessToken, err := u.generateAccessToken(ctx, updatedSession)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate access token for user session")
	}

	newRefreshToken, err := u.generateRefreshToken(ctx, updatedSession)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate refresh token for user session")
	}

	return &RefreshSessionResponse{
		AccessToken:       accessToken,
		RefreshToken:      newRefreshToken,
		SessionExpiration: updatedSession.Expiration,
	}, nil
}

// InvalidateSession invalidates a user session using either access or refresh token
func (u *userSessionManager) InvalidateSession(ctx context.Context, accessToken, refreshToken string) error {
	var sessionID string

	// Check refresh token first if it's provided
	if refreshToken != "" {
		verifyOutput, err := u.idp.VerifyToken(ctx, refreshToken)
		if err != nil {
			// Check if the returned error is because the token is expired
			if goerrors.Is(err, jwt.ErrTokenExpired()) {
				// Token is expired, so we can consider the user logged out
				return nil
			}
			return errors.New("failed to verify refresh token", errors.WithErrorCode(errors.EUnauthorized))
		}

		// Verify this is a session refresh token
		if tokenType, ok := verifyOutput.PrivateClaims["type"]; !ok || tokenType != refreshTokenType {
			return errors.New("token is not a refresh token", errors.WithErrorCode(errors.EUnauthorized))
		}

		sessionID = verifyOutput.PrivateClaims[SessionIDClaim]
	} else if accessToken != "" {
		// Decode and validate the jwt access token
		verifyOutput, err := u.idp.VerifyToken(ctx, accessToken)
		if err != nil {
			return errors.New("failed to verify access token", errors.WithErrorCode(errors.EUnauthorized))
		}

		// Verify this is a session access token
		if tokenType, ok := verifyOutput.PrivateClaims["type"]; !ok || tokenType != UserSessionAccessTokenType {
			return errors.New("token is not an access token", errors.WithErrorCode(errors.EUnauthorized))
		}

		sessionID = verifyOutput.PrivateClaims[SessionIDClaim]
	} else {
		// No tokens provided
		return nil
	}

	// The session id should never be empty at this point because the token type is checked above so we
	// will return an internal server error if it is
	if sessionID == "" {
		return errors.New("token is missing %s claim", SessionIDClaim)
	}

	// Get user session from db
	session, err := u.dbClient.UserSessions.GetUserSessionByID(ctx, gid.FromGlobalID(sessionID))
	if err != nil {
		return errors.Wrap(err, "failed to get user sessions by session id")
	}

	if session == nil {
		// User is already logged out
		return nil
	}

	// Delete the session
	if err := u.dbClient.UserSessions.DeleteUserSession(ctx, session); err != nil {
		return errors.Wrap(err, "failed to delete user session")
	}

	return nil
}

// VerifyCSRFToken verifies that the CSRF token is valid and matches the specified session ID
func (u *userSessionManager) VerifyCSRFToken(ctx context.Context, requestSessionID string, csrfToken string) error {
	// Decode and validate the jwt access token
	verifyOutput, err := u.idp.VerifyToken(ctx, csrfToken)
	if err != nil {
		return errors.New("csrf token is invalid %v", err, errors.WithErrorCode(errors.EUnauthorized))
	}

	// Verify this is a csrf token
	if tokenType, ok := verifyOutput.PrivateClaims["type"]; !ok || tokenType != UserSessionCSRFTokenType {
		return errors.New("csrf token has an invalid type %s", tokenType, errors.WithErrorCode(errors.EUnauthorized))
	}

	// Verify session matches current session
	sessionID := verifyOutput.PrivateClaims[SessionIDClaim]
	if sessionID == "" {
		return errors.New("csrf token is missing session id claim", errors.WithErrorCode(errors.EUnauthorized))
	}

	if gid.ToGlobalID(types.UserSessionModelType, requestSessionID) != sessionID {
		return errors.New("csrf token session id does not match current session", errors.WithErrorCode(errors.EUnauthorized))
	}

	return nil
}

// cleanupOldSessions removes expired sessions and enforces session limits per user
func (u *userSessionManager) cleanupOldSessions(ctx context.Context, userID string) error {
	// Find the oldest session for the user
	sortBy := db.UserSessionSortableFieldExpirationAsc
	oldestSessionResponse, err := u.dbClient.UserSessions.GetUserSessions(ctx, &db.GetUserSessionsInput{
		Sort: &sortBy,
		Filter: &db.UserSessionFilter{
			UserID: &userID,
		},
		PaginationOptions: &pagination.Options{
			First: ptr.Int32(1),
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get user sessions")
	}

	if len(oldestSessionResponse.UserSessions) > 0 {
		oldestSession := &oldestSessionResponse.UserSessions[0]
		if oldestSession.IsExpired() || int(oldestSessionResponse.PageInfo.TotalCount) > u.maxSessionsPerUser {
			// Delete the oldest session if it is expired or if there are more than max number of sessions
			if err := u.dbClient.UserSessions.DeleteUserSession(ctx, oldestSession); err != nil {
				return fmt.Errorf("failed to delete oldest user session: %w", err)
			}
		}
	}

	return nil
}

// generateCSRFToken creates a CSRF token
func (u *userSessionManager) generateCSRFToken(ctx context.Context, session *models.UserSession) (string, error) {
	token, err := u.idp.GenerateToken(ctx, &TokenInput{
		Expiration: &session.Expiration,
		Subject:    gid.ToGlobalID(types.UserModelType, session.UserID),
		Audience:   tokenAudience,
		Claims: map[string]string{
			"type":         UserSessionCSRFTokenType,
			SessionIDClaim: session.GetGlobalID(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate csrf token: %w", err)
	}

	return string(token), nil
}

// generateAccessToken creates a short-lived access token
func (u *userSessionManager) generateAccessToken(ctx context.Context, session *models.UserSession) (string, error) {
	expiration := time.Now().Add(u.accessTokenExpirationMinutes).UTC()
	token, err := u.idp.GenerateToken(ctx, &TokenInput{
		Expiration: &expiration,
		Subject:    gid.ToGlobalID(types.UserModelType, session.UserID),
		Audience:   tokenAudience,
		Claims: map[string]string{
			"type":         UserSessionAccessTokenType,
			SessionIDClaim: session.GetGlobalID(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate access token: %w", err)
	}

	return string(token), nil
}

// generateRefreshToken creates a long-lived refresh token
func (u *userSessionManager) generateRefreshToken(ctx context.Context, session *models.UserSession) (string, error) {
	token, err := u.idp.GenerateToken(ctx, &TokenInput{
		Expiration: &session.Expiration,
		JwtID:      session.RefreshTokenID,
		Subject:    gid.ToGlobalID(types.UserModelType, session.UserID),
		Audience:   tokenAudience,
		Claims: map[string]string{
			"type":         refreshTokenType,
			SessionIDClaim: session.GetGlobalID(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return string(token), nil
}

// GetUserSessionAccessTokenCookieName returns the cookie name for user session access token cookie
func GetUserSessionAccessTokenCookieName(secure bool) string {
	if secure {
		return cookieHostPrefix + accessTokenCookieName
	}
	return accessTokenCookieName
}

// GetUserSessionRefreshTokenCookieName returns the cookie name for user session refresh token cookie
func GetUserSessionRefreshTokenCookieName(secure bool) string {
	if secure {
		return cookieHostPrefix + refreshTokenCookieName
	}
	return refreshTokenCookieName
}

// GetUserSessionCSRFTokenCookieName returns the cookie name for user session csrf token cookie
func GetUserSessionCSRFTokenCookieName() string {
	// CSRF cookie doesn't use the "__Host-" prefix because it needs to be accessed by the UI javascript for the double submit cookie pattern
	return csrfTokenCookieName
}

// GetRequestUserSessionID returns the user session ID for this request if it exists
func GetRequestUserSessionID(r *http.Request) (string, bool) {
	caller := GetCaller(r.Context())

	if caller == nil {
		return "", false
	}

	if userCaller, ok := caller.(*UserCaller); ok && userCaller.UserSessionID != nil {
		return *userCaller.UserSessionID, true
	}

	return "", false
}
