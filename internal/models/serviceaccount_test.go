package models

import (
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestServiceAccount_IsClientCredentialsEnabled(t *testing.T) {
	tests := []struct {
		name             string
		clientSecretHash *string
		expected         bool
	}{
		{
			name:             "enabled when hash is set",
			clientSecretHash: ptr.String("somehash"),
			expected:         true,
		},
		{
			name:             "disabled when hash is nil",
			clientSecretHash: nil,
			expected:         false,
		},
		{
			name:             "enabled with empty hash string",
			clientSecretHash: ptr.String(""),
			expected:         true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sa := &ServiceAccount{ClientSecretHash: test.clientSecretHash}
			assert.Equal(t, test.expected, sa.ClientCredentialsEnabled())
		})
	}
}

func TestServiceAccount_GenerateClientSecret(t *testing.T) {
	maxExpirationDays := 90

	tests := []struct {
		name              string
		expiresAt         *time.Time
		maxExpirationDays int
		expectErrorCode   errors.CodeType
	}{
		{
			name:              "generates secret with default expiration",
			expiresAt:         nil,
			maxExpirationDays: maxExpirationDays,
		},
		{
			name:              "generates secret with custom expiration",
			expiresAt:         ptr.Time(time.Now().Add(48 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
		},
		{
			name:              "generates secret at minimum expiration boundary",
			expiresAt:         ptr.Time(time.Now().Add(25 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
		},
		{
			name:              "generates secret at maximum expiration boundary",
			expiresAt:         ptr.Time(time.Now().Add(89 * 24 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
		},
		{
			name:              "rejects expiration too soon",
			expiresAt:         ptr.Time(time.Now().Add(1 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "rejects expiration at exactly minimum boundary",
			expiresAt:         ptr.Time(time.Now().Add(23 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "rejects expiration too far",
			expiresAt:         ptr.Time(time.Now().Add(91 * 24 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "rejects expiration in the past",
			expiresAt:         ptr.Time(time.Now().Add(-24 * time.Hour)),
			maxExpirationDays: maxExpirationDays,
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "rejects zero max expiration days",
			expiresAt:         nil,
			maxExpirationDays: 0,
			expectErrorCode:   errors.EInvalid,
		},
		{
			name:              "rejects negative max expiration days",
			expiresAt:         nil,
			maxExpirationDays: -1,
			expectErrorCode:   errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sa := &ServiceAccount{
				SecretExpirationEmailSentAt: ptr.Time(time.Now()),
			}

			secret, err := sa.GenerateClientSecret(test.expiresAt, test.maxExpirationDays)

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, secret)
			assert.NotNil(t, sa.ClientSecretHash)
			assert.NotNil(t, sa.ClientSecretExpiresAt)
			assert.Nil(t, sa.SecretExpirationEmailSentAt)
			assert.True(t, sa.VerifyClientSecret(secret))
		})
	}
}

func TestServiceAccount_GenerateClientSecret_Uniqueness(t *testing.T) {
	sa := &ServiceAccount{}
	secrets := make(map[string]bool)

	for i := 0; i < 10; i++ {
		secret, err := sa.GenerateClientSecret(nil, 90)
		require.NoError(t, err)
		assert.False(t, secrets[secret], "generated duplicate secret")
		secrets[secret] = true
	}
}

func TestServiceAccount_VerifyClientSecret(t *testing.T) {
	saWithSecret := &ServiceAccount{}
	actualSecret, err := saWithSecret.GenerateClientSecret(nil, 90)
	require.NoError(t, err)

	tests := []struct {
		name           string
		serviceAccount *ServiceAccount
		secret         string
		expected       bool
	}{
		{
			name:           "valid secret",
			serviceAccount: saWithSecret,
			secret:         actualSecret,
			expected:       true,
		},
		{
			name:           "invalid secret",
			serviceAccount: saWithSecret,
			secret:         "wrong-secret",
			expected:       false,
		},
		{
			name:           "empty secret",
			serviceAccount: saWithSecret,
			secret:         "",
			expected:       false,
		},
		{
			name:           "no hash set",
			serviceAccount: &ServiceAccount{},
			secret:         actualSecret,
			expected:       false,
		},
		{
			name: "expired secret",
			serviceAccount: &ServiceAccount{
				ClientSecretHash:      saWithSecret.ClientSecretHash,
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(-1 * time.Hour)),
			},
			secret:   actualSecret,
			expected: false,
		},
		{
			name: "secret expires exactly now",
			serviceAccount: &ServiceAccount{
				ClientSecretHash:      saWithSecret.ClientSecretHash,
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(-1 * time.Second)),
			},
			secret:   actualSecret,
			expected: false,
		},
		{
			name: "secret not yet expired",
			serviceAccount: &ServiceAccount{
				ClientSecretHash:      saWithSecret.ClientSecretHash,
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(1 * time.Hour)),
			},
			secret:   actualSecret,
			expected: true,
		},
		{
			name: "nil expiration with valid hash",
			serviceAccount: &ServiceAccount{
				ClientSecretHash:      saWithSecret.ClientSecretHash,
				ClientSecretExpiresAt: nil,
			},
			secret:   actualSecret,
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.expected, test.serviceAccount.VerifyClientSecret(test.secret))
		})
	}
}

func TestServiceAccount_Validate(t *testing.T) {
	validTrustPolicy := []OIDCTrustPolicy{{
		Issuer:          "https://example.com",
		BoundClaimsType: BoundClaimsTypeString,
		BoundClaims:     map[string]string{"sub": "test"},
	}}

	tests := []struct {
		name            string
		serviceAccount  ServiceAccount
		expectErrorCode errors.CodeType
	}{
		{
			name: "valid with OIDC trust policy",
			serviceAccount: ServiceAccount{
				Metadata:          ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:              "valid-sa",
				OIDCTrustPolicies: validTrustPolicy,
			},
		},
		{
			name: "valid with client credentials",
			serviceAccount: ServiceAccount{
				Metadata:              ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:                  "valid-sa",
				ClientSecretHash:      ptr.String("hash"),
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(24 * time.Hour)),
			},
		},
		{
			name: "valid with both auth methods",
			serviceAccount: ServiceAccount{
				Metadata:              ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:                  "valid-sa",
				OIDCTrustPolicies:     validTrustPolicy,
				ClientSecretHash:      ptr.String("hash"),
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(24 * time.Hour)),
			},
		},
		{
			name: "valid with multiple trust policies",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{
					{Issuer: "https://example1.com", BoundClaimsType: BoundClaimsTypeString, BoundClaims: map[string]string{"sub": "test1"}},
					{Issuer: "https://example2.com", BoundClaimsType: BoundClaimsTypeString, BoundClaims: map[string]string{"sub": "test2"}},
				},
			},
		},
		{
			name: "valid with glob bound claims type",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:          "https://example.com",
					BoundClaimsType: BoundClaimsTypeGlob,
					BoundClaims:     map[string]string{"sub": "test*"},
				}},
			},
		},
		{
			name: "invalid - no auth method",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - empty trust policies and no client credentials",
			serviceAccount: ServiceAccount{
				Metadata:          ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:              "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - client secret hash without expiration",
			serviceAccount: ServiceAccount{
				Metadata:          ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:              "valid-sa",
				ClientSecretHash:  ptr.String("hash"),
				OIDCTrustPolicies: validTrustPolicy,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid - client secret expiration without hash",
			serviceAccount: ServiceAccount{
				Metadata:              ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:                  "valid-sa",
				ClientSecretExpiresAt: ptr.Time(time.Now().Add(24 * time.Hour)),
				OIDCTrustPolicies:     validTrustPolicy,
			},
			expectErrorCode: errors.EInternal,
		},
		{
			name: "invalid - trust policy without issuer",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					BoundClaimsType: BoundClaimsTypeString,
					BoundClaims:     map[string]string{"sub": "test"},
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - trust policy with invalid issuer URL",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:          "not-a-url",
					BoundClaimsType: BoundClaimsTypeString,
					BoundClaims:     map[string]string{"sub": "test"},
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - trust policy without bound claims type",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:      "https://example.com",
					BoundClaims: map[string]string{"sub": "test"},
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - trust policy without bound claims",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:          "https://example.com",
					BoundClaimsType: BoundClaimsTypeString,
					BoundClaims:     map[string]string{},
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - trust policy with nil bound claims",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:          "https://example.com",
					BoundClaimsType: BoundClaimsTypeString,
					BoundClaims:     nil,
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - glob claim with only wildcard",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: []OIDCTrustPolicy{{
					Issuer:          "https://example.com",
					BoundClaimsType: BoundClaimsTypeGlob,
					BoundClaims:     map[string]string{"sub": "*"},
				}},
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - exceeds max trust policies",
			serviceAccount: ServiceAccount{
				Metadata: ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:     "valid-sa",
				OIDCTrustPolicies: func() []OIDCTrustPolicy {
					policies := make([]OIDCTrustPolicy, 11)
					for i := range policies {
						policies[i] = OIDCTrustPolicy{
							Issuer:          "https://example.com",
							BoundClaimsType: BoundClaimsTypeString,
							BoundClaims:     map[string]string{"sub": "test"},
						}
					}
					return policies
				}(),
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - empty name",
			serviceAccount: ServiceAccount{
				Metadata:          ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:              "",
				OIDCTrustPolicies: validTrustPolicy,
			},
			expectErrorCode: errors.EInvalid,
		},
		{
			name: "invalid - name with invalid characters",
			serviceAccount: ServiceAccount{
				Metadata:          ResourceMetadata{TRN: types.TRNPrefix + "service_account:group/sa-name"},
				Name:              "invalid name!",
				OIDCTrustPolicies: validTrustPolicy,
			},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.serviceAccount.Validate()

			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestServiceAccount_GetGroupPath(t *testing.T) {
	tests := []struct {
		name     string
		trn      string
		expected string
	}{
		{
			name:     "single level group",
			trn:      types.TRNPrefix + "service_account:group/sa-name",
			expected: "group",
		},
		{
			name:     "nested group",
			trn:      types.TRNPrefix + "service_account:parent/child/sa-name",
			expected: "parent/child",
		},
		{
			name:     "deeply nested group",
			trn:      types.TRNPrefix + "service_account:a/b/c/d/sa-name",
			expected: "a/b/c/d",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sa := &ServiceAccount{Metadata: ResourceMetadata{TRN: test.trn}}
			assert.Equal(t, test.expected, sa.GetGroupPath())
		})
	}
}

func TestServiceAccount_GetResourcePath(t *testing.T) {
	tests := []struct {
		name     string
		trn      string
		expected string
	}{
		{
			name:     "single level group",
			trn:      types.TRNPrefix + "service_account:group/sa-name",
			expected: "group/sa-name",
		},
		{
			name:     "nested group",
			trn:      types.TRNPrefix + "service_account:parent/child/sa-name",
			expected: "parent/child/sa-name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sa := &ServiceAccount{Metadata: ResourceMetadata{TRN: test.trn}}
			assert.Equal(t, test.expected, sa.GetResourcePath())
		})
	}
}
