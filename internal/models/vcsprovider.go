package models

import (
	"strings"
	"time"
)

// VCSProviderType defines the type of version control system (VCS) provider.
type VCSProviderType string

// VCSProviderType and OAuthTokenType enums.
const (
	GitLabProviderType VCSProviderType = "gitlab"
	GitHubProviderType VCSProviderType = "github"
)

// VCSProvider represents a version control system provider.
type VCSProvider struct {
	OAuthAccessTokenExpiresAt *time.Time
	CreatedBy                 string
	Hostname                  string
	Name                      string
	Description               string
	ResourcePath              string
	Type                      VCSProviderType
	GroupID                   string
	OAuthClientSecret         string
	OAuthClientID             string
	OAuthState                *string
	OAuthAccessToken          *string
	OAuthRefreshToken         *string
	Metadata                  ResourceMetadata
	AutoCreateWebhooks        bool
}

// Validate returns an error if the model is not valid
func (vp *VCSProvider) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(vp.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	return verifyValidDescription(vp.Description)
}

// GetGroupPath returns the group path
func (vp *VCSProvider) GetGroupPath() string {
	return vp.ResourcePath[:strings.LastIndex(vp.ResourcePath, "/")]
}
