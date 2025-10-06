package awskms

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestNewPlugin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockKMSClient := newMockKmsClient(t)
	mockSTSClient := newMockStsClient(t)

	clientBuilder := func(_ context.Context, _ string) (kmsClient, stsClient, error) {
		return mockKMSClient, mockSTSClient, nil
	}

	mockLogger, _ := logger.NewForTest()

	jwsProvider, err := newPlugin(
		ctx,
		mockLogger,
		map[string]string{
			"region":       "us-east-1",
			"tags":         "env=test,service=tharsis",
			"alias_prefix": "test-key",
		},
		clientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, jwsProvider.kmsClient)
	assert.NotNil(t, jwsProvider.stsClient)
	assert.NotNil(t, jwsProvider.logger)
	assert.NotNil(t, jwsProvider.tags)
	assert.Equal(t, 2, len(jwsProvider.tags))
	// Verify tags
	assert.Equal(t, "env", *jwsProvider.tags[0].TagKey)
	assert.Equal(t, "test", *jwsProvider.tags[0].TagValue)
	assert.Equal(t, "service", *jwsProvider.tags[1].TagKey)
	assert.Equal(t, "tharsis", *jwsProvider.tags[1].TagValue)
	assert.Equal(t, "test-key", jwsProvider.keyAliasPrefix)
}

func TestNewPluginWithMissingConfig(t *testing.T) {
	mockLogger, _ := logger.NewForTest()

	_, err := newPlugin(
		context.Background(),
		mockLogger,
		map[string]string{},
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

	mockKMSClient := newMockKmsClient(t)
	mockSTSClient := newMockStsClient(t)

	mockKMSClient.On("Sign", ctx, mock.Anything).Return(&kms.SignOutput{
		Signature: signature,
	}, nil)

	clientBuilder := func(_ context.Context, _ string) (kmsClient, stsClient, error) {
		return mockKMSClient, mockSTSClient, nil
	}

	mockLogger, _ := logger.NewForTest()

	jwsProvider, err := newPlugin(
		ctx,
		mockLogger,
		map[string]string{
			"region": "us-east-1",
		},
		clientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	signedToken, err := jwsProvider.Sign(ctx, payload, keyID, nil, jwkKey.KeyID())
	if err != nil {
		t.Fatal(err)
	}

	_, err = jws.Verify(signedToken, jws.WithKey(jwa.RS256, jwkKey))
	if err != nil {
		t.Fatal(err)
	}
}

func TestJWSProvider_Create(t *testing.T) {
	ctx := context.Background()
	keyID := "test-key"

	t.Run("successful key creation", func(t *testing.T) {
		mockSTS := newMockStsClient(t)
		mockKMS := newMockKmsClient(t)

		mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(&sts.GetCallerIdentityOutput{
			Account: ptr.String("123456789012"),
			Arn:     ptr.String("arn:aws:iam::123456789012:user/test-user"),
		}, nil)

		mockKMS.On("CreateKey", ctx, mock.Anything).Return(&kms.CreateKeyOutput{
			KeyMetadata: &types.KeyMetadata{
				KeyId: ptr.String("test-key-id"),
			},
		}, nil)

		mockKMS.On("CreateAlias", ctx, mock.Anything).Return(&kms.CreateAliasOutput{}, nil)

		privKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		require.NoError(t, err)

		mockKMS.On("GetPublicKey", ctx, mock.Anything).Return(&kms.GetPublicKeyOutput{
			PublicKey: pubKeyBytes,
		}, nil)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			stsClient:      mockSTS,
			keyAliasPrefix: "test-prefix",
		}

		result, err := provider.Create(ctx, keyID)
		require.NoError(t, err)

		expectedJWK, err := jwk.FromRaw(&privKey.PublicKey)
		require.NoError(t, err)
		require.NoError(t, jwk.AssignKeyID(expectedJWK))
		require.NoError(t, expectedJWK.Set(jwk.AlgorithmKey, jwa.RS256))
		require.NoError(t, expectedJWK.Set(jwk.KeyUsageKey, jwk.ForSignature))

		assert.Equal(t, expectedJWK.Algorithm(), result.PublicKey.Algorithm())
		assert.Equal(t, expectedJWK.KeyUsage(), result.PublicKey.KeyUsage())
		assert.Equal(t, expectedJWK.KeyType(), result.PublicKey.KeyType())
	})

	t.Run("key policy creation fails", func(t *testing.T) {
		mockSTS := newMockStsClient(t)
		mockKMS := newMockKmsClient(t)

		mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(nil, assert.AnError)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			stsClient:      mockSTS,
			keyAliasPrefix: "test-prefix",
		}

		_, err := provider.Create(ctx, keyID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create key policy")
	})

	t.Run("create key fails", func(t *testing.T) {
		mockSTS := newMockStsClient(t)
		mockKMS := newMockKmsClient(t)

		mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(&sts.GetCallerIdentityOutput{
			Account: ptr.String("123456789012"),
			Arn:     ptr.String("arn:aws:iam::123456789012:user/test-user"),
		}, nil)

		mockKMS.On("CreateKey", ctx, mock.Anything).Return(nil, assert.AnError)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			stsClient:      mockSTS,
			keyAliasPrefix: "test-prefix",
		}

		_, err := provider.Create(ctx, keyID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create KMS key")
	})

	t.Run("create alias fails", func(t *testing.T) {
		mockSTS := newMockStsClient(t)
		mockKMS := newMockKmsClient(t)

		mockSTS.On("GetCallerIdentity", ctx, mock.Anything).Return(&sts.GetCallerIdentityOutput{
			Account: ptr.String("123456789012"),
			Arn:     ptr.String("arn:aws:iam::123456789012:user/test-user"),
		}, nil)

		mockKMS.On("CreateKey", ctx, mock.Anything).Return(&kms.CreateKeyOutput{
			KeyMetadata: &types.KeyMetadata{
				KeyId: ptr.String("test-key-id"),
			},
		}, nil)

		mockKMS.On("CreateAlias", ctx, mock.Anything).Return(nil, assert.AnError)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			stsClient:      mockSTS,
			keyAliasPrefix: "test-prefix",
		}

		_, err := provider.Create(ctx, keyID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create alias for KMS key")
	})
}

func TestJWSProvider_Delete(t *testing.T) {
	ctx := context.Background()
	keyID := "test-key"

	t.Run("successful key deletion", func(t *testing.T) {
		mockKMS := newMockKmsClient(t)

		mockKMS.On("DescribeKey", ctx, mock.Anything).Return(&kms.DescribeKeyOutput{
			KeyMetadata: &types.KeyMetadata{
				KeyId: ptr.String("test-key-id"),
			},
		}, nil)

		mockKMS.On("ScheduleKeyDeletion", ctx, mock.Anything).Return(&kms.ScheduleKeyDeletionOutput{}, nil)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			keyAliasPrefix: "test-prefix",
		}

		err := provider.Delete(ctx, keyID, nil)
		require.NoError(t, err)
	})

	t.Run("describe key fails", func(t *testing.T) {
		mockKMS := newMockKmsClient(t)

		mockKMS.On("DescribeKey", ctx, mock.Anything).Return(nil, assert.AnError)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			keyAliasPrefix: "test-prefix",
		}

		err := provider.Delete(ctx, keyID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to describe KMS key")
	})

	t.Run("schedule key deletion fails", func(t *testing.T) {
		mockKMS := newMockKmsClient(t)

		mockKMS.On("DescribeKey", ctx, mock.Anything).Return(&kms.DescribeKeyOutput{
			KeyMetadata: &types.KeyMetadata{
				KeyId: ptr.String("test-key-id"),
			},
		}, nil)

		mockKMS.On("ScheduleKeyDeletion", ctx, mock.Anything).Return(nil, assert.AnError)

		provider := &JWSProvider{
			kmsClient:      mockKMS,
			keyAliasPrefix: "test-prefix",
		}

		err := provider.Delete(ctx, keyID, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to schedule KMS key deletion")
	})
}

func TestJWSProvider_SupportsKeyRotation(t *testing.T) {
	provider := &JWSProvider{}
	assert.True(t, provider.SupportsKeyRotation())
}
