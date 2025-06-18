package models

import (
	"net/url"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*FederatedRegistry)(nil)

// FederatedRegistry represents a client-side federated registry.
type FederatedRegistry struct {
	Metadata  ResourceMetadata
	Hostname  string
	GroupID   string
	Audience  string
	CreatedBy string
}

// GetID returns the ID of the FederatedRegistry resource
func (f *FederatedRegistry) GetID() string {
	return f.Metadata.ID
}

// GetGlobalID returns the GID of the FederatedRegistry resource
func (f *FederatedRegistry) GetGlobalID() string {
	return gid.ToGlobalID(f.GetModelType(), f.Metadata.ID)
}

// GetModelType returns the Model's type
func (f *FederatedRegistry) GetModelType() types.ModelType {
	return types.FederatedRegistryModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (f *FederatedRegistry) ResolveMetadata(key string) (*string, error) {
	return f.Metadata.resolveFieldValue(key)
}

// Validate validates the FederatedRegistry resource
func (f *FederatedRegistry) Validate() error {
	// Validate hostname
	if err := validateHostname(f.Hostname); err != nil {
		return err
	}

	// Validate audience
	if err := validateAudience(f.Audience); err != nil {
		return err
	}

	return nil
}

// validateHostname validates that the hostname follows proper format
func validateHostname(hostname string) error {
	// Check for empty hostname first
	if hostname == "" {
		return errors.New("hostname cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	// Check maximum length
	if len(hostname) > 64 {
		return errors.New("hostname cannot exceed 64 characters", errors.WithErrorCode(errors.EInvalid))
	}

	// Use url.Parse to validate the hostname
	// First, add a scheme if not present to make url.Parse work correctly
	urlToCheck := hostname
	if !strings.Contains(urlToCheck, "://") {
		urlToCheck = "https://" + urlToCheck
	}

	parsedURL, err := url.Parse(urlToCheck)
	if err != nil {
		return errors.Wrap(err, "invalid hostname format", errors.WithErrorCode(errors.EInvalid))
	}

	// Check for empty host
	if parsedURL.Host == "" {
		return errors.New("invalid hostname: host part is empty", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// validateAudience validates that the audience meets basic requirements
func validateAudience(audience string) error {
	if audience == "" {
		return errors.New("audience cannot be empty", errors.WithErrorCode(errors.EInvalid))
	}

	// Maximum length requirement
	if len(audience) > 64 {
		return errors.New("audience cannot exceed 64 characters", errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
