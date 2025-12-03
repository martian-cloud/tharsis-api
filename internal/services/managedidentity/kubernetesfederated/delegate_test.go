package kubernetesfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
)

func TestValidateManagedIdentityData(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		expectAudience string
		expectErr      string
		existingData   []byte
		inputData      []byte
	}{
		{
			name:           "New data payload",
			inputData:      []byte(`{"audience":"kubernetes"}`),
			expectAudience: "kubernetes",
		},
		{
			name:           "Update data payload",
			existingData:   []byte(`{"audience":"kubernetes"}`),
			inputData:      []byte(`{"audience":"test_kube"}`),
			expectAudience: "test_kube",
		},
		{
			name:      "Invalid data payload",
			inputData: []byte(`{"invalidField":"123"}`),
			expectErr: "audience field is missing from payload",
		},
		{
			name:      "Empty data payload",
			inputData: []byte(""),
			expectErr: "managed identity data is required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			delegate, err := New(ctx, auth.NewMockSigningKeyManager(t))
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
				assert.Equal(t, test.expectAudience, decodedData.Audience)
			}
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockIDP := auth.NewMockSigningKeyManager(t)
	mockIDP.On("GenerateToken", mock.Anything, mock.Anything).Return([]byte("signedtokenforkubenetes"), nil)

	delegate, err := New(ctx, mockIDP)
	if err != nil {
		t.Fatal(err)
	}

	dataBuffer, err := json.Marshal(&Data{Audience: "kubernetes"})
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

	assert.Equal(t, []byte("signedtokenforkubenetes"), payload)
}
