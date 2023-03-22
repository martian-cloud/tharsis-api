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

// customProviderURL is a url.URL for a custom provider instance.
var customProviderURL = url.URL{Scheme: "https", Host: "example.com", Path: "/instances/github"}

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

func TestDefaultURL(t *testing.T) {
	provider, err := New(context.TODO(), nil, nil, "")
	assert.Nil(t, err)

	assert.Equal(t, defaultURL, provider.DefaultURL())
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
				ProviderURL:        defaultURL,
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
				ProviderURL:        defaultURL,
				OAuthClientID:      "an-oauth-client-id",
				OAuthState:         "an-oauth-state",
				RedirectURL:        "https://tharsis.domain/v1/vcs/auth/callback",
				UseReadWriteScopes: false,
			},
			expectedURL: expectedAuthorizationCodeURL,
		},
		{
			name: "positive: valid input with custom provider URL; expect authorization URL",
			input: &types.BuildOAuthAuthorizationURLInput{
				ProviderURL:        customProviderURL, // Theoretical GitHub instance hosted under a path.
				OAuthClientID:      "an-oauth-client-id",
				OAuthState:         "an-oauth-state",
				RedirectURL:        "https://tharsis.domain/v1/vcs/auth/callback",
				UseReadWriteScopes: true,
			},
			expectedURL: customProviderURL.String() + "/login/oauth/authorize?client_id=an-oauth-client-id" +
				"&redirect_uri=https%3A%2F%2Ftharsis.domain%2Fv1%2Fvcs%2Fauth%2Fcallback" +
				"&scope=repo+read%3Auser&state=an-oauth-state",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := New(context.TODO(), nil, nil, "")
			assert.Nil(t, err)

			actualURL, err := provider.BuildOAuthAuthorizationURL(test.input)
			assert.Nil(t, err)
			assert.Equal(t, test.expectedURL, actualURL)
		})
	}
}

func TestBuildRepositoryURL(t *testing.T) {
	repositoryPath := "/owner/repository"

	testCases := []struct {
		name        string
		input       *types.BuildRepositoryURLInput
		expectedURL string
	}{
		{
			name: "repository URL with provider's default API URL",
			input: &types.BuildRepositoryURLInput{
				ProviderURL:    defaultURL,
				RepositoryPath: repositoryPath,
			},
			expectedURL: "https://github.com" + repositoryPath, // "api." prefix should be stripped.
		},
		{
			name: "repository URL with custom API URL",
			input: &types.BuildRepositoryURLInput{
				ProviderURL:    customProviderURL,
				RepositoryPath: repositoryPath,
			},
			expectedURL: customProviderURL.String() + repositoryPath,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			provider, err := New(context.TODO(), nil, nil, "")
			assert.Nil(t, err)

			actualURL, err := provider.BuildRepositoryURL(test.input)
			assert.Nil(t, err)
			assert.Equal(t, test.expectedURL, actualURL)
		})
	}
}

func TestTestConnection(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		expectedError error
		input         *types.TestConnectionInput
		name          string
	}{
		{
			name: "positive: token and URL are valid; expect no errors",
			input: &types.TestConnectionInput{
				ProviderURL: defaultURL,
				AccessToken: "an-access-token",
			},
		},
		{
			name: "positive: token and URL are valid for a custom instance; expect no errors",
			input: &types.TestConnectionInput{
				ProviderURL: customProviderURL,
				AccessToken: "an-access-token",
			},
		},
		{
			name: "negative: token or URL is invalid; expect error",
			input: &types.TestConnectionInput{
				ProviderURL: defaultURL,
				AccessToken: "an-invalid-access-token",
			},
			expectedError: fmt.Errorf("failed to connect to VCS provider. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
				assert.Equal(t, test.input.ProviderURL.Path+"/user", r.URL.Path)

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
				ProviderURL:    defaultURL,
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
			name: "positive: input is valid with custom github instance; expect no errors",
			input: &types.GetProjectInput{
				ProviderURL:    customProviderURL,
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
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
			},
			expectedError: fmt.Errorf("failed to query for project. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/repos/",
					test.input.RepositoryPath,
				)

				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
				ProviderURL:    defaultURL,
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
			name: "positive: input is valid with custom provider url; expect no errors",
			input: &types.GetDiffInput{
				ProviderURL:    customProviderURL,
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
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				Ref:            "feature/branch",
			},
			expectedError: fmt.Errorf("failed to get diff. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/repos",
					test.input.RepositoryPath,
					"commits",
					test.input.Ref,
				)
				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
				ProviderURL:    defaultURL,
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
			name: "positive: input is valid with custom provider instance; expect no errors",
			input: &types.GetDiffsInput{
				ProviderURL:    customProviderURL,
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
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				BaseRef:        "base-commit-id",
				HeadRef:        "head-commit-id",
			},
			expectedError: fmt.Errorf("failed to get diffs. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/repos",
					test.input.RepositoryPath,
					"compare",
					test.input.BaseRef+"..."+test.input.HeadRef,
				)
				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
				ProviderURL:    defaultURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				Ref:            "main", // Attempting to download main branch.
			},
		},
		{
			name: "positive: input is valid with custom provider URL; expect no errors",
			input: &types.GetArchiveInput{
				ProviderURL:    customProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				Ref:            "main", // Attempting to download main branch.
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.GetArchiveInput{
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				Ref:            "feature/branch",
			},
			expectedError: fmt.Errorf("failed to get repository archive. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/",
					"repos",
					test.input.RepositoryPath,
					"tarball",
					test.input.Ref,
				)
				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
				ProviderURL:       defaultURL,
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
			name: "positive: input is valid with custom provider URL; expect no errors",
			input: &types.CreateAccessTokenInput{
				ProviderURL:       customProviderURL,
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
				ProviderURL:       defaultURL,
				ClientID:          "invalid",
				ClientSecret:      "invalid",
				AuthorizationCode: "invalid",
				RedirectURI:       "https://tharsis.domain/v1/vcs/auth/callback",
			},
			expectedError: fmt.Errorf("failed to create access token. Response status: %s", "400"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/",
					"login",
					"oauth",
					"access_token",
				)

				// Host will be without 'api.' prefix.
				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, strings.TrimPrefix(test.input.ProviderURL.Host, "api."), r.URL.Host)
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
				ProviderURL:    defaultURL,
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
			name: "positive: input is valid with custom provider URL; expect no errors",
			input: &types.CreateWebhookInput{
				ProviderURL:    customProviderURL,
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
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				WebhookToken:   []byte("webhook-auth-token"),
			},
			expectedError: fmt.Errorf("failed to create webhook. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/repos",
					test.input.RepositoryPath,
					"hooks",
				)

				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
				ProviderURL:    defaultURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookID:      "50",
			},
		},
		{
			name: "positive: input is valid with custom provider instance; expect no errors",
			input: &types.DeleteWebhookInput{
				ProviderURL:    customProviderURL,
				AccessToken:    "an-access-token",
				RepositoryPath: "owner/repository",
				WebhookID:      "50",
			},
		},
		{
			name: "negative: input is invalid; expect error",
			input: &types.DeleteWebhookInput{
				ProviderURL:    defaultURL,
				AccessToken:    "some-token",
				RepositoryPath: "owner/repo",
				WebhookID:      "50",
			},
			expectedError: fmt.Errorf("failed to delete webhook. Response status: %s", "401"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			client := newTestClient(func(r *http.Request) *http.Response {
				expectedPath := path.Join(
					test.input.ProviderURL.Path,
					"/repos",
					test.input.RepositoryPath,
					"hooks",
					test.input.WebhookID,
				)

				assert.Equal(t, test.input.ProviderURL.Scheme, r.URL.Scheme)
				assert.Equal(t, test.input.ProviderURL.Host, r.URL.Host)
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
