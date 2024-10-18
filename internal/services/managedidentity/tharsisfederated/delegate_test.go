package tharsisfederated

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	jwsprovider "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

func TestSetManagedIdentityData(t *testing.T) {
	// Test cases
	tests := []struct {
		name         string
		expectPath   string
		expectErr    string
		existingData []byte
		inputData    []byte
		expectHosts  []string
	}{
		{
			name:         "new data payload",
			existingData: []byte{},
			inputData:    []byte(`{"serviceAccountPath":"service/account/path", "hosts": ["example.com", "myotherdomain.com"]}`),
			expectPath:   "service/account/path",
			expectHosts:  []string{"example.com", "myotherdomain.com"},
		},
		{
			name:         "update data payload",
			existingData: []byte(`{"serviceAccountPath":"original/path", "subject": "TV9tYW5hZ2VkSWRlbnRpdHktMQ", "hosts": ["myotherdomain.com"]}`),
			inputData:    []byte(`{"serviceAccountPath":"updated/path", "hosts": ["updatedhost.com"]}`),
			expectPath:   "updated/path",
			expectHosts:  []string{"updatedhost.com"},
		},
		{
			name:      "invalid data payload",
			inputData: []byte(`{"invalidField":"invalid/field/value"}`),
			expectErr: "service account path field is missing from payload",
		},
		{
			name:      "empty data payload",
			inputData: []byte(""),
			expectErr: "invalid managed identity data: unexpected end of JSON input",
		},
		{
			name:      "invalid hosts",
			inputData: []byte(`{"serviceAccountPath":"service/account/path", "hosts": ["invalid~.com"]}`),
			expectErr: "invalid hosts: ['invalid~.com': domain has invalid character '~' at offset 7]",
		},
		{
			name:      "duplicate hosts",
			inputData: []byte(`{"serviceAccountPath":"service/account/path", "hosts": ["gooGle.com", "google.com"]}`),
			expectErr: "invalid hosts: ['google.com': has already been specified]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			delegate, err := New(ctx, &jwsprovider.MockProvider{}, "http://test")
			if err != nil {
				t.Fatal(err)
			}

			managedIdentity := &models.ManagedIdentity{
				Metadata: models.ResourceMetadata{
					ID: "managedIdentity-1",
				},
			}

			if test.existingData != nil {
				managedIdentity.Data = []byte(base64.StdEncoding.EncodeToString(test.existingData))
			}

			err = delegate.SetManagedIdentityData(
				ctx,
				managedIdentity,
				[]byte(base64.StdEncoding.EncodeToString(test.inputData)),
			)

			if test.expectErr != "" {
				assert.EqualError(t, err, test.expectErr)
			} else if err != nil {
				t.Fatal(err)
			} else {

				decodedData, err := decodeData(managedIdentity.Data)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, test.expectPath, decodedData.ServiceAccountPath)
				assert.Equal(t, test.expectHosts, decodedData.Hosts)
				assert.Equal(t, gid.ToGlobalID(gid.ManagedIdentityType, managedIdentity.Metadata.ID), decodedData.Subject)
			}
		})
	}
}

func TestValidateHostWithPort(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		host           string
		expectMessages []string
	}{
		{
			name: "valid host",
			host: "example.com",
		},
		{
			name: "valid host",
			host: "myhost",
		},
		{
			name: "host with port 0",
			host: "example.com:0",
		},
		{
			name: "host with port",
			host: "example.com:8080",
		},
		{
			name: "host with port 65535",
			host: "example.com:65535",
		},
		{
			name:           "invalid host",
			host:           "invalid~.com",
			expectMessages: []string{"'invalid~.com': domain has invalid character '~' at offset 7"},
		},
		{
			name:           "host with missing port",
			host:           "example.com:",
			expectMessages: []string{"'example.com:': port expected"},
		},
		{
			name:           "host with decimal port",
			host:           "example.com:86.92",
			expectMessages: []string{"'example.com:86.92': invalid port, port must be a valid integer"},
		},
		{
			name:           "host with invalid port",
			host:           "example.com:InvalidPort",
			expectMessages: []string{"'example.com:InvalidPort': invalid port, port must be a valid integer"},
		},
		{
			name:           "host with negative port",
			host:           "example.com:-1",
			expectMessages: []string{"'example.com:-1': invalid port, port must be between 0 and 65535"},
		},
		{
			name:           "host with port above range",
			host:           "example.com:65536",
			expectMessages: []string{"'example.com:65536': invalid port, port must be between 0 and 65535"},
		},
		{
			name: "host with compound errors",
			host: "inv~valid:-85.6",
			expectMessages: []string{
				"'inv~valid': domain has invalid character '~' at offset 3",
				"'inv~valid:-85.6': invalid port, port must be a valid integer",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			messages := validateHostWithPort(test.host)

			assert.Len(t, messages, len(test.expectMessages))

			for _, expected := range test.expectMessages {
				assert.Contains(t, messages, expected)
			}
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	issuer := "http://test"

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	mockJWSProvider := jwsprovider.MockProvider{}
	mockJWSProvider.Test(t)

	mockJWSProvider.On("Sign", ctx, mock.Anything).Return(func(_ context.Context, token []byte) []byte {
		signed, sErr := jws.Sign(token, jws.WithKey(jwa.RS256, privKey))
		if sErr != nil {
			t.Fatal(sErr)
		}
		return signed
	}, nil)

	delegate, err := New(ctx, &mockJWSProvider, issuer)
	if err != nil {
		t.Fatal(err)
	}

	dataBuffer, err := json.Marshal(&Data{ServiceAccountPath: "test/path", Subject: "sub-123"})
	if err != nil {
		t.Fatal(err)
	}

	identity := models.ManagedIdentity{
		Metadata: models.ResourceMetadata{
			ID: "managedIdentity-1",
		},
		Data: []byte(base64.StdEncoding.EncodeToString(dataBuffer)),
	}
	job := models.Job{
		Metadata: models.ResourceMetadata{
			ID: "job-1",
		},
		MaxJobDuration: 720,
	}

	payload, err := delegate.CreateCredentials(ctx, &identity, &job)
	if err != nil {
		t.Fatal(err)
	}

	parsedToken, err := jwt.Parse(payload, jwt.WithVerify(false))
	if err != nil {
		t.Fatal(err)
	}

	maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute
	assert.Equal(t, parsedToken.Subject(), "sub-123")
	assert.Equal(t, parsedToken.Issuer(), issuer)
	assert.Equal(t, parsedToken.Audience(), []string{"tharsis"})
	assert.True(t, parsedToken.Expiration().After(time.Now()) && parsedToken.Expiration().Before(time.Now().Add(maxJobDuration)))
}
