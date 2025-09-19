package memory

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	_, err := New(map[string]string{})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
}

func TestSign(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	require.NoError(t, err)

	jwsProvider, err := New(map[string]string{})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	privJWK, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err = jwk.AssignKeyID(privJWK); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	privKeyBytes, err := json.Marshal(privJWK)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	token := jwt.New()
	err = token.Set(jwt.SubjectKey, "123")
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	signedToken, err := jwsProvider.Sign(ctx, payload, "", privKeyBytes, "")
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	pubKey, err := privJWK.PublicKey()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	err = jwk.AssignKeyID(pubKey)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	_, err = jws.Verify(signedToken, jws.WithKey(jwa.RS256, pubKey))
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
}

func TestInMemoryJWSProvider_Create(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successful key creation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			provider := &InMemoryJWSProvider{}

			result, err := provider.Create(ctx, "test-key-id")

			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.KeyData)
			require.NotNil(t, result.PublicKey)

			// Verify the key data is valid JSON
			var keyData map[string]interface{}
			err = json.Unmarshal(result.KeyData, &keyData)
			require.NoError(t, err)

			// Verify the public key has required fields
			require.Equal(t, jwa.RS256, result.PublicKey.Algorithm())
			require.Equal(t, string(jwk.ForSignature), result.PublicKey.KeyUsage())
		})
	}
}

func TestInMemoryJWSProvider_Delete(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "successful key deletion",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			provider := &InMemoryJWSProvider{}

			err := provider.Delete(ctx, "test-key-id", []byte("test-key-data"))

			require.NoError(t, err)
		})
	}
}

func TestInMemoryJWSProvider_SupportsKeyRotation(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{
			name:     "supports key rotation",
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			provider := &InMemoryJWSProvider{}

			result := provider.SupportsKeyRotation()

			require.Equal(t, test.expected, result)
		})
	}
}
