package models

import (
	"net/url"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*VCSProvider)(nil)

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

// GetID returns the Metadata ID.
func (v *VCSProvider) GetID() string {
	return v.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (v *VCSProvider) GetGlobalID() string {
	return gid.ToGlobalID(v.GetModelType(), v.Metadata.ID)
}

// GetModelType returns the model type.
func (v *VCSProvider) GetModelType() types.ModelType {
	return types.VCSProviderModelType
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

// GetResourcePath returns the resource path
func (v *VCSProvider) GetResourcePath() string {
	return strings.Split(v.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetGroupPath returns the group path
func (v *VCSProvider) GetGroupPath() string {
	resourcePath := v.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}
