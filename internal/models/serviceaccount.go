package models

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/smithy-go/ptr"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*ServiceAccount)(nil)

const (
	maximumTrustPolicies = 10
)

// BoundClaimsType defines the type of comparison to be used for bound claims
type BoundClaimsType string

const (
	// BoundClaimsTypeString is used for exact string matching
	BoundClaimsTypeString BoundClaimsType = "STRING"
	// BoundClaimsTypeGlob is used for glob pattern matching (i.e. a wildcard character can be used within the claim value)
	BoundClaimsTypeGlob BoundClaimsType = "GLOB"
)

// OIDCTrustPolicy defined the IDP that can be used for logging into the service account
type OIDCTrustPolicy struct {
	BoundClaimsType BoundClaimsType
	BoundClaims     map[string]string
	Issuer          string
}

// ServiceAccount provided M2M authentication
type ServiceAccount struct {
	Metadata          ResourceMetadata
	Name              string
	Description       string
	GroupID           string
	CreatedBy         string
	OIDCTrustPolicies []OIDCTrustPolicy
}

// GetID returns the Metadata ID.
func (s *ServiceAccount) GetID() string {
	return s.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (s *ServiceAccount) GetGlobalID() string {
	return gid.ToGlobalID(s.GetModelType(), s.Metadata.ID)
}

// GetModelType returns the model type.
func (s *ServiceAccount) GetModelType() types.ModelType {
	return types.ServiceAccountModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *ServiceAccount) ResolveMetadata(key string) (*string, error) {
	val, err := s.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			return ptr.String(s.GetGroupPath()), nil
		default:
			return nil, err
		}
	}

	return val, nil
}

// Validate returns an error if the model is not valid
func (s *ServiceAccount) Validate() error {
	// Verify name satisfies constraints
	if err := verifyValidName(s.Name); err != nil {
		return err
	}

	// Verify description satisfies constraints
	if err := verifyValidDescription(s.Description); err != nil {
		return err
	}

	// Verify at least one trust policy is defined
	policyCount := len(s.OIDCTrustPolicies)
	if policyCount == 0 {
		return errors.New("A minimum of one OIDC trust policy is required", errors.WithErrorCode(errors.EInvalid))
	}
	if policyCount > maximumTrustPolicies {
		return errors.New(fmt.Sprintf("%d exceeds the limit of %d OIDC trust policies", policyCount, maximumTrustPolicies), errors.WithErrorCode(errors.EInvalid))
	}

	for _, policy := range s.OIDCTrustPolicies {
		// Verify issuer is defined
		if policy.Issuer == "" {
			return errors.New("Issuer URL is required for trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify that bound claims type is defined
		if policy.BoundClaimsType == "" {
			return errors.New("Bound claims type is required for trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify that issuer is a valid URL
		if _, err := url.ParseRequestURI(policy.Issuer); err != nil {
			return errors.New("Invalid issuer URL", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify at least one claim is present in each trust policy
		if len(policy.BoundClaims) == 0 {
			return errors.New("A minimum of one claim is required in each OIDC trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		if policy.BoundClaimsType == BoundClaimsTypeGlob {
			for k, v := range policy.BoundClaims {
				if v == "*" {
					return errors.New("the trust policy claim %q can't contain only a wildcard character", k, errors.WithErrorCode(errors.EInvalid))
				}
			}
		}
	}

	return nil
}

// GetResourcePath returns the ServiceAccount resource's path
func (s *ServiceAccount) GetResourcePath() string {
	return strings.Split(s.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
}

// GetGroupPath returns the group path
func (s *ServiceAccount) GetGroupPath() string {
	resourcePath := s.GetResourcePath()
	return resourcePath[:strings.LastIndex(resourcePath, "/")]
}
