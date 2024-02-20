package models

import (
	"net/url"
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
	URL                       url.URL
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

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (v *VCSProvider) ResolveMetadata(key string) (string, error) {
	val, err := v.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			val = v.GetGroupPath()
		default:
			return "", err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (v *VCSProvider) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(v.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	return verifyValidDescription(v.Description)
}

// GetGroupPath returns the group path
func (v *VCSProvider) GetGroupPath() string {
	return v.ResourcePath[:strings.LastIndex(v.ResourcePath, "/")]
}
