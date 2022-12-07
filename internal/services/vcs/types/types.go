package types

import "time"

// Request method and content types, headers, etc. mainly for convenience.
const (
	GETMethodType    = "GET"
	POSTMethodType   = "POST"
	DELETEMethodType = "DELETE"

	JSONContentType = "application/json"
	FormContentType = "application/x-www-form-urlencoded"

	BearerAuthPrefix = "Bearer "

	HTTPSScheme       = "https"
	V1WebhookEndpoint = "v1/vcs/events"
)

// ToVCSEventTypeInput is the input for translating event types
// to VCSEventType equivalents.
type ToVCSEventTypeInput struct {
	EventHeader string
	Ref         string
}

// BuildOAuthAuthorizationURLInput is the input for building an
// authorization code URL that can be used to complete OAuth flow.
type BuildOAuthAuthorizationURLInput struct {
	Hostname           string
	OAuthClientID      string
	OAuthState         string
	RedirectURL        string
	UseReadWriteScopes bool // When true, API requests read-write scopes.
}

// BuildRepositoryURLInput is the input for building a repository URL.
type BuildRepositoryURLInput struct {
	Hostname       string
	RepositoryPath string
}

// TestConnectionInput is the input for testing a connection with a provider.
type TestConnectionInput struct {
	Hostname    string
	AccessToken string
}

// CreateAccessTokenInput is the input for creating an access token from a provider.
type CreateAccessTokenInput struct {
	Hostname          string
	ClientID          string
	ClientSecret      string
	AuthorizationCode string
	RedirectURI       string
	RefreshToken      string // Required when renewing a token, only for GitLab.
}

// GetProjectInput is the input for retrieving a project.
type GetProjectInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
}

// GetDiffInput is the input for retrieving a diff for a ref.
type GetDiffInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
	Ref            string // Branch or commit ID to diff.
}

// GetDiffsInput is the input for comparing two Git references.
type GetDiffsInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
	BaseRef        string // What we're comparing from. (Parent branch, tag etc.)
	HeadRef        string // What we're comparing to. (New branch, tag etc.)
}

// GetArchiveInput is the input for downloading a source archive.
type GetArchiveInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
	Ref            string
}

// CreateWebhookInput is the input for creating a webhook.
type CreateWebhookInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
	WebhookToken   []byte
}

// DeleteWebhookInput is the input for deleting a webhook.
type DeleteWebhookInput struct {
	Hostname       string
	AccessToken    string
	RepositoryPath string
	WebhookID      string
}

// AccessTokenPayload is the payload returned for creating /
// renewing an access token.
type AccessTokenPayload struct {
	ExpirationTimestamp *time.Time
	AccessToken         string
	RefreshToken        string
}

// GetProjectPayload is a subset of the payload returned when
// querying for a Git project.
type GetProjectPayload struct {
	DefaultBranch string
}

// GetDiffsPayload is the payload returned when retrieving diff(s).
type GetDiffsPayload struct {
	AlteredFiles map[string]struct{}
}

// WebhookPayload is the payload for manipulating webhooks.
type WebhookPayload struct {
	WebhookID string
}
