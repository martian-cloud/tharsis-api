package models

import (
	"fmt"
	"net/url"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
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
	if len(s.OIDCTrustPolicies) == 0 {
		return errors.NewError(errors.EInvalid, "A minimum of one OIDC trust policy is required")
	}

	issuerMap := map[string]bool{}

	for _, policy := range s.OIDCTrustPolicies {
		// Verify issuer is defined
		if policy.Issuer == "" {
			return errors.NewError(errors.EInvalid, "Issuer URL is required for trust policy")
		}

		// Verify that issuer is a valid URL
		if _, err := url.ParseRequestURI(policy.Issuer); err != nil {
			return errors.NewError(errors.EInvalid, "Invalid issuer URL")
		}

		// Verify that issuer URL hasn't already been defined
		if ok := issuerMap[policy.Issuer]; ok {
			return errors.NewError(errors.EInvalid, fmt.Sprintf("Issuer %s can only be included in a single trust policy for this service account", policy.Issuer))
		}

		/// Add issuer URL to map
		issuerMap[policy.Issuer] = true

		// Verify at least one claim is present in each trust policy
		if len(policy.BoundClaims) == 0 {
			return errors.NewError(errors.EInvalid, "A minimum of one claim is required in each OIDC trust policy")
		}
	}

	return nil
}

// GetGroupPath returns the group path
func (s *ServiceAccount) GetGroupPath() string {
	return s.ResourcePath[:strings.LastIndex(s.ResourcePath, "/")]
}
