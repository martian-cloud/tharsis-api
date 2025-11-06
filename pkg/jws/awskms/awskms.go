// Package awskms package
package awskms

//go:generate go tool mockery --name kmsClient --inpackage --case underscore
//go:generate go tool mockery --name stsClient --inpackage --case underscore

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go/ptr"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"

	jwsplugin "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	defaultKeyAliasPrefix = "tharsis-signing-key"
	keyDeletionPeriodDays = 7
	defaultKeySpec        = types.KeySpecRsa2048
)

var pluginDataRequiredFields = []string{"region"}

type kmsClient interface {
	Sign(ctx context.Context, params *kms.SignInput, optFns ...func(*kms.Options)) (*kms.SignOutput, error)
	GetPublicKey(ctx context.Context, params *kms.GetPublicKeyInput, optFns ...func(*kms.Options)) (*kms.GetPublicKeyOutput, error)
	CreateKey(ctx context.Context, params *kms.CreateKeyInput, optFns ...func(*kms.Options)) (*kms.CreateKeyOutput, error)
	CreateAlias(ctx context.Context, params *kms.CreateAliasInput, optFns ...func(*kms.Options)) (*kms.CreateAliasOutput, error)
	ScheduleKeyDeletion(ctx context.Context, params *kms.ScheduleKeyDeletionInput, optFns ...func(*kms.Options)) (*kms.ScheduleKeyDeletionOutput, error)
	DescribeKey(ctx context.Context, params *kms.DescribeKeyInput, optFns ...func(*kms.Options)) (*kms.DescribeKeyOutput, error)
}

type stsClient interface {
	GetCallerIdentity(ctx context.Context, params *sts.GetCallerIdentityInput, optFns ...func(*sts.Options)) (*sts.GetCallerIdentityOutput, error)
}

type signer struct {
	kmsClient kmsClient
	ctx       context.Context
	keyID     string
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
	output, err := s.kmsClient.Sign(s.ctx, &input)
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
	kmsClient      kmsClient
	stsClient      stsClient
	logger         logger.Logger
	tags           []types.Tag
	keySpec        types.KeySpec
	keyAliasPrefix string
}

// New creates an InMemoryJWSProvider
func New(ctx context.Context, logger logger.Logger, pluginData map[string]string) (*JWSProvider, error) {
	return newPlugin(ctx, logger, pluginData, clientBuilder)
}

func newPlugin(
	ctx context.Context,
	logger logger.Logger,
	pluginData map[string]string,
	clientBuilder func(ctx context.Context, region string) (kmsClient, stsClient, error),
) (*JWSProvider, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("AWS KMS JWS provider plugin requires plugin data '%s' field", field)
		}
	}

	kms, sts, err := clientBuilder(ctx, pluginData["region"])
	if err != nil {
		return nil, err
	}

	tags := []types.Tag{}

	rawTags := pluginData["tags"]
	if rawTags != "" {
		tagPairs := strings.Split(rawTags, ",")
		for _, tagPair := range tagPairs {
			parts := strings.SplitN(tagPair, "=", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid tag format: %s", tagPair)
			}
			tags = append(tags, types.Tag{
				TagKey:   ptr.String(strings.TrimSpace(parts[0])),
				TagValue: ptr.String(strings.TrimSpace(parts[1])),
			})
		}
	}

	var keySpec = defaultKeySpec

	keySpecRaw := pluginData["key_spec"]
	if keySpecRaw != "" {
		keySpec = types.KeySpec(keySpecRaw)
		if !slices.Contains([]types.KeySpec{types.KeySpecRsa2048, types.KeySpecRsa3072, types.KeySpecRsa4096}, keySpec) {
			return nil, fmt.Errorf("invalid key spec for aws kms signing key: %q", keySpecRaw)
		}
	}

	keyAliasPrefix := pluginData["alias_prefix"]
	if keyAliasPrefix == "" {
		keyAliasPrefix = defaultKeyAliasPrefix
	}

	return &JWSProvider{
		logger:         logger,
		kmsClient:      kms,
		stsClient:      sts,
		tags:           tags,
		keySpec:        keySpec,
		keyAliasPrefix: keyAliasPrefix,
	}, nil
}

// SupportsKeyRotation indicates if the plugin supports key rotation
func (j *JWSProvider) SupportsKeyRotation() bool {
	return true
}

// Create creates a new signing key
func (j *JWSProvider) Create(ctx context.Context, keyID string) (*jwsplugin.CreateKeyResponse, error) {
	// Define key parameters
	keyUsage := types.KeyUsageTypeSignVerify
	description := "Asymmetric KMS key for signing and verifying"

	keyPolicy, err := j.createKeyPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create key policy: %v", err)
	}

	// Create the KMS key
	createKeyInput := &kms.CreateKeyInput{
		KeyUsage:    keyUsage,
		KeySpec:     j.keySpec,
		Description: &description,
		Policy:      keyPolicy,
		Tags:        j.tags,
	}

	createKeyOutput, err := j.kmsClient.CreateKey(ctx, createKeyInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create KMS key, %v", err)
	}

	// Create an alias for the KMS key
	createAliasInput := &kms.CreateAliasInput{
		AliasName:   ptr.String(j.buildAlias(keyID)),
		TargetKeyId: createKeyOutput.KeyMetadata.KeyId,
	}

	_, err = j.kmsClient.CreateAlias(ctx, createAliasInput)
	if err != nil {
		return nil, fmt.Errorf("failed to create alias for KMS key, %v", err)
	}

	pubKey, err := j.getPublicKey(ctx, createKeyOutput.KeyMetadata.KeyId)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from AWS KMS %v", err)
	}

	return &jwsplugin.CreateKeyResponse{
		PublicKey: pubKey,
	}, nil
}

// Delete schedules the deletion of a signing key
func (j *JWSProvider) Delete(ctx context.Context, keyID string, _ []byte) error {
	keyInfo, err := j.kmsClient.DescribeKey(ctx, &kms.DescribeKeyInput{
		KeyId: ptr.String(j.buildAlias(keyID)),
	})
	if err != nil {
		return fmt.Errorf("failed to describe KMS key: %v", err)
	}

	if _, err = j.kmsClient.ScheduleKeyDeletion(ctx, &kms.ScheduleKeyDeletionInput{
		KeyId:               keyInfo.KeyMetadata.KeyId,
		PendingWindowInDays: ptr.Int32(keyDeletionPeriodDays),
	}); err != nil {
		return fmt.Errorf("failed to schedule KMS key deletion: %v", err)
	}

	return nil
}

// Sign signs a JWT payload
func (j *JWSProvider) Sign(ctx context.Context, payload []byte, keyID string, _ []byte, publicKeyID string) ([]byte, error) {
	hdrs := jws.NewHeaders()
	if err := hdrs.Set(jws.TypeKey, "JWT"); err != nil {
		return nil, err
	}
	if err := hdrs.Set(jws.KeyIDKey, publicKeyID); err != nil {
		return nil, err
	}

	sig := jws.NewSignature()
	sig.SetProtectedHeaders(hdrs)

	_, signedToken, err := sig.Sign(payload, &signer{kmsClient: j.kmsClient, keyID: j.buildAlias(keyID), ctx: ctx}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token using AWS KMS JWS provider %w", err)
	}

	return signedToken, nil
}

func (j *JWSProvider) buildAlias(keyID string) string {
	return fmt.Sprintf("alias/%s-%s", j.keyAliasPrefix, keyID)
}

func (j *JWSProvider) createKeyPolicy(ctx context.Context) (*string, error) {
	callerIdentity, err := j.stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return nil, fmt.Errorf("cannot get caller identity: %w", err)
	}

	accountID := *callerIdentity.Account

	// Extract principal type and name (user or role) from ARN
	arnParts := strings.Split(*callerIdentity.Arn, ":")

	if len(arnParts) < 6 {
		return nil, fmt.Errorf("invalid ARN format for aws caller identity: %s", *callerIdentity.Arn)
	}

	// Extract partition from the caller's ARN (e.g., aws, aws-us-gov)
	partition := arnParts[1]
	principalType := arnParts[5] // e.g., user/Name or assumed-role/RoleName/SessionName
	nameParts := strings.Split(principalType, "/")

	var principalARN string

	if nameParts[0] == "assumed-role" {
		// For assumed roles, the format is assumed-role/RoleName/SessionName
		if len(nameParts) < 2 {
			return nil, fmt.Errorf("invalid assumed-role ARN format: %s", *callerIdentity.Arn)
		}

		principalARN = fmt.Sprintf("arn:%s:iam::%s:role/%s", partition, accountID, nameParts[1])
	} else {
		// For IAM user
		principalARN = fmt.Sprintf("arn:%s:iam::%s:user/%s", partition, accountID, nameParts[len(nameParts)-1])
	}

	// Define key policy with dynamic partition and account ID
	policy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":    "Enable root account access for viewing and deleting key",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"AWS": fmt.Sprintf("arn:%s:iam::%s:root", partition, accountID),
				},
				"Action": []string{
					"kms:Describe*",
					"kms:List*",
					"kms:Get*",
					"kms:ScheduleKeyDeletion",
					"kms:Delete*",
				},
				"Resource": "*",
			},
			{
				"Sid":    "Allow full access to tharsis service",
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"AWS": principalARN,
				},
				"Action":   "kms:*",
				"Resource": "*",
			},
		},
	}

	// Convert policy to JSON string
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal IAM key policy: %w", err)
	}
	return ptr.String(string(policyJSON)), nil
}

func clientBuilder(ctx context.Context, region string) (kmsClient, stsClient, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, nil, err
	}

	return kms.NewFromConfig(awsCfg), sts.NewFromConfig(awsCfg), nil
}

func (j *JWSProvider) getPublicKey(ctx context.Context, keyID *string) (jwk.Key, error) {
	output, err := j.kmsClient.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: keyID})
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

	jwkPubKey, err := jwk.FromRaw(rsaPubKey)
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
