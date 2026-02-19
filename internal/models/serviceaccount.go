package models

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"golang.org/x/crypto/bcrypt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*ServiceAccount)(nil)

const (
	maximumTrustPolicies = 10
	clientSecretBytes    = 48 // 384 bits; NIST recommends minimum 128 bits for strong secrets
	// minClientSecretExpiration is the minimum expiration time for client secrets (1 day)
	minClientSecretExpiration = 24 * time.Hour
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
	Metadata                    ResourceMetadata
	ClientSecretExpiresAt       *time.Time
	SecretExpirationEmailSentAt *time.Time
	Name                        string
	Description                 string
	GroupID                     string
	CreatedBy                   string
	OIDCTrustPolicies           []OIDCTrustPolicy
	ClientSecretHash            *string
}

// ClientCredentialsEnabled returns true if client credentials are configured
func (s *ServiceAccount) ClientCredentialsEnabled() bool {
	return s.ClientSecretHash != nil
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

	// Verify at least one authentication method is enabled
	policyCount := len(s.OIDCTrustPolicies)
	if policyCount == 0 && !s.ClientCredentialsEnabled() {
		return errors.New("at least one OIDC trust policy or client credentials must be configured", errors.WithErrorCode(errors.EInvalid))
	}

	if policyCount > maximumTrustPolicies {
		return errors.New(fmt.Sprintf("%d exceeds the limit of %d OIDC trust policies", policyCount, maximumTrustPolicies), errors.WithErrorCode(errors.EInvalid))
	}

	for _, policy := range s.OIDCTrustPolicies {
		// Verify issuer is defined
		if policy.Issuer == "" {
			return errors.New("issuer URL is required for trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify that bound claims type is defined
		if policy.BoundClaimsType == "" {
			return errors.New("bound claims type is required for trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify that issuer is a valid URL
		if _, err := url.ParseRequestURI(policy.Issuer); err != nil {
			return errors.New("invalid issuer URL", errors.WithErrorCode(errors.EInvalid))
		}

		// Verify at least one claim is present in each trust policy
		if len(policy.BoundClaims) == 0 {
			return errors.New("a minimum of one claim is required in each OIDC trust policy", errors.WithErrorCode(errors.EInvalid))
		}

		if policy.BoundClaimsType == BoundClaimsTypeGlob {
			for k, v := range policy.BoundClaims {
				if v == "*" {
					return errors.New("the trust policy claim %q can't contain only a wildcard character", k, errors.WithErrorCode(errors.EInvalid))
				}
			}
		}
	}

	// Verify client secret and expiration are both set or both nil
	if (s.ClientSecretHash == nil) != (s.ClientSecretExpiresAt == nil) {
		return errors.New("client secret and expiration must both be set or both be nil")
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

// GenerateClientSecret generates and sets a cryptographically secure client secret.
// expiresAt is optional; if nil, defaults to maxExpirationDays from now.
// Must be between 1 day and maxExpirationDays from now.
// If maxExpirationDays is 0, client credentials are disabled.
func (s *ServiceAccount) GenerateClientSecret(expiresAt *time.Time, maxExpirationDays int) (string, error) {
	if maxExpirationDays <= 0 {
		return "", errors.New("client credentials are disabled", errors.WithErrorCode(errors.EInvalid))
	}

	now := time.Now()
	minExpiration := now.Add(minClientSecretExpiration)
	maxExpiration := now.Add(time.Duration(maxExpirationDays) * 24 * time.Hour)

	expiration := maxExpiration
	if expiresAt != nil {
		if expiresAt.Before(minExpiration) || expiresAt.After(maxExpiration) {
			return "", errors.New(
				"client secret expiration must be between 1 and %d days from now", maxExpirationDays,
				errors.WithErrorCode(errors.EInvalid),
			)
		}

		expiration = *expiresAt
	}

	secretBytes := make([]byte, clientSecretBytes)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", err
	}

	secret := base64.URLEncoding.EncodeToString(secretBytes)

	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	s.ClientSecretHash = ptr.String(string(hash))
	s.ClientSecretExpiresAt = ptr.Time(expiration)
	s.SecretExpirationEmailSentAt = nil

	return secret, nil
}

// VerifyClientSecret verifies the provided secret against the stored hash and checks expiration.
func (s *ServiceAccount) VerifyClientSecret(secret string) bool {
	if s.ClientSecretHash == nil {
		return false
	}

	if s.ClientSecretExpiresAt != nil && time.Now().After(*s.ClientSecretExpiresAt) {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(*s.ClientSecretHash), []byte(secret)) == nil
}
