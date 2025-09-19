// Package azurefederated package
package azurefederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// Data contains the data fields specific to this managed identity type
type Data struct {
	Subject  string `json:"subject"`
	ClientID string `json:"clientId"`
	TenantID string `json:"tenantId"`
}

// InputData contains the input data fields specific to this managed identity type
type InputData struct {
	ClientID string `json:"clientId"`
	TenantID string `json:"tenantId"`
}

// Delegate for the Azure OIDC Federated managed identity type
type Delegate struct {
	signingKeyManager auth.SigningKeyManager
}

// New creates a new Delegate instance
func New(_ context.Context, signingKeyManager auth.SigningKeyManager) (*Delegate, error) {
	return &Delegate{
		signingKeyManager: signingKeyManager,
	}, nil
}

// CreateCredentials returns a signed JWT token for the managed identity
func (d *Delegate) CreateCredentials(ctx context.Context, identity *models.ManagedIdentity, job *models.Job) ([]byte, error) {
	federatedData, err := decodeData(identity.Data)
	if err != nil {
		return nil, nil
	}

	maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute

	return d.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Subject:    federatedData.Subject,
		Expiration: ptr.Time(time.Now().Add(maxJobDuration)),
		Audience:   "azure",
		Claims: map[string]string{
			"job_id": job.GetGlobalID(),
		},
	})
}

// SetManagedIdentityData updates the managed identity custom data payload
func (d *Delegate) SetManagedIdentityData(_ context.Context, managedIdentity *models.ManagedIdentity, input []byte) error {
	decodedData, err := base64.StdEncoding.DecodeString(string(input))
	if err != nil {
		return te.Wrap(err, "failed to decode managed identity data", te.WithErrorCode(te.EInvalid))
	}

	inputData := Data{}
	if err = json.Unmarshal(decodedData, &inputData); err != nil {
		return te.Wrap(err, "invalid managed identity data", te.WithErrorCode(te.EInvalid))
	}

	if inputData.ClientID == "" {
		return errors.New("clientId field is missing from payload")
	}

	if inputData.TenantID == "" {
		return errors.New("tenantId field is missing from payload")
	}

	var federatedData *Data

	if managedIdentity.Data == nil || len(managedIdentity.Data) == 0 {
		federatedData = &Data{
			Subject: managedIdentity.GetGlobalID(),
		}
	} else {
		federatedData, err = decodeData(managedIdentity.Data)
		if err != nil {
			return err
		}
	}

	federatedData.ClientID = inputData.ClientID
	federatedData.TenantID = inputData.TenantID

	buffer, err := json.Marshal(&federatedData)
	if err != nil {
		return err
	}

	managedIdentity.Data = []byte(base64.StdEncoding.EncodeToString(buffer))

	return nil
}

func decodeData(data []byte) (*Data, error) {
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	federatedData := Data{}
	if err := json.Unmarshal(decodedData, &federatedData); err != nil {
		return nil, err
	}

	return &federatedData, nil
}
