package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
)

// Various constant values to make testing easier.
const (
	authorizationHeader = "Authorization"
	sampleValidToken    = types.BearerAuthPrefix + "an-access-token"
)

// roundTripFunc implements the RoundTripper interface.
type roundTripFunc func(r *http.Request) *http.Response

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r), nil
}

// newTestClient returns *http.Client with Transport replaced to avoid making real calls.
func newTestClient(fn roundTripFunc) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(fn),
	}
}

func TestDefaultAPIHostname(t *testing.T) {
	provider, err := New(context.TODO(), nil, nil, "")
	assert.Nil(t, err)

	assert.Equal(t, defaultAPIHostname, provider.DefaultAPIHostname())
}

func TestMergeRequestActionIsSupported(t *testing.T) {
	testCases := []struct {
		action          string
		name            string
		expectSupported bool
	}{
		{
			name:            "positive: action supported",
			action:          "opened",
			expectSupported: true,
		},
		{
			name:            "negative: action unsupported",
			action:          "closed",
			expectSupported: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := New(context.TODO(), nil, nil, "")
			assert.Nil(t, err)

			assert.Equal(t, test.expectSupported, provider.MergeRequestActionIsSupported(test.action))
		})
	}
}

func TestToVCSEventType(t *testing.T) {
	testCases := []struct {
		input              *types.ToVCSEventTypeInput
		name               string
		expectVCSEventType models.VCSEventType
	}{
		{
			name: "positive: branch push event type; expect models.BranchEventType",
			input: &types.ToVCSEventTypeInput{
				EventHeader: pushEvent,
				Ref:         "refs/heads/main",
			},
			expectVCSEventType: models.BranchEventType,
		},
		{
			name: "positive: tag event type; expect models.TagEventType",
			input: &types.ToVCSEventTypeInput{
				EventHeader: pushEvent,
				Ref:         "refs/tags/v0.1",
			},
			expectVCSEventType: models.TagEventType,
		},
		{
			name: "positive: merge request event type; expect models.TagEventType",
			input: &types.ToVCSEventTypeInput{
				EventHeader: pullRequestEvent,
				Ref:         "refs/heads/feature/branch",
			},
			expectVCSEventType: models.MergeRequestEventType,
		},
		{
			name: "negative: event type unsupported",
			input: &types.ToVCSEventTypeInput{
				EventHeader: "random",
				Ref:         "random",
			},
			expectVCSEventType: "",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := New(context.TODO(), nil, nil, "")
			assert.Nil(t, err)

			assert.Equal(t, test.expectVCSEventType, provider.ToVCSEventType(test.input))
		})
	}
}

func TestBuildOAuthAuthorizationURL(t *testing.T) {
	// URL should be sample for both test cases since GitHub
	// doesn't support read-only scopes.
	expectedAuthorizationCodeURL := "https://github.com/login/oauth/authorize?client_id=an-oauth-client-id" +
		"&redirect_uri=https%3A%2F%2Ftharsis.domain%2Fv1%2Fvcs%2Fauth%2Fcallback" +
		"&scope=repo+read%3Auser&state=an-oauth-state"

	testCases := []struct {
		input       *types.BuildOAuthAuthorizationURLInput
		expectedURL string
		name        string
	}{
		{
			name: "positive: valid input with read-write scopes; expect authorization URL",
			input: &types.BuildOAuthAuthorizationURLInput{
				Hostname:           defaultAPIHostname,
				OAuthClientID:      "an-oauth-client-id",
				OAuthState:         "an-oauth-state",
				RedirectURL:        "https://tharsis.domain/v1/vcs/auth/callback",
				UseReadWriteScopes: true,
			},
			expectedURL: expectedAuthorizationCodeURL,
		},
		{
			name: "positive: valid input without read-write scopes; expect authorization URL",
			input: &types.BuildOAuthAuthorizationURLInput{
				Hostname:           defaultAPIHostname,
				OAuthClientID:      "an-oauth-client-id",
				OAuthState:         "an-oauth-state",
				RedirectURL:        "https://tharsis.domain/v1/vcs/auth/callback",
				UseReadWriteScopes: false,
			},
			expectedURL: expectedAuthorizationCodeURL,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := New(context.TODO(), nil, nil, "")
			assert.Nil(t, err)

			actualURL := provider.BuildOAuthAuthorizationURL(test.input)
			assert.NotEmpty(t, actualURL)
			assert.Equal(t, test.expectedURL, actualURL)
		})
	}
}

func TestBuildRepositoryURL(t *testing.T) {
	provider, err := New(context.TODO(), nil, nil, "")
	assert.Nil(t, err)

	repositoryURL := provider.BuildRepositoryURL(&types.BuildRepositoryURLInput{
		Hostname:       defaultAPIHostname,
		RepositoryPath: "owner/repository",
	})

	expectedURL := "https://github.com/owner/repository"

	assert.Equal(t, expectedURL, repositoryURL)
}

func TestTestConnection(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError error
		input         *types.TestConnectionInput
		name          string
	}{
		{
			name: "positive: token and hostname are valid; expect no errors",
			input: &types.TestConnectionInput{
				Hostname:    defaultAPIHostname,
				AccessToken: "an-access-token",
			},
		},
		{
			name: "negative: token or hostname is invalid; expect error",
			input: &types.TestConnectionInput{
				Hostname:    defaultAPIHostname,
				AccessToken: "an-invalid-access-token",
			},
			expectedError: fmt.Errorf("failed to connect to VCS provider at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, "/user", r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       nil,
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			provider, err := New(ctx, nil, client, "")
			assert.Nil(t, err)

			err = provider.TestConnection(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError   error
		input           *types.GetProjectInput
		response        *getProjectResponse
		expectedPayload *types.GetProjectPayload
		name            string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.GetProjectInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
			},
			response: &getProjectResponse{
				DefaultBranch: "main",
			},
			expectedPayload: &types.GetProjectPayload{
				DefaultBranch: "main",
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.GetProjectInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
			},
			expectedError: fmt.Errorf("failed to query for project at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/repos/",
					test.input.RepositoryPath,
				)

				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				// Marshal the response payload.
				responsePayload, err := json.Marshal(test.response)
				assert.Nil(t, err)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(responsePayload)),
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "")
			assert.Nil(t, err)

			payload, err := provider.GetProject(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, payload)
				assert.Equal(t, test.expectedPayload.DefaultBranch, payload.DefaultBranch)
			}
		})
	}
}

func TestGetDiff(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError   error
		input           *types.GetDiffInput
		response        *getDiffsResponse
		expectedPayload *types.GetDiffsPayload
		name            string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.GetDiffInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				Ref:            "main",
			},
			response: &getDiffsResponse{
				Files: []struct {
					Filename string `json:"filename"`
				}{
					struct {
						Filename string "json:\"filename\""
					}{
						Filename: "file.txt", // The changed file(s).
					},
					{
						Filename: "file.txt",
					},
				},
			},
			expectedPayload: &types.GetDiffsPayload{
				AlteredFiles: map[string]struct{}{
					"file.txt": {}, // We should see the same file only once.
				},
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.GetDiffInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				Ref:            "feature/branch",
			},
			expectedError: fmt.Errorf("failed to get diff at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/repos",
					test.input.RepositoryPath,
					"commits",
					test.input.Ref,
				)
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				// Marshal the response payload.
				responsePayload, err := json.Marshal(test.response)
				assert.Nil(t, err)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(responsePayload)),
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "")
			assert.Nil(t, err)

			payload, err := provider.GetDiff(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, payload)
				assert.Equal(t, test.expectedPayload.AlteredFiles, payload.AlteredFiles)
			}
		})
	}
}

func TestGetDiffs(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError   error
		input           *types.GetDiffsInput
		response        *getDiffsResponse
		expectedPayload *types.GetDiffsPayload
		name            string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.GetDiffsInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				BaseRef:        "base-commit-id",
				HeadRef:        "head-commit-id",
			},
			response: &getDiffsResponse{
				Files: []struct {
					Filename string `json:"filename"`
				}{
					struct {
						Filename string "json:\"filename\""
					}{
						Filename: "file.txt", // The changed file(s).
					},
					{
						Filename: "file.txt",
					},
				},
			},
			expectedPayload: &types.GetDiffsPayload{
				AlteredFiles: map[string]struct{}{
					"file.txt": {}, // We should see the same file only once.
				},
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.GetDiffsInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				BaseRef:        "base-commit-id",
				HeadRef:        "head-commit-id",
			},
			expectedError: fmt.Errorf("failed to get diffs at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/repos",
					test.input.RepositoryPath,
					"compare",
					test.input.BaseRef+"..."+test.input.HeadRef,
				)
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				// Marshal the response payload.
				responsePayload, err := json.Marshal(test.response)
				assert.Nil(t, err)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(responsePayload)),
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "")
			assert.Nil(t, err)

			payload, err := provider.GetDiffs(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, payload)
				assert.Equal(t, test.expectedPayload.AlteredFiles, payload.AlteredFiles)
			}
		})
	}
}

func TestGetArchive(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError error
		input         *types.GetArchiveInput
		name          string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.GetArchiveInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				Ref:            "main", // Attempting to download main branch.
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.GetArchiveInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				Ref:            "feature/branch",
			},
			expectedError: fmt.Errorf("failed to get repository archive at hostname %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/",
					"repos",
					test.input.RepositoryPath,
					"tarball",
					test.input.Ref,
				)
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       nil, // Payload won't matter since it never uses it.
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "")
			assert.Nil(t, err)

			_, err = provider.GetArchive(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateAccessToken(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError   error
		input           *types.CreateAccessTokenInput
		response        *createAccessTokenResponse
		expectedPayload *types.AccessTokenPayload
		name            string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.CreateAccessTokenInput{
				Hostname:          defaultAPIHostname,
				ClientID:          "some-client-id",
				ClientSecret:      "some-client-secret",
				AuthorizationCode: "some-authorization-code",
				RedirectURI:       "https://tharsis.domain/v1/vcs/auth/callback",
				// Other fields aren't used for GitHub.
			},
			response: &createAccessTokenResponse{
				AccessToken: "some-access-token",
			},
			expectedPayload: &types.AccessTokenPayload{
				AccessToken: "some-access-token",
				// RefreshToken isn't used for GitHub.
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.CreateAccessTokenInput{
				Hostname:          defaultAPIHostname,
				ClientID:          "invalid",
				ClientSecret:      "invalid",
				AuthorizationCode: "invalid",
				RedirectURI:       "https://tharsis.domain/v1/vcs/auth/callback",
			},
			expectedError: fmt.Errorf("failed to create access token at hostname: %s. Response status: %s", defaultAPIHostname, "400"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/",
					"login",
					"oauth",
					"access_token",
				)

				// Host will be without 'api.' prefix.
				assert.Equal(t, strings.Trim(defaultAPIHostname, "api."), r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				// Parse the queries.
				queries, err := url.ParseQuery(r.URL.RawQuery)
				assert.Nil(t, err)

				// Validate the values.
				assert.Equal(t, test.input.ClientID, queries.Get("client_id"))
				assert.Equal(t, test.input.ClientSecret, queries.Get("client_secret"))
				assert.Equal(t, test.input.AuthorizationCode, queries.Get("code"))
				assert.Equal(t, test.input.RedirectURI, queries.Get("redirect_uri"))

				// Emulate malformed input.
				if queries.Get("client_id") == "invalid" {
					return &http.Response{
						StatusCode: http.StatusBadRequest,
						Body:       nil,
						Status:     "400",
						Header:     make(http.Header),
					}
				}

				// Marshal the response payload.
				responsePayload, err := json.Marshal(test.response)
				assert.Nil(t, err)

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(responsePayload)),
					Status:     "200",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "https://tharsis.domain")
			assert.Nil(t, err)

			payload, err := provider.CreateAccessToken(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, payload)
				assert.Equal(t, test.expectedPayload.AccessToken, payload.AccessToken)
			}
		})
	}
}

func TestCreateWebhook(t *testing.T) {
	ctx := context.Background()

	sampleRequestBody := &createWebhookBody{
		Name: "web",
		Config: map[string]interface{}{
			"url":          "https://tharsis.domain/v1/vcs/events?token=webhook-auth-token",
			"content_type": "json",
			"insecure_ssl": float64(0), // Marshalling will convert to float64.
		},
		Events: eventTypes,
		Active: true,
	}

	testCases := []struct {
		expectedError   error
		input           *types.CreateWebhookInput
		response        *createWebhookResponse
		expectedPayload *types.WebhookPayload
		name            string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.CreateWebhookInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookToken:   []byte("webhook-auth-token"),
			},
			response: &createWebhookResponse{
				ID: 50,
			},
			expectedPayload: &types.WebhookPayload{
				WebhookID: "50",
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.CreateWebhookInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				WebhookToken:   []byte("webhook-auth-token"),
			},
			expectedError: fmt.Errorf("failed to create webhook at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/repos",
					test.input.RepositoryPath,
					"hooks",
				)

				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				// Validate request body.
				var body createWebhookBody
				assert.Nil(t, json.NewDecoder(r.Body).Decode(&body))
				assert.Equal(t, sampleRequestBody.Active, body.Active)
				assert.Equal(t, sampleRequestBody.Events, body.Events)
				assert.Equal(t, sampleRequestBody.Name, body.Name)
				assert.Equal(t, sampleRequestBody.Config, body.Config)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				// Marshal the response payload.
				responsePayload, err := json.Marshal(test.response)
				assert.Nil(t, err)

				return &http.Response{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(bytes.NewBuffer(responsePayload)),
					Status:     "201",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "https://tharsis.domain")
			assert.Nil(t, err)

			payload, err := provider.CreateWebhook(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			} else {
				assert.NotNil(t, payload)
				assert.Equal(t, test.expectedPayload.WebhookID, payload.WebhookID)
			}
		})
	}
}

func TestDeleteWebhook(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError error
		input         *types.DeleteWebhookInput
		name          string
	}{
		{
			name: "positive: input is valid; expect no errors",
			input: &types.DeleteWebhookInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookID:      "50",
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.DeleteWebhookInput{
				Hostname:       defaultAPIHostname,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				WebhookID:      "50",
			},
			expectedError: fmt.Errorf("failed to delete webhook at hostname: %s. Response status: %s", defaultAPIHostname, "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/repos",
					test.input.RepositoryPath,
					"hooks",
					test.input.WebhookID,
				)

				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				if r.Header.Get(authorizationHeader) != sampleValidToken {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       nil,
						Status:     "401",
						Header:     make(http.Header),
					}
				}

				return &http.Response{
					StatusCode: http.StatusNoContent,
					Body:       nil,
					Status:     "204",
					Header:     make(http.Header),
				}
			})

			logger, _ := logger.NewForTest()
			provider, err := New(ctx, logger, client, "")
			assert.Nil(t, err)

			err = provider.DeleteWebhook(ctx, test.input)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
