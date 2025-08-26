package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/middleware"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	emptyCookieValue = ""
)

type createSessionRequest struct {
	Token string `json:"token"`
}

type setTokenCookiesInput struct {
	accessToken       string
	refreshToken      string
	csrfToken         *string
	sessionExpiration time.Time
}

type userSessionController struct {
	respWriter                   response.Writer
	userSessionManager           auth.UserSessionManager
	enableSecureCookies          bool
	accessTokenExpirationMinutes time.Duration
	csrfMiddleware               middleware.Handler
	tharsisUIDomain              string
	logger                       logger.Logger
}

// NewUserSessionController creates a new controller for user session management
func NewUserSessionController(
	respWriter response.Writer,
	userSessionManager auth.UserSessionManager,
	accessTokenExpirationMinutes int,
	csrfMiddleware middleware.Handler,
	enableSecureCookies bool,
	tharsisUIDomain string,
	logger logger.Logger,
) Controller {
	return &userSessionController{
		respWriter:                   respWriter,
		userSessionManager:           userSessionManager,
		enableSecureCookies:          enableSecureCookies,
		accessTokenExpirationMinutes: time.Duration(accessTokenExpirationMinutes) * time.Minute,
		csrfMiddleware:               csrfMiddleware,
		tharsisUIDomain:              tharsisUIDomain,
		logger:                       logger,
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
	// If a user has multiple browser tabs open, they may end up sending concurrent create session requests; therefore,
	// if we already have a valid session from a previous create session request, we will return here without creating
	// a new session
	if requestSessionID, ok := auth.GetRequestUserSessionID(r); ok {
		// Verify that CSRF token is valid and matches session ID
		csrfTokenCookie, _ := r.Cookie(auth.GetUserSessionCSRFTokenCookieName())
		if csrfTokenCookie != nil && c.userSessionManager.VerifyCSRFToken(r.Context(), requestSessionID, csrfTokenCookie.Value) == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	// Parse the request body
	var req createSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		c.respWriter.RespondWithError(r.Context(), w, errors.Wrap(err, "failed to parse request body", errors.WithErrorCode(errors.EInvalid)))
		return
	}

	userAgent := r.Header.Get("User-Agent")

	// Create session using the session manager
	tokens, err := c.userSessionManager.CreateSession(r.Context(), req.Token, userAgent)
	if err != nil {
		c.logger.Infow("Invalid request to create new user session",
			"error", err,
			"user_agent", userAgent,
			"subject", ptr.ToString(auth.GetSubject(r.Context())),
		)
		c.respWriter.RespondWithError(r.Context(), w, err)
		return
	}

	// Set cookies and respond
	c.setTokenCookies(w, &setTokenCookiesInput{
		accessToken:       tokens.AccessToken,
		refreshToken:      tokens.RefreshToken,
		csrfToken:         &tokens.CSRFToken,
		sessionExpiration: tokens.SessionExpiration,
	})

	w.WriteHeader(http.StatusNoContent)
}

// RefreshSession handles session token refresh
func (c *userSessionController) RefreshSession(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshTokenCookie, err := r.Cookie(auth.GetUserSessionRefreshTokenCookieName(c.enableSecureCookies))
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
	c.setTokenCookies(w, &setTokenCookiesInput{
		accessToken:       tokens.AccessToken,
		refreshToken:      tokens.RefreshToken,
		sessionExpiration: tokens.SessionExpiration,
	})
	w.WriteHeader(http.StatusNoContent)
}

// Logout handles user logout and session invalidation
func (c *userSessionController) Logout(w http.ResponseWriter, r *http.Request) {
	var accessToken, refreshToken string

	// Get tokens from cookies
	if accessTokenCookie, _ := r.Cookie(auth.GetUserSessionAccessTokenCookieName(c.enableSecureCookies)); accessTokenCookie != nil {
		accessToken = accessTokenCookie.Value
	}
	if refreshTokenCookie, _ := r.Cookie(auth.GetUserSessionRefreshTokenCookieName(c.enableSecureCookies)); refreshTokenCookie != nil {
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
	c.clearTokenCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

// setTokenCookies sets the access and refresh token cookies
func (c *userSessionController) setTokenCookies(w http.ResponseWriter, input *setTokenCookiesInput) {
	accessExpiration := time.Now().Add(c.accessTokenExpirationMinutes)

	http.SetCookie(w, &http.Cookie{
		Name:     auth.GetUserSessionAccessTokenCookieName(c.enableSecureCookies),
		Value:    input.accessToken,
		HttpOnly: true,
		Secure:   c.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  accessExpiration,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     auth.GetUserSessionRefreshTokenCookieName(c.enableSecureCookies),
		Value:    input.refreshToken,
		HttpOnly: true,
		Secure:   c.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  input.sessionExpiration,
	})

	if input.csrfToken != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     auth.GetUserSessionCSRFTokenCookieName(),
			Value:    *input.csrfToken,
			HttpOnly: false, // http only is false here to support the double submit cookie for csrf token
			Secure:   c.enableSecureCookies,
			SameSite: http.SameSiteStrictMode,
			Path:     "/",
			Expires:  input.sessionExpiration,
			Domain:   c.tharsisUIDomain,
		})
	}
}

// clearTokenCookies clears the token cookies
func (c *userSessionController) clearTokenCookies(w http.ResponseWriter) {
	// Set cookies with empty value and past expiration date to clear them
	expiredTime := time.Now().Add(-24 * time.Hour)

	http.SetCookie(w, &http.Cookie{
		Name:     auth.GetUserSessionAccessTokenCookieName(c.enableSecureCookies),
		Value:    emptyCookieValue,
		HttpOnly: true,
		Secure:   c.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     auth.GetUserSessionRefreshTokenCookieName(c.enableSecureCookies),
		Value:    emptyCookieValue,
		HttpOnly: true,
		Secure:   c.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     auth.GetUserSessionCSRFTokenCookieName(),
		Value:    emptyCookieValue,
		HttpOnly: false,
		Secure:   c.enableSecureCookies,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Expires:  expiredTime,
		Domain:   c.tharsisUIDomain,
	})
}
