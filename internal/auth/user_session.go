package auth

//go:generate go tool mockery --name UserSessionManager --inpackage --case underscore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

// ErrSessionAlreadyExists is returned when trying to create a session while one already exists
var ErrSessionAlreadyExists = errors.New("an active session already exists for this user", errors.WithErrorCode(errors.EConflict))

// ErrInvalidLoginCredentials is returned when the provided login credentials are invalid
var ErrInvalidLoginCredentials = errors.New("invalid username or password", errors.WithErrorCode(errors.EUnauthorized))

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
	cookieHostPrefix           = "__Host-"
	emptyCookieValue           = ""
	maxRetryAttemptsOnOLEError = 5
	validRedirectURL           = "http://localhost"
	validCodeChallengeMethod   = "S256"
)

// CreateSessionInput is the input for creating a session
type CreateSessionInput struct {
	Token     *string
	Username  *string
	Password  *string
	UserAgent string
}

// CreateSessionResponse is the response for creating a session
type CreateSessionResponse struct {
	AccessToken  string
	RefreshToken string
	CSRFToken    string
	Session      *models.UserSession
}

// RefreshSessionResponse is the response for updating a session
type RefreshSessionResponse struct {
	AccessToken  string
	RefreshToken string
	Session      *models.UserSession
}

// ExchangeOAuthCodeForSessionTokenInput is the input for exchanging an OAuth code for a session token
type ExchangeOAuthCodeForSessionTokenInput struct {
	OAuthCode         string
	OAuthCodeVerifier string
	RedirectURI       string
}

// ExchangeOAuthCodeForSessionTokenResponse is the response for exchanging an OAuth code for a session token
type ExchangeOAuthCodeForSessionTokenResponse struct {
	AccessToken string
	ExpiresIn   int
}

// SetUserSessionCookiesInput is the input for setting user session cookies
type SetUserSessionCookiesInput struct {
	AccessToken       string
	RefreshToken      string
	CsrfToken         *string
	SessionExpiration time.Time
}

// InitiateSessionOauthCodeFlowInput is the input for initiating an OAuth code flow
type InitiateSessionOauthCodeFlowInput struct {
	CodeChallenge       string
	CodeChallengeMethod string
	RedirectURI         string
	UserSessionID       string
}

// UserSessionManager interface defines the operations for managing user sessions
type UserSessionManager interface {
	GetCurrentSession(ctx context.Context) (*models.UserSession, error)
	CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionResponse, error)
	RefreshSession(ctx context.Context, refreshToken string) (*RefreshSessionResponse, error)
	InvalidateSession(ctx context.Context, accessToken, refreshToken string) error
	VerifyCSRFToken(ctx context.Context, requestSessionID string, csrfToken string) error
	ExchangeOAuthCodeForSessionToken(ctx context.Context, input *ExchangeOAuthCodeForSessionTokenInput) (*ExchangeOAuthCodeForSessionTokenResponse, error)
	InitiateSessionOauthCodeFlow(ctx context.Context, input *InitiateSessionOauthCodeFlowInput) (string, error)
	SetUserSessionCookies(w http.ResponseWriter, input *SetUserSessionCookiesInput)
	ClearUserSessionCookies(w http.ResponseWriter)
	GetUserSessionAccessTokenCookieName() string
	GetUserSessionRefreshTokenCookieName() string
	GetUserSessionCSRFTokenCookieName() string
}

// userSessionManager implements the UserSessionManager interface
type userSessionManager struct {
	dbClient                      *db.Client
	signingKeyManager             SigningKeyManager
	authenticator                 Authenticator
	logger                        logger.Logger
	accessTokenExpirationMinutes  time.Duration
	refreshTokenExpirationMinutes time.Duration
	maxSessionsPerUser            int
	enableSecureCookies           bool
	tharsisUIDomain               string
	userCredentialLoginEnabled    bool
}

// NewUserSessionManager creates a new UserSessionManager instance
func NewUserSessionManager(
	dbClient *db.Client,
	signingKeyManager SigningKeyManager,
	authenticator Authenticator,
	logger logger.Logger,
	accessTokenExpirationMinutes int,
	refreshTokenExpirationMinutes int,
	maxSessionsPerUser int,
	tharsisAPIURL string,
	tharsisUIURL string,
	userCredentialLoginEnabled bool,
) (UserSessionManager, error) {
	parsedUIURL, err := url.Parse(tharsisUIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tharsis ui url: %v", err)
	}

	// Check if external url uses https to determine if secure cookies should be enabled
	enableSecureCookies := false
	if parsedAPIURL, err := url.Parse(tharsisAPIURL); err == nil && parsedAPIURL.Scheme == "https" {
		enableSecureCookies = true
	}

	return &userSessionManager{
		dbClient:                      dbClient,
		signingKeyManager:             signingKeyManager,
		authenticator:                 authenticator,
		logger:                        logger,
		accessTokenExpirationMinutes:  time.Duration(accessTokenExpirationMinutes) * time.Minute,
		refreshTokenExpirationMinutes: time.Duration(refreshTokenExpirationMinutes) * time.Minute,
		maxSessionsPerUser:            maxSessionsPerUser,
		enableSecureCookies:           enableSecureCookies,
		tharsisUIDomain:               parsedUIURL.Hostname(),
		userCredentialLoginEnabled:    userCredentialLoginEnabled,
	}, nil
}

func (u *userSessionManager) GetCurrentSession(ctx context.Context) (*models.UserSession, error) {
	if sessionID, ok := GetRequestUserSessionID(ctx); ok {
		session, err := u.dbClient.UserSessions.GetUserSessionByID(ctx, sessionID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get user session by id")
		}
		if session == nil || session.IsExpired() {
			return nil, nil
		}
		return session, nil
	}
	return nil, nil
}

func (u *userSessionManager) ExchangeOAuthCodeForSessionToken(ctx context.Context, input *ExchangeOAuthCodeForSessionTokenInput) (*ExchangeOAuthCodeForSessionTokenResponse, error) {
	if input.OAuthCode == "" || input.OAuthCodeVerifier == "" || input.RedirectURI == "" {
		return nil, errors.New("missing required fields in input", errors.WithErrorCode(errors.EInvalid))
	}

	var token string
	if err := u.dbClient.RetryOnOLE(ctx, func() error {
		// Find session using auth code
		sessions, err := u.dbClient.UserSessions.GetUserSessions(ctx, &db.GetUserSessionsInput{
			Filter: &db.UserSessionFilter{
				OAuthCode: &input.OAuthCode,
			},
		})
		if err != nil {
			return fmt.Errorf("error retrieving user session: %w", err)
		}

		if len(sessions.UserSessions) > 0 {
			session := sessions.UserSessions[0]

			if session.OAuthRedirectURI == nil || *session.OAuthRedirectURI != input.RedirectURI {
				return errors.New("invalid redirect uri", errors.WithErrorCode(errors.EUnauthorized))
			}

			if session.OAuthCodeExpiration == nil || time.Now().After(*session.OAuthCodeExpiration) {
				return errors.New("auth code has expired", errors.WithErrorCode(errors.EUnauthorized))
			}

			if session.IsExpired() {
				return errors.New("session has expired", errors.WithErrorCode(errors.EUnauthorized))
			}

			// Create sha 256 hash of code verifier
			hash := sha256.Sum256([]byte(input.OAuthCodeVerifier))
			// Perform base64url encoding without padding
			hashedVerifier := base64.RawURLEncoding.EncodeToString(hash[:])
			if session.OAuthCodeChallenge == nil || hashedVerifier != *session.OAuthCodeChallenge {
				return errors.New("code verifier does not match code challenge", errors.WithErrorCode(errors.EUnauthorized))
			}

			// Remove oauth code and related fields from session because it's a one-time use code
			session.OAuthCode = nil
			session.OAuthCodeChallenge = nil
			session.OAuthCodeChallengeMethod = nil
			session.OAuthCodeExpiration = nil
			session.OAuthRedirectURI = nil
			updatedSession, err := u.dbClient.UserSessions.UpdateUserSession(ctx, &session)
			if err != nil {
				return fmt.Errorf("error updating user session: %w", err)
			}

			token, err = u.generateAccessToken(ctx, updatedSession)
			if err != nil {
				return fmt.Errorf("error generating access token: %w", err)
			}
		}

		return nil
	}, db.WithRetryOnOLEAttempts(maxRetryAttemptsOnOLEError)); err != nil {
		return nil, err
	}

	if token != "" {
		return &ExchangeOAuthCodeForSessionTokenResponse{
			AccessToken: string(token),
			ExpiresIn:   int(u.accessTokenExpirationMinutes.Seconds()),
		}, nil
	}

	return nil, errors.New("no user session found for auth code", errors.WithErrorCode(errors.EUnauthorized))
}

func (u *userSessionManager) InitiateSessionOauthCodeFlow(ctx context.Context, input *InitiateSessionOauthCodeFlowInput) (string, error) {
	if input.RedirectURI != validRedirectURL && !strings.HasPrefix(input.RedirectURI, validRedirectURL+":") {
		return "", errors.New("invalid redirect uri %s, only redirects to %s are supported", input.RedirectURI, validRedirectURL, errors.WithErrorCode(errors.EInvalid))
	}

	if input.CodeChallengeMethod != validCodeChallengeMethod {
		return "", errors.New("invalid code challenge method", errors.WithErrorCode(errors.EInvalid))
	}

	// Generate a 32-byte random value for the authorization code
	codeBytes := make([]byte, 32)
	_, err := rand.Read(codeBytes)
	if err != nil {
		return "", errors.New("failed to generate auth code")
	}

	// Encode to URL-safe base64 without padding
	authCode := base64.RawURLEncoding.EncodeToString(codeBytes)

	if err := u.dbClient.RetryOnOLE(ctx, func() error {
		// Find the user session
		session, err := u.dbClient.UserSessions.GetUserSessionByID(ctx, input.UserSessionID)
		if err != nil {
			return errors.Wrap(err, "failed to get user session by id")
		}
		if session == nil || session.IsExpired() {
			return errors.New("user session not found or has expired", errors.WithErrorCode(errors.EUnauthorized))
		}

		session.OAuthCode = &authCode
		session.OAuthCodeChallenge = &input.CodeChallenge
		session.OAuthCodeChallengeMethod = &input.CodeChallengeMethod
		session.OAuthCodeExpiration = ptr.Time(time.Now().Add(time.Minute).UTC())
		session.OAuthRedirectURI = &input.RedirectURI

		_, err = u.dbClient.UserSessions.UpdateUserSession(ctx, session)
		if err != nil {
			return errors.Wrap(err, "failed to update user session with oauth code")
		}
		return nil
	}, db.WithRetryOnOLEAttempts(maxRetryAttemptsOnOLEError)); err != nil {
		return "", err
	}

	return authCode, nil
}

// CreateSession creates a new user session and returns the access and refresh tokens
func (u *userSessionManager) CreateSession(ctx context.Context, input *CreateSessionInput) (*CreateSessionResponse, error) {
	var user *models.User

	if input.Token != nil {
		// Verify the token and get user information
		caller, err := u.authenticator.Authenticate(ctx, *input.Token, false)
		if err != nil {
			return nil, errors.Wrap(err, "oidc token is invalid")
		}

		userCaller, ok := caller.(*UserCaller)
		if !ok {
			return nil, errors.New("invalid caller type", errors.WithErrorCode(errors.EUnauthorized))
		}

		user = userCaller.User
	} else if input.Username != nil && input.Password != nil {
		if !u.userCredentialLoginEnabled {
			return nil, errors.New("user credential login is disabled", errors.WithErrorCode(errors.EInvalid))
		}

		// User can login with either username or email
		usernameToSearch := *input.Username
		if strings.Contains(usernameToSearch, "@") {
			// Remove email domain if username is an email
			parts := strings.Split(usernameToSearch, "@")
			usernameToSearch = parts[0]
		}

		u, err := u.dbClient.Users.GetUserByTRN(ctx, types.UserModelType.BuildTRN(usernameToSearch))
		if err != nil {
			return nil, err
		}

		if u == nil {
			return nil, ErrInvalidLoginCredentials
		}

		// if an email was used to login, check if email matches
		if usernameToSearch != *input.Username && u.Email != *input.Username {
			return nil, ErrInvalidLoginCredentials
		}

		// Verify password
		if ok := u.VerifyPassword(*input.Password); !ok {
			return nil, ErrInvalidLoginCredentials
		}

		user = u
	} else {
		return nil, errors.New("either token or username and password must be provided", errors.WithErrorCode(errors.EInvalid))
	}

	// Check if active session already exists
	caller := GetCaller(ctx)
	if caller != nil {
		if userCaller, ok := caller.(*UserCaller); ok && userCaller.UserSessionID != nil && userCaller.User.Metadata.ID == user.Metadata.ID {
			return nil, ErrSessionAlreadyExists
		}
	}

	// Create user session
	session := &models.UserSession{
		UserID:         user.Metadata.ID,
		RefreshTokenID: uuid.New().String(), // Create uuid for the session
		Expiration:     time.Now().Add(u.refreshTokenExpirationMinutes).UTC(),
		UserAgent:      input.UserAgent,
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
		"user_id", user.Metadata.ID,
		"session_id", session.Metadata.ID,
		"user_agent", input.UserAgent,
	)

	return &CreateSessionResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		CSRFToken:    csrfToken,
		Session:      session,
	}, nil
}

// RefreshSession validates a refresh token and returns new tokens
func (u *userSessionManager) RefreshSession(ctx context.Context, refreshToken string) (*RefreshSessionResponse, error) {
	// Decode and validate the jwt refresh token
	verifyOutput, err := u.signingKeyManager.VerifyToken(ctx, refreshToken)
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
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		Session:      updatedSession,
	}, nil
}

// InvalidateSession invalidates a user session using either access or refresh token
func (u *userSessionManager) InvalidateSession(ctx context.Context, accessToken, refreshToken string) error {
	var sessionID string

	// Check refresh token first if it's provided
	if refreshToken != "" {
		verifyOutput, err := u.signingKeyManager.VerifyToken(ctx, refreshToken)
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
		verifyOutput, err := u.signingKeyManager.VerifyToken(ctx, accessToken)
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
	verifyOutput, err := u.signingKeyManager.VerifyToken(ctx, csrfToken)
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

// setTokenCookies sets the access and refresh token cookies
func (u *userSessionManager) SetUserSessionCookies(w http.ResponseWriter, input *SetUserSessionCookiesInput) {
	accessExpiration := time.Now().Add(u.accessTokenExpirationMinutes)

	http.SetCookie(w, &http.Cookie{
		Name:     u.GetUserSessionAccessTokenCookieName(),
		Value:    input.AccessToken,
		HttpOnly: true,
		Secure:   u.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  accessExpiration,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     u.GetUserSessionRefreshTokenCookieName(),
		Value:    input.RefreshToken,
		HttpOnly: true,
		Secure:   u.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  input.SessionExpiration,
	})

	if input.CsrfToken != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     u.GetUserSessionCSRFTokenCookieName(),
			Value:    *input.CsrfToken,
			HttpOnly: false, // http only is false here to support the double submit cookie for csrf token
			Secure:   u.enableSecureCookies,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
			Expires:  input.SessionExpiration,
			Domain:   u.tharsisUIDomain,
		})
	}
}

// clearTokenCookies clears the token cookies
func (u *userSessionManager) ClearUserSessionCookies(w http.ResponseWriter) {
	// Set cookies with empty value and past expiration date to clear them
	expiredTime := time.Now().Add(-24 * time.Hour)

	http.SetCookie(w, &http.Cookie{
		Name:     u.GetUserSessionAccessTokenCookieName(),
		Value:    emptyCookieValue,
		HttpOnly: true,
		Secure:   u.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     u.GetUserSessionRefreshTokenCookieName(),
		Value:    emptyCookieValue,
		HttpOnly: true,
		Secure:   u.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     u.GetUserSessionCSRFTokenCookieName(),
		Value:    emptyCookieValue,
		HttpOnly: false,
		Secure:   u.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
		Domain:   u.tharsisUIDomain,
	})
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
	token, err := u.signingKeyManager.GenerateToken(ctx, &TokenInput{
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
	token, err := u.signingKeyManager.GenerateToken(ctx, &TokenInput{
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
	token, err := u.signingKeyManager.GenerateToken(ctx, &TokenInput{
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
func (u *userSessionManager) GetUserSessionAccessTokenCookieName() string {
	if u.enableSecureCookies {
		return cookieHostPrefix + accessTokenCookieName
	}
	return accessTokenCookieName
}

// GetUserSessionRefreshTokenCookieName returns the cookie name for user session refresh token cookie
func (u *userSessionManager) GetUserSessionRefreshTokenCookieName() string {
	if u.enableSecureCookies {
		return cookieHostPrefix + refreshTokenCookieName
	}
	return refreshTokenCookieName
}

// GetUserSessionCSRFTokenCookieName returns the cookie name for user session csrf token cookie
func (u *userSessionManager) GetUserSessionCSRFTokenCookieName() string {
	// CSRF cookie doesn't use the "__Host-" prefix because it needs to be accessed by the UI javascript for the double submit cookie pattern
	return csrfTokenCookieName
}

// GetRequestUserSessionID returns the user session ID for this request if it exists
func GetRequestUserSessionID(ctx context.Context) (string, bool) {
	caller := GetCaller(ctx)

	if caller == nil {
		return "", false
	}

	if userCaller, ok := caller.(*UserCaller); ok && userCaller.UserSessionID != nil {
		return *userCaller.UserSessionID, true
	}

	return "", false
}
