package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs/types"
)

const (
	// Content-Type specific to GitHub.
	jsonContentType = "application/vnd.github+json"

	// pushEvent represents a GitHub push event.
	pushEvent = "push"

	// pullRequestEvent represents a GitHub pull request event.
	pullRequestEvent = "pull_request"

	// gitHubReadWriteOAuthScopes represents space-separated OAuth scopes that are requested
	// from the GitHub VCS provider. Passed in as 'scope' query parameter.
	// NOTE: GitHub does not seem to support read-only 'repo' scope.
	// https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps#available-scopes
	gitHubReadWriteOAuthScopes = "repo read:user"
)

var (
	// eventTypes that get registered with GitHub. These can be used
	// to determine webhook events as well.
	eventTypes = []string{
		pushEvent,        // For changes pushed to branch or tag.
		pullRequestEvent, // For pull requests.
	}

	// supportedGitHubPRActions contains the list of actions
	// for a pull request that can trigger a run.
	supportedGitHubPRActions = map[string]struct{}{
		"opened":      {}, // When a PR is opened.
		"synchronize": {}, // When a PR is updated.
	}

	// defaultURL is the default API URL for this provider type.
	defaultURL = url.URL{
		Scheme: "https",
		Host:   "api.github.com",
	}
)

// createWebhookBody is the request body for creating a webhook.
type createWebhookBody struct {
	Name   string                 `json:"name"`
	Config map[string]interface{} `json:"config"`
	Events []string               `json:"events"`
	Active bool                   `json:"active"`
}

// getProjectResponse is the response struct for retrieving a project.
type getProjectResponse struct {
	DefaultBranch string `json:"default_branch"`
}

// getDiffsResponse is the response struct for retrieving diff(s).
type getDiffsResponse struct {
	Files []struct {
		Filename string `json:"filename"`
	} `json:"files"`
}

// createWebhookResponse is the response struct for creating
// a webhook in GitHub.
type createWebhookResponse struct {
	ID int `json:"id"`
}

// createAccessTokenResponse is the response struct for creating an access token.
type createAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

// Provider represents a particular VCS provider.
type Provider struct {
	logger     logger.Logger
	client     *http.Client
	tharsisURL string
}

// New creates a new Provider instance.
func New(
	ctx context.Context,
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
	_, ok := supportedGitHubPRActions[action]
	return ok
}

// ToVCSEventType determines whether the event is supported
// and translates the event type to VCSEventType equivalent.
func (p *Provider) ToVCSEventType(input *types.ToVCSEventTypeInput) models.VCSEventType {
	// Since GitHub uses 'push' events for both tags and branches,
	// we must differentiate between the two by using the ref.
	if input.EventHeader == pushEvent {
		if strings.HasPrefix(input.Ref, "refs/heads/") {
			return models.BranchEventType
		}
		return models.TagEventType
	}

	if input.EventHeader == pullRequestEvent {
		return models.MergeRequestEventType
	}

	return ""
}

// BuildOAuthAuthorizationURL build the authorization code URL which is
// used to redirect the user to the VCS provider to complete OAuth flow.
func (p *Provider) BuildOAuthAuthorizationURL(input *types.BuildOAuthAuthorizationURLInput) (string, error) {
	// Add queries.
	queries := input.ProviderURL.Query()
	queries.Add("client_id", input.OAuthClientID)
	queries.Add("redirect_uri", input.RedirectURL)
	queries.Add("state", input.OAuthState)
	queries.Add("scope", gitHubReadWriteOAuthScopes)
	input.ProviderURL.RawQuery = queries.Encode()

	// Can't use GitHub's API here. Must remove ".api" prefix.
	input.ProviderURL.Host = p.stripAPIPrefix(input.ProviderURL.Host)
	return url.JoinPath(input.ProviderURL.String(), "login/oauth/authorize")
}

// BuildRepositoryURL returns the repository URL associated with the provider.
func (p *Provider) BuildRepositoryURL(input *types.BuildRepositoryURLInput) (string, error) {
	input.ProviderURL.Host = p.stripAPIPrefix(input.ProviderURL.Host)
	return url.JoinPath(input.ProviderURL.String(), input.RepositoryPath)
}

// TestConnection simply queries for the user metadata that's
// associated with the access token to verify validity.
// https://docs.github.com/en/rest/users/users#get-the-authenticated-user
func (p *Provider) TestConnection(ctx context.Context, input *types.TestConnectionInput) error {
	endpoint, err := url.JoinPath(input.ProviderURL.String(), "user")
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
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
// https://docs.github.com/en/rest/repos/repos#get-a-repository
func (p *Provider) GetProject(ctx context.Context, input *types.GetProjectInput) (*types.GetProjectPayload, error) {
	// Build the request URL.
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
	resp, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"failed to query for project. Response status: %s", resp.Status)
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
// https://docs.github.com/en/rest/commits/commits#get-a-commit
func (p *Provider) GetDiff(ctx context.Context, input *types.GetDiffInput) (*types.GetDiffsPayload, error) {
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
		"commits",
		input.Ref,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
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

	diffResp := getDiffsResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&diffResp); err != nil {
		return nil, err
	}

	return &types.GetDiffsPayload{
		AlteredFiles: p.createChangesMap(&diffResp),
	}, nil
}

// GetDiffs retrieves diffs for two different refs (branches, commits, etc.)
// https://docs.github.com/en/rest/commits/commits#compare-two-commits
func (p *Provider) GetDiffs(ctx context.Context, input *types.GetDiffsInput) (*types.GetDiffsPayload, error) {
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
		"compare",
		input.BaseRef + "..." + input.HeadRef,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
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
			p.logger.Errorf("failed to close response body in GetDiff: %v", err)
		}
	}()

	diffResp := getDiffsResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&diffResp); err != nil {
		return nil, err
	}

	return &types.GetDiffsPayload{
		AlteredFiles: p.createChangesMap(&diffResp),
	}, nil
}

// GetArchive downloads the entire repository archive for a branch or tag.
// https://docs.github.com/en/rest/repos/contents#download-a-repository-archive-tar
func (p *Provider) GetArchive(ctx context.Context, input *types.GetArchiveInput) (*http.Response, error) {
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
		"tarball",
		input.Ref,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
	request.Header.Add("Authorization", types.BearerAuthPrefix+input.AccessToken)

	// Make the request.
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
// https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#web-application-flow
func (p *Provider) CreateAccessToken(ctx context.Context, input *types.CreateAccessTokenInput) (*types.AccessTokenPayload, error) {
	path := strings.Join([]string{
		"login",
		"oauth",
		"access_token",
	}, "/")

	// Add queries.
	queries := input.ProviderURL.Query()
	queries.Add("client_id", input.ClientID)
	queries.Add("client_secret", input.ClientSecret)
	queries.Add("code", input.AuthorizationCode)
	queries.Add("redirect_uri", input.RedirectURI)
	input.ProviderURL.RawQuery = queries.Encode()

	// Cannot use GitHub's API hostname here. Must trim "api." prefix here.
	input.ProviderURL.Host = p.stripAPIPrefix(input.ProviderURL.Host)

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
			p.logger.Errorf("failed to close response body in GetDiffs: %v", err)
		}
	}()

	tokenResp := createAccessTokenResponse{}
	if err = json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &types.AccessTokenPayload{
		AccessToken: tokenResp.AccessToken,
	}, nil
}

// CreateWebhook creates a webhook at the specified provider.
// Returns the webhook ID from the response.
// https://docs.github.com/en/rest/webhooks/repos#create-a-repository-webhook
func (p *Provider) CreateWebhook(ctx context.Context, input *types.CreateWebhookInput) (*types.WebhookPayload, error) {
	// Build the request URL.
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
		"hooks",
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return nil, err
	}

	// Build Tharsis webhook endpoint.
	parsedURL, err := url.Parse(p.tharsisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Tharsis URL: %v", err)
	}
	parsedURL.Path = types.V1WebhookEndpoint

	// Add the token as a query param.
	queries := parsedURL.Query()
	queries.Set("token", string(input.WebhookToken))
	parsedURL.RawQuery = queries.Encode()

	// Create the request body.
	body := createWebhookBody{
		Name:   "web", // Only possible value.
		Active: true,
		Events: eventTypes,
		Config: map[string]interface{}{
			// GitHub doesn't seem to support passing in token via 'token' field.
			"url":          parsedURL.String(),
			"content_type": "json",
			"insecure_ssl": 0, // Don't allow webhook to connect with insecure SSL.
		},
	}

	marshalledBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(marshalledBody))
	if err != nil {
		return nil, fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
	request.Header.Add("Content-Type", jsonContentType)
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
// https://docs.github.com/en/rest/webhooks/repos#delete-a-repository-webhook
func (p *Provider) DeleteWebhook(ctx context.Context, input *types.DeleteWebhookInput) error {
	// Build the request URL
	path := strings.Join([]string{
		"repos",
		input.RepositoryPath,
		"hooks",
		input.WebhookID,
	}, "/")

	endpoint, err := url.JoinPath(input.ProviderURL.String(), path)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to prepare HTTP request: %v", err)
	}

	// Add the headers.
	request.Header.Add("Accept", jsonContentType)
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

// stripAPIPrefix removes the API prefix if using the provider's default URL.
// This is necessary for any user-facing actions, such as, OAuth flow,
// repository URL etc. i.e. actions that take place in the browser.
func (p *Provider) stripAPIPrefix(host string) string {
	if host == defaultURL.Host {
		host = strings.TrimPrefix(host, "api.")
	}

	return host
}

// createChangesMap creates a unique map of files that have been altered.
func (p *Provider) createChangesMap(diffResp *getDiffsResponse) map[string]struct{} {
	changesMap := map[string]struct{}{}
	for _, file := range diffResp.Files {
		if _, ok := changesMap[file.Filename]; !ok {
			changesMap[file.Filename] = struct{}{}
		}
	}

	return changesMap
}
