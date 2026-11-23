package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestSCIMToken_Validate(t *testing.T) {
	in := func(d time.Duration) *time.Time {
		expiresAt := time.Now().Add(d)
		return &expiresAt
	}

	tests := []struct {
		name            string
		token           *SCIMToken
		expectErrorCode errors.CodeType
	}{
		{
			name:  "valid",
			token: &SCIMToken{TokenHash: "hash", IDPIssuerURL: "https://idp.example.com"},
		},
		{
			name:            "missing token hash",
			token:           &SCIMToken{IDPIssuerURL: "https://idp.example.com"},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "missing IDP issuer URL",
			token:           &SCIMToken{TokenHash: "hash"},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:  "expiration within bounds",
			token: &SCIMToken{TokenHash: "hash", IDPIssuerURL: "https://idp.example.com", ExpiresAt: in(48 * time.Hour)},
		},
		{
			name:            "expiration in the past",
			token:           &SCIMToken{TokenHash: "hash", IDPIssuerURL: "https://idp.example.com", ExpiresAt: in(-time.Hour)},
			expectErrorCode: errors.EInvalid,
		},
		{
			name:            "expiration under 1 day",
			token:           &SCIMToken{TokenHash: "hash", IDPIssuerURL: "https://idp.example.com", ExpiresAt: in(12 * time.Hour)},
			expectErrorCode: errors.EInvalid,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.token.Validate()
			if test.expectErrorCode != "" {
				assert.Equal(t, test.expectErrorCode, errors.ErrorCode(err))
				return
			}

			assert.NoError(t, err)
		})
	}
}
