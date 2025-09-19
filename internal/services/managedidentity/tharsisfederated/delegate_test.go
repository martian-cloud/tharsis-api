package tharsisfederated

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
		name                                   string
		expectPath                             string
		expectErr                              string
		existingData                           []byte
		inputData                              []byte
		expectUseServiceAccountForTerraformCLI bool
	}{
		{
			name:                                   "new data payload",
			existingData:                           []byte{},
			inputData:                              []byte(`{"serviceAccountPath":"service/account/path"}`),
			expectPath:                             "service/account/path",
			expectUseServiceAccountForTerraformCLI: false,
		},
		{
			name:                                   "new data payload",
			existingData:                           []byte{},
			inputData:                              []byte(`{"serviceAccountPath":"service/account/path", "useServiceAccountForTerraformCLI": true}`),
			expectPath:                             "service/account/path",
			expectUseServiceAccountForTerraformCLI: true,
		},
		{
			name:                                   "update data payload",
			existingData:                           []byte(`{"serviceAccountPath":"original/path", "subject": "TV9tYW5hZ2VkSWRlbnRpdHktMQ", "useServiceAccountForTerraformCLI": true}`),
			inputData:                              []byte(`{"serviceAccountPath":"updated/path", "useServiceAccountForTerraformCLI": false}`),
			expectPath:                             "updated/path",
			expectUseServiceAccountForTerraformCLI: false,
		},
		{
			name:                                   "update data payload with useServiceAccountForTerraformCLI omitted",
			existingData:                           []byte(`{"serviceAccountPath":"original/path", "subject": "TV9tYW5hZ2VkSWRlbnRpdHktMQ"}`),
			inputData:                              []byte(`{"serviceAccountPath":"updated/path", "useServiceAccountForTerraformCLI": true}`),
			expectPath:                             "updated/path",
			expectUseServiceAccountForTerraformCLI: true,
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

				assert.Equal(t, test.expectPath, decodedData.ServiceAccountPath)
				assert.Equal(t, test.expectUseServiceAccountForTerraformCLI, decodedData.UseServiceAccountForTerraformCLI)
				assert.Equal(t, managedIdentity.GetGlobalID(), decodedData.Subject)
			}
		})
	}
}

func TestCreateCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockIDP := auth.NewMockSigningKeyManager(t)

	matcher := mock.MatchedBy(func(input *auth.TokenInput) bool {
		return input.Subject == "sub-123" && input.Expiration != nil
	})
	mockIDP.On("GenerateToken", ctx, matcher).Return([]byte("signedtoken"), nil)

	delegate, err := New(ctx, mockIDP)
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

	assert.Equal(t, []byte("signedtoken"), payload)
}
