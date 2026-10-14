package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestAuthenticationMiddleware_DeprecatedIDPTokenWarning(t *testing.T) {
	testCases := []struct {
		name        string
		caller      auth.Caller
		expectWarns bool
	}{
		{
			name:        "user authenticated with an identity provider token",
			caller:      auth.NewUserCaller(&models.User{Email: "user@example.com"}, nil, nil, nil, nil),
			expectWarns: true,
		},
		{
			name:   "user authenticated with a session token",
			caller: auth.NewUserCaller(&models.User{Email: "user@example.com"}, nil, nil, nil, ptr.String("session-1")),
		},
		{
			name:   "caller other than a user",
			caller: &auth.SystemCaller{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			testLogger, logs := logger.NewForTest()

			authenticator := auth.NewMockAuthenticator(t)
			authenticator.On("Authenticate", mock.Anything, "the-token", true).Return(test.caller, nil)

			userSessionManager := auth.NewMockUserSessionManager(t)
			userSessionManager.On("GetUserSessionAccessTokenCookieName").Return("tharsis_access_token")

			var gotCaller auth.Caller
			handler := NewAuthenticationMiddleware(testLogger, authenticator, response.NewWriter(testLogger), userSessionManager)(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					gotCaller = auth.GetCaller(r.Context())
				}),
			)

			req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
			req.Header.Set("Authorization", "Bearer the-token")
			handler.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, test.caller, gotCaller)

			warnings := logs.FilterMessageSnippet("identity provider token is deprecated")
			if test.expectWarns {
				assert.Equal(t, 1, warnings.Len())
			} else {
				assert.Equal(t, 0, warnings.Len())
			}
		})
	}
}
