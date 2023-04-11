package models

import (
	"fmt"
	"net/url"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	maximumTrustPolicies = 10
)

// OIDCTrustPolicy defined the IDP that can be used for logging into the service account
type OIDCTrustPolicy struct {
	BoundClaims map[string]string
	Issuer      string
}

// ServiceAccount provided M2M authentication
type ServiceAccount struct {
	Metadata          ResourceMetadata
	ResourcePath      string
	Name              string
	Description       string
	GroupID           string
	CreatedBy         string
	OIDCTrustPolicies []OIDCTrustPolicy
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (s *ServiceAccount) ResolveMetadata(key string) (string, error) {
	return s.Metadata.resolveFieldValue(key)
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
		return errors.New(errors.EInvalid, "A minimum of one OIDC trust policy is required")
	}
	if policyCount > maximumTrustPolicies {
		return errors.New(errors.EInvalid,
			fmt.Sprintf("%d exceeds the limit of %d OIDC trust policies", policyCount, maximumTrustPolicies))
	}

	for _, policy := range s.OIDCTrustPolicies {
		// Verify issuer is defined
		if policy.Issuer == "" {
			return errors.New(errors.EInvalid, "Issuer URL is required for trust policy")
		}

		// Verify that issuer is a valid URL
		if _, err := url.ParseRequestURI(policy.Issuer); err != nil {
			return errors.New(errors.EInvalid, "Invalid issuer URL")
		}

		// Verify at least one claim is present in each trust policy
		if len(policy.BoundClaims) == 0 {
			return errors.New(errors.EInvalid, "A minimum of one claim is required in each OIDC trust policy")
		}
	}

	return nil
}

// GetGroupPath returns the group path
func (s *ServiceAccount) GetGroupPath() string {
	return s.ResourcePath[:strings.LastIndex(s.ResourcePath, "/")]
}
