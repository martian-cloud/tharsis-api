package azurefederated

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
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	jwsprovider "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/jws"
)

func TestValidateManagedIdentityData(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		expectClientID string
		expectTenantID string
		expectErr      string
		existingData   []byte
		inputData      []byte
	}{
		{
			name:           "New data payload",
			existingData:   []byte{},
			inputData:      []byte(`{"clientId":"client1", "tenantId":"tenant1"}`),
			expectClientID: "client1",
			expectTenantID: "tenant1",
		},
		{
			name:           "Update data payload",
			existingData:   []byte(`{"clientId":"oldclient", "tenantId":"oldtenant", "subject": "TV9tYW5hZ2VkSWRlbnRpdHktMQ"}`),
			inputData:      []byte(`{"clientId":"client1", "tenantId":"tenant1"}`),
			expectClientID: "client1",
			expectTenantID: "tenant1",
		},
		{
			name:      "Invalid data payload",
			inputData: []byte(`{"invalidField":"123"}`),
			expectErr: "clientId field is missing from payload",
		},
		{
			name:      "Empty data payload",
			inputData: []byte(""),
			expectErr: "invalid managed identity data: unexpected end of JSON input",
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

				assert.Equal(t, test.expectClientID, decodedData.ClientID)
				assert.Equal(t, test.expectTenantID, decodedData.TenantID)
				assert.Equal(t, managedIdentity.GetGlobalID(), decodedData.Subject)
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

	dataBuffer, err := json.Marshal(&Data{TenantID: "tenant1", ClientID: "client1", Subject: "sub-123"})
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
	assert.Equal(t, parsedToken.Audience(), []string{"azure"})
	assert.True(t, parsedToken.Expiration().After(time.Now()) && parsedToken.Expiration().Before(time.Now().Add(maxJobDuration)))
}
