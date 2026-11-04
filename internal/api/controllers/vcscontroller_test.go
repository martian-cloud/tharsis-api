package controllers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestVCSController_WebhookBodySize(t *testing.T) {
	payload := `{"ref":"refs/heads/main"}`
	// padded returns payload padded to size bytes with leading whitespace, which the decoder has to read.
	padded := func(size int) string {
		return strings.Repeat(" ", size-len(payload)) + payload
	}

	tests := []struct {
		name          string
		provider      models.VCSProviderType
		body          string
		contentLength bool
		expectStatus  int
	}{
		{
			name:          "a payload at the limit is accepted",
			provider:      models.GitHubProviderType,
			body:          padded(maxVCSWebhookBodySize),
			contentLength: true,
			expectStatus:  http.StatusOK,
		},
		{
			name:          "a payload over the limit with a Content-Length is too large",
			provider:      models.GitHubProviderType,
			body:          padded(maxVCSWebhookBodySize + 1),
			contentLength: true,
			expectStatus:  http.StatusRequestEntityTooLarge,
		},
		{
			// A chunked request is only found to be over the limit while the handler reads it.
			name:         "a GitHub payload over the limit without a Content-Length is too large",
			provider:     models.GitHubProviderType,
			body:         padded(maxVCSWebhookBodySize + 1),
			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:         "a GitLab payload over the limit without a Content-Length is too large",
			provider:     models.GitLabProviderType,
			body:         padded(maxVCSWebhookBodySize + 1),
			expectStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:         "a payload under the limit without a Content-Length is accepted",
			provider:     models.GitLabProviderType,
			body:         payload,
			expectStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			caller := auth.NewVCSWorkspaceLinkCaller(&models.VCSProvider{Type: test.provider}, &models.WorkspaceVCSProviderLink{}, nil, nil)
			authenticator := auth.NewMockAuthenticator(t)
			authenticator.On("Authenticate", mock.Anything, "token", false).Return(caller, nil).Maybe() // Not called when the Content-Length is rejected up front.

			// A reader that is not a *strings.Reader leaves the request without a Content-Length.
			var body io.Reader = strings.NewReader(test.body)
			if !test.contentLength {
				body = io.MultiReader(body)
			}
			req := httptest.NewRequest(http.MethodPost, "/vcs/events?token=token", body)

			rec := httptest.NewRecorder()
			testLogger, _ := logger.NewForTest()
			router := chi.NewRouter()
			NewVCSController(testLogger, response.NewWriter(testLogger), authenticator, &vcsWebhookStub{}).RegisterRoutes(router)
			router.ServeHTTP(rec, req)

			assert.Equal(t, test.expectStatus, rec.Code, rec.Body.String())
			if test.expectStatus != http.StatusOK {
				assertVCSJSONError(t, rec)
			}
		})
	}
}

// vcsWebhookStub accepts every webhook event. The VCS service has no mockery mock, so it embeds the
// interface: a call to any other method panics.
type vcsWebhookStub struct {
	vcs.Service
}

func (vcsWebhookStub) ProcessWebhookEvent(context.Context, *vcs.ProcessWebhookEventInput) error {
	return nil
}

// assertVCSJSONError checks that an error response is the standard JSON error body.
func assertVCSJSONError(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var body struct {
		Detail string `json:"detail"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body), "error body is not JSON: %q", rec.Body.String())
	assert.NotEmpty(t, body.Detail)
}
