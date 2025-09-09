package awsfederated

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

func TestSetManagedIdentityData(t *testing.T) {
	// Test cases
	tests := []struct {
		name         string
		expectRole   string
		expectErr    string
		existingData []byte
		inputData    []byte
	}{
		{
			name:         "New data payload",
			existingData: []byte{},
			inputData:    []byte(`{"role":"arn:role"}`),
			expectRole:   "arn:role",
		},
		{
			name:         "Update data payload",
			existingData: []byte(`{"role":"arn:originalRole", "subject": "TV9tYW5hZ2VkSWRlbnRpdHktMQ"}`),
			inputData:    []byte(`{"role":"arn:role"}`),
			expectRole:   "arn:role",
		},
		{
			name:      "Invalid data payload",
			inputData: []byte(`{"invalidField":"arn:role"}`),
			expectErr: "role field is missing from payload",
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

			delegate, err := New(ctx, auth.NewMockIdentityProvider(t))
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

				assert.Equal(t, test.expectRole, decodedData.Role)
				assert.Equal(t, managedIdentity.GetGlobalID(), decodedData.Subject)
			}
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockIDP := auth.NewMockIdentityProvider(t)

	matcher := mock.MatchedBy(func(input *auth.TokenInput) bool {
		return input.Subject == "sub-123" && input.Audience == "aws" && input.Expiration != nil
	})
	mockIDP.On("GenerateToken", ctx, matcher).Return([]byte("signedtoken"), nil)

	delegate, err := New(ctx, mockIDP)
	if err != nil {
		t.Fatal(err)
	}

	dataBuffer, err := json.Marshal(&Data{Role: "testrole", Subject: "sub-123"})
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

	assert.Equal(t, []byte("signedtoken"), payload)
}
