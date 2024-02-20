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
	val, err := s.Metadata.resolveFieldValue(key)
	if err != nil {
		switch key {
		case "group_path":
			val = s.GetGroupPath()
		default:
			return "", err
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

		// Verify that issuer is a valid URL
		if _, err := url.ParseRequestURI(policy.Issuer); err != nil {
			return errors.New("Invalid issuer URL", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify at least one claim is present in each trust policy
		if len(policy.BoundClaims) == 0 {
			return errors.New("A minimum of one claim is required in each OIDC trust policy", errors.WithErrorCode(errors.EInvalid))
		}
	}

	return nil
}

// GetGroupPath returns the group path
func (s *ServiceAccount) GetGroupPath() string {
	return s.ResourcePath[:strings.LastIndex(s.ResourcePath, "/")]
}
