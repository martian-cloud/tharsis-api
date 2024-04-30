package awskms

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewPlugin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keyID := "123"

	c := mockClient{}
	c.Test(t)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	c.On("GetPublicKey", ctx, &kms.GetPublicKeyInput{KeyId: &keyID}).Return(&kms.GetPublicKeyOutput{
		PublicKey: pubKey,
	}, nil)

	clientBuilder := func(_ context.Context, _ string) (client, error) {
		return &c, nil
	}

	jwsProvider, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": keyID,
		},
		clientBuilder,
		getPublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, jwsProvider.client)
	assert.NotNil(t, jwsProvider.keyID)
	assert.NotNil(t, jwsProvider.pubKey)
	assert.NotNil(t, jwsProvider.keySet)
}

func TestNewPluginWithMissingConfig(t *testing.T) {
	_, err := newPlugin(
		context.Background(),
		map[string]string{},
		nil,
		nil,
	)
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.Contains(t, err.Error(), "AWS KMS JWS provider plugin requires plugin data")
}

func TestSignWithValidKey(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pubKey, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	jwkKey, err := jwk.FromRaw(privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	err = jwk.AssignKeyID(jwkKey)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "test-key"

	token := jwt.New()
	_ = token.Set(jwt.SubjectKey, "123")

	payload, err := jwt.NewSerializer().Serialize(token)
	if err != nil {
		t.Fatal(err)
	}

	hdrs := jws.NewHeaders()
	_ = hdrs.Set(jws.TypeKey, "JWT")
	_ = hdrs.Set(jws.KeyIDKey, jwkKey.KeyID())

	sig := jws.NewSignature()
	sig.SetProtectedHeaders(hdrs)
	signer, err := jws.NewSigner(jwa.RS256)
	if err != nil {
		t.Fatal(err)
	}

	signature, _, err := sig.Sign(payload, signer, privKey)
	if err != nil {
		t.Fatal(err)
	}

	c := mockClient{}
	c.Test(t)

	c.On("GetPublicKey", ctx, &kms.GetPublicKeyInput{KeyId: &keyID}).Return(&kms.GetPublicKeyOutput{
		PublicKey: pubKey,
	}, nil)

	c.On("Sign", ctx, mock.Anything).Return(&kms.SignOutput{
		Signature: signature,
	}, nil)

	mockClientBuilder := func(_ context.Context, _ string) (client, error) {
		return &c, nil
	}

	jwsProvider, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": keyID,
		},
		mockClientBuilder,
		getPublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	signedToken, err := jwsProvider.Sign(ctx, payload)
	if err != nil {
		t.Fatal(err)
	}

	_, err = jws.Verify(signedToken, jws.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatal(err)
	}
}

func TestVerify(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	invalidPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	pubKeyDerFormat, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	jwkKey, err := jwk.FromRaw(privKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	err = jwk.AssignKeyID(jwkKey)
	if err != nil {
		t.Fatal(err)
	}

	keyID := "test-key"

	// Test cases
	tests := []struct {
		name      string
		privKey   *rsa.PrivateKey
		kid       string
		expectErr string
	}{
		{
			name:    "Valid Signature",
			privKey: privKey,
			kid:     jwkKey.KeyID(),
		},
		{
			name:      "Invalid Key ID",
			privKey:   privKey,
			kid:       "invalid",
			expectErr: `key provider 0 failed: failed to find key with key ID "invalid" in key set`,
		},
		{
			name:      "Invalid Signature",
			privKey:   invalidPrivKey,
			kid:       jwkKey.KeyID(),
			expectErr: "could not verify message using any of the signatures or keys",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			c := mockClient{}
			c.Test(t)

			c.On("GetPublicKey", ctx, &kms.GetPublicKeyInput{KeyId: &keyID}).Return(&kms.GetPublicKeyOutput{
				PublicKey: pubKeyDerFormat,
			}, nil)

			token := jwt.New()
			_ = token.Set(jwt.SubjectKey, "123")

			payload, err := jwt.NewSerializer().Serialize(token)
			if err != nil {
				t.Fatal(err)
			}

			hdrs := jws.NewHeaders()
			_ = hdrs.Set(jws.TypeKey, "JWT")
			_ = hdrs.Set(jws.KeyIDKey, test.kid)
			signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256, test.privKey, jws.WithProtectedHeaders(hdrs)))
			if err != nil {
				t.Fatal(err)
			}

			mockClientBuilder := func(_ context.Context, _ string) (client, error) {
				return &c, nil
			}

			jwsProvider, err := newPlugin(
				ctx,
				map[string]string{
					"region": "us-east-1",
					"key_id": keyID,
				},
				mockClientBuilder,
				getPublicKey,
			)
			if err != nil {
				t.Fatal(err)
			}

			err = jwsProvider.Verify(ctx, signed)
			if test.expectErr != "" {
				assert.EqualError(t, err, test.expectErr)
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
}
