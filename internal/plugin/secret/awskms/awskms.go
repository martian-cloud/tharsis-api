// Package awskms package
package awskms

//go:generate mockery --name client --inpackage --case underscore

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/secret"
)

type client interface {
	Encrypt(ctx context.Context, params *kms.EncryptInput, optFns ...func(*kms.Options)) (*kms.EncryptOutput, error)
	Decrypt(ctx context.Context, params *kms.DecryptInput, optFns ...func(*kms.Options)) (*kms.DecryptOutput, error)
}

var (
	pluginDataRequiredFields = []string{"key_id", "region"}
)

// secretManager uses AWS KMS key for encrypting secrets
type secretManager struct {
	client client
	keyID  string
}

// New creates an AWS KMS secret manager plugin which encrypts secrets using the KMS key
func New(ctx context.Context, pluginData map[string]string) (secret.Manager, error) {
	return newPlugin(ctx, pluginData, clientBuilder)
}

func newPlugin(
	ctx context.Context,
	pluginData map[string]string,
	clientBuilder func(ctx context.Context, region string) (client, error),
) (*secretManager, error) {
	for _, field := range pluginDataRequiredFields {
		if _, ok := pluginData[field]; !ok {
			return nil, fmt.Errorf("AWS KMS secret manager plugin requires plugin data %q field", field)
		}
	}

	// Create KMS client
	c, err := clientBuilder(ctx, pluginData["region"])
	if err != nil {
		return nil, err
	}

	return &secretManager{
		client: c,
		keyID:  pluginData["key_id"],
	}, nil
}

// Create encrypts the value using the KMS key and returns the encrypted value
func (s *secretManager) Create(ctx context.Context, key string, value string) ([]byte, error) {
	// Encrypt value using kms key
	input := kms.EncryptInput{
		KeyId:             &s.keyID,
		Plaintext:         []byte(value),
		EncryptionContext: createEncryptionContext(key),
	}

	output, err := s.client.Encrypt(ctx, &input)
	if err != nil {
		return nil, fmt.Errorf("aws kms secret manager failed to encrypt secret key %q: %w", key, err)
	}

	return output.CiphertextBlob, nil
}

// Update encrypts the new value using the KMS key
func (s *secretManager) Update(ctx context.Context, key string, _ []byte, newValue string) ([]byte, error) {
	// Delegate to create since this plugin doesn't need to update any existing values
	return s.Create(ctx, key, newValue)
}

// Get decrypts the secret using the KMS key
func (s *secretManager) Get(ctx context.Context, key string, secret []byte) (string, error) {
	// Decrypt secret using kms key
	input := kms.DecryptInput{
		KeyId:             &s.keyID,
		CiphertextBlob:    secret,
		EncryptionContext: createEncryptionContext(key),
	}
	output, err := s.client.Decrypt(ctx, &input)
	if err != nil {
		return "", fmt.Errorf("aws kms secret manager failed to deccrypt secret key %q: %w", key, err)
	}
	return string(output.Plaintext), nil
}

func clientBuilder(ctx context.Context, region string) (client, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	return kms.NewFromConfig(awsCfg), nil
}

// createEncryptionContext uses the app name and variable name for additional security when encrypting secrets
func createEncryptionContext(varName string) map[string]string {
	return map[string]string{
		"app":           "tharsis",
		"variable_name": varName,
	}
}
