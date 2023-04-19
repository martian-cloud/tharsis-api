// Package gitlab package
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// V4 API endpoint for GitLab.
	apiV4Endpoint = "api/v4"

	// gitLabReadWriteOAuthScopes represents space-separated OAuth scopes that are requested
	// from the GitLab VCS provider. Passed in as 'scope' query parameter.
	// https://docs.gitlab.com/ee/integration/oauth_provider.html#authorized-applications
	gitLabReadWriteOAuthScopes = "api read_repository"

	// gitLabReadOnlyOAuthScopes is similar to above expect it requests read-only permissions.
	// These base permissions are needed when webhooks aren't being used in order to validate
	// the access token, download a repository tarball among other API interactions.
	gitLabReadOnlyOAuthScopes = "read_user read_api"
)

var (
	// Event type that get registered with GitLab. These are
	// specific to configuring webhooks and cannot be used
	// to determine the event type on a webhook payload.
	eventTypes = []string{
		"push_events",
		"tag_push_events",
		"merge_requests_events",
	}

	// supportedGitLabMRActions contains the list of actions
	// for a merge request that can trigger a run.
	supportedGitLabMRActions = map[string]struct{}{
		"open":   {}, // When a MR is opened.
		"update": {}, // When a MR is updated.
	}

	// supportedGitLabEvents contains events that are supported
	// for the GitLab VCS provider. The header 'X-Gitlab-Event'
	// is checked to make sure the event is supported.
	supportedGitLabEvents = map[string]models.VCSEventType{
		"Push Hook":          models.BranchEventType,
		"Tag Push Hook":      models.TagEventType,
		"Merge Request Hook": models.MergeRequestEventType,
	}

	// defaultURL is the default API URL for this provider type.
	defaultURL = url.URL{
		Scheme: "https",
		Host:   "gitlab.com",
	}
)

// createAccessTokenResponse is the response struct for creating an access token.
type createAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	CreatedAt    int64  `json:"created_at"`
}

// getProjectResponse is the response struct for retrieving a project.
type getProjectResponse struct {
	DefaultBranch string `json:"default_branch"`
}

// getDiffsResponse is the response struct for retrieving diffs.
type getDiffsResponse struct {
	Diffs []struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	} `json:"diffs"`
}

// getDiffResponse is the response struct for retrieving diff.
type getDiffResponse struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
}

// createWebhookResponse is the response struct for creating
// a webhook in GitLab.
type createWebhookResponse struct {
	ID int `json:"id"`
}

// Provider represents a particular VCS provider.
type Provider struct {
	logger     logger.Logger
	client     *http.Client
	tharsisURL string
}

// New creates a new Provider instance.
func New(
	_ context.Context,
	logger logger.Logger,
	client *http.Client,
	tharsisURL string,
) (*Provider, error) {
	return &Provider{
		logger,
		client,
		tharsisURL,
	}, nil
}

// DefaultURL returns the default API URL for this provider.
func (p *Provider) DefaultURL() url.URL {
	return defaultURL
}

// MergeRequestActionIsSupported returns true if the merge request action is supported.
func (p *Provider) MergeRequestActionIsSupported(action string) bool {
	_, ok := supportedGitLabMRActions[action]
	return ok
}

// ToVCSEventType determines whether the event is supported
// and translates the event type to VCSEventType equivalent.
func (p *Provider) ToVCSEventType(input *types.ToVCSEventTypeInput) models.VCSEventType {
	return supportedGitLabEvents[input.EventHeader]
}

// BuildOAuthAuthorizationURL build the authorization code URL which is
// used to redirect the user to the VCS provider to complete OAuth flow.
func (p *Provider) BuildOAuthAuthorizationURL(input *types.BuildOAuthAuthorizationURLInput) (string, error) {
	// Use appropriate scopes.
	scopes := gitLabReadOnlyOAuthScopes
	if input.UseReadWriteScopes {
		scopes = gitLabReadWriteOAuthScopes
	}

	queries := input.ProviderURL.Query()
	queries.Add("client_id", input.OAuthClientID)
	queries.Add("redirect_uri", input.RedirectURL)
	queries.Add("response_type", "code")
	queries.Add("state", input.OAuthState)
	queries.Add("scope", scopes)
	input.ProviderURL.RawQuery = queries.Encode()

	return url.JoinPath(input.ProviderURL.String(), "oauth/authorize")
}

// BuildRepositoryURL returns the repository URL associated with the provider.
func (p *Provider) BuildRepositoryURL(input *types.BuildRepositoryURLInput) (string, error) {
	return url.JoinPath(input.ProviderURL.String(), input.RepositoryPath)
}

// TestConnection simply queries for the user metadata that's
// associated with the access token to verify validity.
// https://docs.gitlab.com/ee/api/users.html#for-normal-users-1
func (p *Provider) TestConnection(ctx context.Context, input *types.TestConnectionInput) error {
	endpoint, err := url.JoinPath(input.ProviderURL.String(), apiV4Endpoint, "user")
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to connect to VCS provider. Response status: %s", resp.Status)
	}

	return nil
}

// GetProject retrieves information about a project or repository.
// https://docs.gitlab.com/ee/api/projects.html#get-single-project
func (p *Provider) GetProject(ctx context.Context, input *types.GetProjectInput) (*types.GetProjectPayload, error) {
	// Build the request URL.
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
	}, "/")

	queries := input.ProviderURL.Query()
	queries.Add("statistics", "true") // 'statistics' contains information about repo size.
	input.ProviderURL.RawQuery = queries.Encode()

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query for project. Response status: %s", resp.Status)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Errorf("failed to close response body in GetProject: %v", err)
		}
	}()

	// Unmarshal the response.
	var project getProjectResponse
	if err = json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	return &types.GetProjectPayload{
		DefaultBranch: project.DefaultBranch,
	}, nil
}

// GetDiff gets the diff for a single ref (branch, tag, commit, etc.).
// https://docs.gitlab.com/ee/api/commits.html#get-the-diff-of-a-commit
func (p *Provider) GetDiff(ctx context.Context, input *types.GetDiffInput) (*types.GetDiffsPayload, error) {
	// Build the request URL.
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
		"repository",
		"commits",
		url.PathEscape(input.Ref),
		"diff",
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get diff. Response status: %s", resp.Status)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Errorf("failed to close response body in GetDiff: %v", err)
		}
	}()

	diffResp := []getDiffResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&diffResp); err != nil {
		return nil, err
	}

	return &types.GetDiffsPayload{
		AlteredFiles: createChangesMap(nil, diffResp),
	}, nil
}

// GetDiffs retrieves diffs for two different refs (branches, commits, etc.)
// https://docs.gitlab.com/ee/api/repositories.html#compare-branches-tags-or-commits
func (p *Provider) GetDiffs(ctx context.Context, input *types.GetDiffsInput) (*types.GetDiffsPayload, error) {
	// Build the request URL.
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
		"repository",
		"compare",
	}, "/")

	// Add queries.
	queries := input.ProviderURL.Query()
	queries.Add("from", input.BaseRef)
	queries.Add("to", input.HeadRef)
	input.ProviderURL.RawQuery = queries.Encode()

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get diffs. Response status: %s", resp.Status)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Errorf("failed to close response body in GetDiffs: %v", err)
		}
	}()

	diffResp := getDiffsResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&diffResp); err != nil {
		return nil, err
	}

	return &types.GetDiffsPayload{
		AlteredFiles: createChangesMap(&diffResp, nil),
	}, nil
}

// GetArchive downloads the entire repository archive for a branch or tag.
// https://docs.gitlab.com/ee/api/repositories.html#get-file-archive
func (p *Provider) GetArchive(ctx context.Context, input *types.GetArchiveInput) (*http.Response, error) {
	// Build the request URL.
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
		"repository",
		"archive.tar.gz", // Default is tar.gz, but incase it changes.
	}, "/")

	// Add queries.
	queries := input.ProviderURL.Query()
	queries.Add("sha", input.Ref)
	input.ProviderURL.RawQuery = queries.Encode()

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", "application/octet-stream")
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get repository archive. Response status: %s", resp.Status)
	}

	return resp, nil
}

// CreateAccessToken sends a POST request to the provider to create
// an access and refresh tokens that can be used to further interact
// with the provider's API.
// https://docs.gitlab.com/ee/api/oauth2.html#authorization-code-flow
func (p *Provider) CreateAccessToken(ctx context.Context, input *types.CreateAccessTokenInput) (*types.AccessTokenPayload, error) {
	path := strings.Join([]string{
		"oauth",
		"token",
	}, "/")

	// Add queries.
	queries := input.ProviderURL.Query()
	queries.Add("client_id", input.ClientID)
	queries.Add("client_secret", input.ClientSecret)
	queries.Add("redirect_uri", input.RedirectURI)

	// Add appropriate params for renewing an access token.
	if input.RefreshToken != "" {
		queries.Add("refresh_token", input.RefreshToken)
		queries.Add("grant_type", "refresh_token")
	} else {
		queries.Add("code", input.AuthorizationCode)
		queries.Add("grant_type", "authorization_code")
	}
	input.ProviderURL.RawQuery = queries.Encode()

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create access token. Response status: %s", resp.Status)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Errorf("failed to close response body in CreateAccessToken: %v", err)
		}
	}()

	tokenResp := createAccessTokenResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	// Parse timestamps.
	createdAtUnix := time.Unix(tokenResp.CreatedAt, 0)
	expiresAtDuration := time.Duration(tokenResp.ExpiresIn) // GitLab's expiration is 7200s.
	expirationTimestamp := createdAtUnix.Add(time.Second * expiresAtDuration)

	return &types.AccessTokenPayload{
		AccessToken:         tokenResp.AccessToken,
		RefreshToken:        tokenResp.RefreshToken,
		ExpirationTimestamp: &expirationTimestamp,
	}, nil
}

// CreateWebhook creates a webhook at the specified provider.
// Returns the webhook ID from the response.
// https://docs.gitlab.com/ee/api/projects.html#add-project-hook
func (p *Provider) CreateWebhook(ctx context.Context, input *types.CreateWebhookInput) (*types.WebhookPayload, error) {
	// Build the request URL.
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
		"hooks",
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return nil, err
	}

	// Build Tharsis webhook endpoint.
	parsedURL, err := url.Parse(p.tharsisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Tharsis URL: %v", err)
	}
	parsedURL.Path = types.V1WebhookEndpoint

	// Add the webhook event types to body form.
	form := url.Values{}
	for _, event := range eventTypes {
		form.Add(event, "true")
	}

	// Add the Tharsis URL and token.
	form.Add("url", parsedURL.String())
	form.Add("token", string(input.WebhookToken))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Content-Type", types.FormContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create webhook. Response status: %s", resp.Status)
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			p.logger.Errorf("failed to close response body in CreateWebhook: %v", err)
		}
	}()

	var webhookResponse createWebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResponse); err != nil {
		return nil, err
	}

	return &types.WebhookPayload{
		WebhookID: strconv.Itoa(webhookResponse.ID),
	}, nil
}

// DeleteWebhook deletes a webhook at the specified provider.
// https://docs.gitlab.com/ee/api/projects.html#delete-project-hook
func (p *Provider) DeleteWebhook(ctx context.Context, input *types.DeleteWebhookInput) error {
	// Build the request URL
	rawPath := strings.Join([]string{
		apiV4Endpoint,
		"projects",
		url.PathEscape(input.RepositoryPath),
		"hooks",
		input.WebhookID,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), rawPath)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add request headers.
	request.Header.Add("Accept", types.JSONContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	resp, err := p.client.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete webhook. Response status: %s", resp.Status)
	}

	return nil
}

// createChangesMap creates a unique map of files that have been altered.
func createChangesMap(diffsResp *getDiffsResponse, diffResp []getDiffResponse) map[string]struct{} {
	changesMap := map[string]struct{}{}

	if diffsResp != nil {
		for _, diff := range diffsResp.Diffs {
			if _, ok := changesMap[diff.OldPath]; !ok {
				changesMap[diff.OldPath] = struct{}{}
			}
			if _, ok := changesMap[diff.NewPath]; !ok {
				changesMap[diff.NewPath] = struct{}{}
			}
		}
	}

	for _, diff := range diffResp {
		if _, ok := changesMap[diff.OldPath]; !ok {
			changesMap[diff.OldPath] = struct{}{}
		}
		if _, ok := changesMap[diff.NewPath]; !ok {
			changesMap[diff.NewPath] = struct{}{}
		}
	}

	return changesMap
}
