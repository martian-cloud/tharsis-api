package gitlab

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
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
			action:          "open",
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
				EventHeader: "Push Hook",
				// Ref is not used for GitLab.
			},
			expectVCSEventType: models.BranchEventType,
		},
		{
			name: "negative: event type unsupported",
			input: &types.ToVCSEventTypeInput{
				EventHeader: "random",
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
			expectedURL: "https://gitlab.com/oauth/authorize?client_id=an-oauth-client-id&" +
				"redirect_uri=https%3A%2F%2Ftharsis.domain%2Fv1%2Fvcs%2Fauth%2Fcallback&" +
				"response_type=code&scope=api+read_repository&state=an-oauth-state",
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
			expectedURL: "https://gitlab.com/oauth/authorize?client_id=an-oauth-client-id&" +
				"redirect_uri=https%3A%2F%2Ftharsis.domain%2Fv1%2Fvcs%2Fauth%2Fcallback&response_type=code&" +
				"scope=read_user+read_api&state=an-oauth-state",
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

	expectedURL := "https://gitlab.com/owner/repository"

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
				expectedPath := path.Join(
					"/",
					apiV4Endpoint,
					"user",
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
					"/",
					apiV4Endpoint,
					"projects",
					test.input.RepositoryPath,
				)

				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path) // Path will contain the unescaped version.

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
				Diffs: []struct {
					OldPath string `json:"old_path"`
					NewPath string `json:"new_path"`
				}{
					struct {
						OldPath string "json:\"old_path\""
						NewPath string "json:\"new_path\""
					}{
						OldPath: "file.txt",
						NewPath: "file.txt",
					},
					{
						OldPath: "other.txt",
						NewPath: "another.txt",
					},
				},
			},
			expectedPayload: &types.GetDiffsPayload{
				AlteredFiles: map[string]struct{}{
					"file.txt":    {}, // We should see the same file only once.
					"other.txt":   {},
					"another.txt": {},
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
					"/",
					apiV4Endpoint,
					"projects",
					test.input.RepositoryPath,
					"repository",
					"commits",
					test.input.Ref,
					"diff",
				)
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path) // Will contain unescaped path.

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
				Diffs: []struct {
					OldPath string `json:"old_path"`
					NewPath string `json:"new_path"`
				}{
					struct {
						OldPath string "json:\"old_path\""
						NewPath string "json:\"new_path\""
					}{
						OldPath: "file.txt",
						NewPath: "file.txt",
					},
					{
						OldPath: "other.txt",
						NewPath: "another.txt",
					},
				},
			},
			expectedPayload: &types.GetDiffsPayload{
				AlteredFiles: map[string]struct{}{
					"file.txt":    {}, // We should see the same file only once.
					"other.txt":   {},
					"another.txt": {},
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
					"/",
					apiV4Endpoint,
					"projects",
					test.input.RepositoryPath,
					"repository",
					"compare",
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
					apiV4Endpoint,
					"projects",
					test.input.RepositoryPath,
					"repository",
					"archive.tar.gz",
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
		grantTypeParam  string
		response        *createAccessTokenResponse
		expectedPayload *types.AccessTokenPayload
		name            string
	}{
		{
			name: "positive: valid input, creating a new token; expect no errors",
			input: &types.CreateAccessTokenInput{
				Hostname:          defaultAPIHostname,
				ClientID:          "some-client-id",
				ClientSecret:      "some-client-secret",
				AuthorizationCode: "some-authorization-code",
				RedirectURI:       "https://tharsis.domain/v1/vcs/auth/callback",
			},
			grantTypeParam: "authorization_code",
			response: &createAccessTokenResponse{
				AccessToken:  "some-access-token",
				RefreshToken: "some-refresh-token",
				ExpiresIn:    7200,
				CreatedAt:    1607635748,
			},
			expectedPayload: &types.AccessTokenPayload{
				AccessToken:  "some-access-token",
				RefreshToken: "some-refresh-token",
			},
		},
		{
			name: "positive: valid input, renewing an old token; expect no errors",
			input: &types.CreateAccessTokenInput{
				Hostname:     defaultAPIHostname,
				ClientID:     "some-client-id",
				ClientSecret: "some-client-secret",
				RedirectURI:  "https://tharsis.domain/v1/vcs/auth/callback",
				RefreshToken: "some-refresh-token", // Only present when renewing.
			},
			grantTypeParam: "refresh_token",
			response: &createAccessTokenResponse{
				AccessToken:  "some-access-token",
				RefreshToken: "some-refresh-token",
				ExpiresIn:    7200,
				CreatedAt:    1607635748,
			},
			expectedPayload: &types.AccessTokenPayload{
				AccessToken:  "some-access-token",
				RefreshToken: "some-refresh-token",
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
			grantTypeParam: "authorization_code",
			expectedError:  fmt.Errorf("failed to create access token at hostname: %s. Response status: %s", defaultAPIHostname, "400"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					"/",
					"oauth",
					"token",
				)

				// Host will be without 'api.' prefix.
				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				// Parse the queries.
				queries, err := url.ParseQuery(r.URL.RawQuery)
				assert.Nil(t, err)

				// Validate the values.
				assert.Equal(t, test.input.ClientID, queries.Get("client_id"))
				assert.Equal(t, test.input.ClientSecret, queries.Get("client_secret"))
				assert.Equal(t, test.input.AuthorizationCode, queries.Get("code"))
				assert.Equal(t, test.input.RefreshToken, queries.Get("refresh_token"))
				assert.Equal(t, test.grantTypeParam, queries.Get("grant_type"))
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

	sampleRequestBody := url.Values{}
	sampleRequestBody.Add("url", "https://tharsis.domain/v1/vcs/events")
	sampleRequestBody.Add("token", "webhook-auth-token")

	// Add webhook events.
	for _, event := range eventTypes {
		sampleRequestBody.Add(event, "true")
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
					"/",
					apiV4Endpoint,
					"projects",
					test.input.RepositoryPath,
					"hooks",
				)

				assert.Equal(t, defaultAPIHostname, r.URL.Host)
				assert.Equal(t, expectedPath, r.URL.Path)

				// Validate request body.
				assert.Nil(t, r.ParseForm())
				assert.Equal(t, sampleRequestBody, r.Form)

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
					"/",
					apiV4Endpoint,
					"projects",
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
