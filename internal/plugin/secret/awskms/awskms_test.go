package awskms

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/stretchr/testify/assert"
)

func TestNewPlugin(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	keyID := "123"

	c := newMockClient(t)

	clientBuilder := func(_ context.Context, _ string) (client, error) {
		return c, nil
	}

	plugin, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": keyID,
		},
		clientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, plugin.client)
	assert.NotNil(t, plugin.keyID)
}

func TestNewPluginWithMissingConfig(t *testing.T) {
	_, err := newPlugin(
		context.Background(),
		map[string]string{},
		nil,
	)
	if err == nil {
		t.Fatal("Expected error")
	}
	assert.Contains(t, err.Error(), "AWS KMS secret manager plugin requires plugin data")
}

func TestCreate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kmsKeyID := "test-key"
	variableName := "foo"
	variableValue := "bar"

	c := newMockClient(t)

	c.On("Encrypt", ctx, &kms.EncryptInput{
		KeyId:     &kmsKeyID,
		Plaintext: []byte(variableValue),
		EncryptionContext: map[string]string{
			"app":           "tharsis",
			"variable_name": variableName,
		},
	}).Return(&kms.EncryptOutput{
		CiphertextBlob: []byte("encrypted-foo-value"),
	}, nil)

	mockClientBuilder := func(_ context.Context, _ string) (client, error) {
		return c, nil
	}

	plugin, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": kmsKeyID,
		},
		mockClientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	secretData, err := plugin.Create(ctx, variableName, variableValue)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte("encrypted-foo-value"), secretData)
}

func TestUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kmsKeyID := "test-key"
	variableName := "foo"
	variableValue := "bar"

	c := newMockClient(t)

	c.On("Encrypt", ctx, &kms.EncryptInput{
		KeyId:     &kmsKeyID,
		Plaintext: []byte(variableValue),
		EncryptionContext: map[string]string{
			"app":           "tharsis",
			"variable_name": variableName,
		},
	}).Return(&kms.EncryptOutput{
		CiphertextBlob: []byte("encrypted-foo-value"),
	}, nil)

	mockClientBuilder := func(_ context.Context, _ string) (client, error) {
		return c, nil
	}

	plugin, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": kmsKeyID,
		},
		mockClientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	secretData, err := plugin.Update(ctx, variableName, []byte("old-value"), variableValue)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte("encrypted-foo-value"), secretData)
}

func TestGet(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kmsKeyID := "test-key"
	variableName := "foo"
	variableValue := "bar"
	variableSecretData := []byte("encrypted-foo-value")

	c := newMockClient(t)

	c.On("Decrypt", ctx, &kms.DecryptInput{
		KeyId:          &kmsKeyID,
		CiphertextBlob: variableSecretData,
		EncryptionContext: map[string]string{
			"app":           "tharsis",
			"variable_name": variableName,
		},
	}).Return(&kms.DecryptOutput{
		Plaintext: []byte(variableValue),
	}, nil)

	mockClientBuilder := func(_ context.Context, _ string) (client, error) {
		return c, nil
	}

	plugin, err := newPlugin(
		ctx,
		map[string]string{
			"region": "us-east-1",
			"key_id": kmsKeyID,
		},
		mockClientBuilder,
	)
	if err != nil {
		t.Fatal(err)
	}

	decryptedValue, err := plugin.Get(ctx, variableName, variableSecretData)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, variableValue, decryptedValue)
}
