package controllers

import (
	"encoding/json"
	goerrors "errors"
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

type createSessionRequest struct {
	Token    *string `json:"token"`
	Username *string `json:"username"`
	Password *string `json:"password"`
}

type userSessionController struct {
	respWriter         response.Writer
	userSessionManager auth.UserSessionManager
	csrfMiddleware     middleware.Handler
	logger             logger.Logger
}

// NewUserSessionController creates a new controller for user session management
func NewUserSessionController(
	respWriter response.Writer,
	userSessionManager auth.UserSessionManager,
	csrfMiddleware middleware.Handler,
	logger logger.Logger,
) Controller {
	return &userSessionController{
		respWriter:         respWriter,
		userSessionManager: userSessionManager,
		csrfMiddleware:     csrfMiddleware,
		logger:             logger,
	}
}

// RegisterRoutes adds login routes to the router
func (c *userSessionController) RegisterRoutes(router chi.Router) {
	router.Group(func(r chi.Router) {
		r.Post("/sessions", c.CreateSession)
	})

	router.Group(func(r chi.Router) {
		// The CSRF check will be enforced for these endpoints
		r.Use(c.csrfMiddleware)
		r.Post("/sessions/refresh", c.RefreshSession)
		r.Post("/sessions/logout", c.Logout)
	})
}

// CreateSession handles user login and session creation
func (c *userSessionController) CreateSession(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	createSessionInput := &auth.CreateSessionInput{
		UserAgent: r.Header.Get("User-Agent"),
		Token:     req.Token,
		Username:  req.Username,
		Password:  req.Password,
	}

	// Create session using the session manager
	tokens, err := c.userSessionManager.CreateSession(r.Context(), createSessionInput)
	if err != nil {
		if goerrors.Is(err, auth.ErrSessionAlreadyExists) {
			// If a user has multiple browser tabs open, they may end up sending concurrent create session requests; therefore,
			// if we already have a valid session from a previous create session request, we will return here without creating
			// a new session
			w.WriteHeader(http.StatusNoContent)
			return
		}

		c.logger.Infow("Invalid request to create new user session",
			"error", err,
			"user_agent", createSessionInput.UserAgent,
			"subject", ptr.ToString(auth.GetSubject(r.Context())),
		)
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Set cookies and respond
	c.userSessionManager.SetUserSessionCookies(w, &auth.SetUserSessionCookiesInput{
		AccessToken:       tokens.AccessToken,
		RefreshToken:      tokens.RefreshToken,
		CsrfToken:         &tokens.CSRFToken,
		SessionExpiration: tokens.Session.Expiration,
	})

	w.WriteHeader(http.StatusNoContent)
}

// RefreshSession handles session token refresh
func (c *userSessionController) RefreshSession(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshTokenCookie, err := r.Cookie(c.userSessionManager.GetUserSessionRefreshTokenCookieName())
	if err != nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.New("refresh token not found in cookie", errors.WithErrorCode(errors.EUnauthorized)))
		return
	}

	// Refresh session using the session manager
	tokens, err := c.userSessionManager.RefreshSession(r.Context(), refreshTokenCookie.Value)
	if err != nil {
		if errors.ErrorCode(err) == errors.EOptimisticLock {
			// For OLE errors, we will return without an error since an optimistic lock can occur if multiple browser
			// tabs send concurrent requests to renew a session.
			w.WriteHeader(http.StatusNoContent)
		} else {
			c.logger.Infow("Invalid request to refresh user session",
				"error", err,
				"subject", ptr.ToString(auth.GetSubject(r.Context())),
			)
			c.respWriter.RespondWithError(r.Context(), w, err)
		}
		return
	}

	// Set new cookies and respond
	c.userSessionManager.SetUserSessionCookies(w, &auth.SetUserSessionCookiesInput{
		AccessToken:       tokens.AccessToken,
		RefreshToken:      tokens.RefreshToken,
		SessionExpiration: tokens.Session.Expiration,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Logout handles user logout and session invalidation
func (c *userSessionController) Logout(w http.ResponseWriter, r *http.Request) {
	var accessToken, refreshToken string

	// Get tokens from cookies
	if accessTokenCookie, _ := r.Cookie(c.userSessionManager.GetUserSessionAccessTokenCookieName()); accessTokenCookie != nil {
		accessToken = accessTokenCookie.Value
	}
	if refreshTokenCookie, _ := r.Cookie(c.userSessionManager.GetUserSessionRefreshTokenCookieName()); refreshTokenCookie != nil {
		refreshToken = refreshTokenCookie.Value
	}

	// Invalidate session using the session manager
	if err := c.userSessionManager.InvalidateSession(r.Context(), accessToken, refreshToken); err != nil {
		c.logger.Infow("Invalid request to logout user session",
			"error", err,
			"subject", ptr.ToString(auth.GetSubject(r.Context())),
		)
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Clear cookies
	c.userSessionManager.ClearUserSessionCookies(w)
	w.WriteHeader(http.StatusNoContent)
}
