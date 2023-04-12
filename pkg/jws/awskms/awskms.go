// Package awskms package
package awskms

//go:generate mockery --name client --inpackage --case underscore

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
)

var pluginDataRequiredFields = []string{"key_id", "region"}

type client interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
}

type signer struct {
	client client
	ctx    context.Context
	keyID  string
}

func (s *signer) Sign(payload []byte, _ interface{}) ([]byte, error) {
	h := crypto.SHA256.New()
	if _, err := h.Write(payload); err != nil {
		return nil, fmt.Errorf("failed to create hash for token payload %v", err)
	}

	input := kms.SignInput{
		KeyId:            &s.keyID,
		Message:          h.Sum(nil),
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
		MessageType:      types.MessageTypeDigest,
	}
	output, err := s.client.Sign(s.ctx, &input)
	if err != nil {
		return nil, err
	}
	return output.Signature, nil
}

func (s *signer) Algorithm() jwa.SignatureAlgorithm {
	return jwa.RS256
}

// JWSProvider uses AWS Asymmetric KMS key for signing tokens
type JWSProvider struct {
	client client
	keyID  string
	pubKey jwk.Key
	keySet []byte
}

// New creates an InMemoryJWSProvider
func New(ctx context.Context, pluginData map[string]string) (*JWSProvider, error) {
	return newPlugin(ctx, pluginData, clientBuilder, getPublicKey)
}

func newPlugin(
	ctx context.Context,
	pluginData map[string]string,
	clientBuilder func(ctx context.Context, region string) (client, error),
	publicKeyGetter func(context.Context, client, string) (jwk.Key, error),
) (*JWSProvider, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("AWS KMS JWS provider plugin requires plugin data '%s' field", field)
		}
	}

	c, err := clientBuilder(ctx, pluginData["region"])
	if err != nil {
		return nil, err
	}

	keyID := pluginData["key_id"]

	pubKey, err := publicKeyGetter(ctx, c, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from AWS KMS %v", err)
	}

	keySet, err := buildKeySet(pubKey)
	if err != nil {
		return nil, fmt.Errorf("failed to build JWK key set %v", err)
	}

	return &JWSProvider{
		client: c,
		pubKey: pubKey,
		keyID:  keyID,
		keySet: keySet,
	}, nil
}

// Sign signs a JWT payload
func (j *JWSProvider) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	hdrs := jws.NewHeaders()
	if err := hdrs.Set(jws.TypeKey, "JWT"); err != nil {
		return nil, err
	}
	if err := hdrs.Set(jws.KeyIDKey, j.pubKey.KeyID()); err != nil {
		return nil, err
	}

	sig := jws.NewSignature()
	sig.SetProtectedHeaders(hdrs)

	_, signedToken, err := sig.Sign(payload, &signer{client: j.client, keyID: j.keyID, ctx: ctx}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token using AWS KMS JWS provider %v", err)
	}

	return signedToken, nil
}

// GetKeySet returns the JWK key set in JSON format
func (j *JWSProvider) GetKeySet(_ context.Context) ([]byte, error) {
	return j.keySet, nil
}

// Verify will return an error if the JWT does not have a valid signature
func (j *JWSProvider) Verify(_ context.Context, token []byte) error {
	if _, err := jws.Verify(token, jwa.RS256, j.pubKey); err != nil {
		return err
	}

	return nil
}

func buildKeySet(pubKey jwk.Key) ([]byte, error) {
	keySet := jwk.NewSet()
	keySet.Add(pubKey)
	buf, err := json.Marshal(keySet)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func clientBuilder(ctx context.Context, region string) (client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return kms.NewFromConfig(awsCfg), nil
}

func getPublicKey(ctx context.Context, client client, keyID string) (jwk.Key, error) {
	output, err := client.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: &keyID})
	if err != nil {
		return nil, err
	}

	pubKey, err := x509.ParsePKIXPublicKey(output.PublicKey)
	if err != nil {
		return nil, err
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key type")
	}

	jwkPubKey, err := jwk.New(rsaPubKey)
	if err != nil {
		return nil, err
	}

	if err := jwk.AssignKeyID(jwkPubKey); err != nil {
		return nil, err
	}

	if err := jwkPubKey.Set(jwk.AlgorithmKey, jwa.RS256); err != nil {
		return nil, err
	}
	if err := jwkPubKey.Set(jwk.KeyUsageKey, jwk.ForSignature); err != nil {
		return nil, err
	}

	return jwkPubKey, nil
}
