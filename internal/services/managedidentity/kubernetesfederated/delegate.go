// Package kubernetes package
package kubernetesfederated

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	maxAudienceLength = 100
)

// InputData contains the input data fields specific to this managed identity type
type InputData struct {
	Audience string `json:"audience"` // Required: audience for JWT token (e.g., "kubernetes")
}

// Data contains the data fields specific to this managed identity type
type Data struct {
	Audience string `json:"audience"`
}

// Delegate for the Kubernetes OIDC managed identity type
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
	// Parse managed identity data to get audience
	kubernetesData, err := decodeData(identity.Data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse managed identity data", errors.WithErrorCode(errors.EInvalid))
	}
	maxJobDuration := time.Duration(job.MaxJobDuration) * time.Minute

	return d.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Subject:    identity.GetGlobalID(), // Use managed identity ID as subject
		Expiration: ptr.Time(time.Now().Add(maxJobDuration)),
		Audience:   kubernetesData.Audience, // Use user-configured audience
		Claims: map[string]string{
			"job_id": job.GetGlobalID(),
		},
	})
}

// SetManagedIdentityData updates the managed identity custom data payload
func (d *Delegate) SetManagedIdentityData(_ context.Context, managedIdentity *models.ManagedIdentity, input []byte) error {
	decodedData, err := base64.StdEncoding.DecodeString(string(input))
	if err != nil {
		return errors.Wrap(err, "failed to decode managed identity data", errors.WithErrorCode(errors.EInvalid))
	}

	var inputData *InputData

	// Check if we have meaningful data to parse
	trimmedData := strings.TrimSpace(string(decodedData))
	if len(trimmedData) > 0 {
		inputData = &InputData{}
		if err = json.Unmarshal([]byte(trimmedData), inputData); err != nil {
			return errors.Wrap(err, "invalid managed identity data format", errors.WithErrorCode(errors.EInvalid))
		}
		if inputData.Audience == "" {
			return errors.New("audience field is missing from payload")
		}
	} else {
		return errors.New("managed identity data is required", errors.WithErrorCode(errors.EInvalid))
	}

	// Validate audience string
	if len(inputData.Audience) > maxAudienceLength {
		return errors.New(fmt.Sprintf("audience string too long, maximum %d characters allowed", maxAudienceLength), errors.WithErrorCode(errors.EInvalid))
	}

	if strings.Contains(inputData.Audience, " ") {
		return errors.New("audience string cannot contain spaces", errors.WithErrorCode(errors.EInvalid))
	}

	var kubernetesData *Data

	if managedIdentity.Data == nil {
		kubernetesData = &Data{}
	} else {
		kubernetesData, err = decodeData(managedIdentity.Data)
		if err != nil {
			return err
		}
	}

	kubernetesData.Audience = inputData.Audience

	buffer, err := json.Marshal(kubernetesData)
	if err != nil {
		return errors.Wrap(err, "failed to marshal updated managed identity data", errors.WithErrorCode(errors.EInternal))
	}

	managedIdentity.Data = []byte(base64.StdEncoding.EncodeToString(buffer))

	return nil
}

func decodeData(data []byte) (*Data, error) {
	// Handle empty data
	if len(data) == 0 {
		return &Data{}, nil
	}
	// Try to decode as base64 first (new format)
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	// Handle empty decoded data
	if len(strings.TrimSpace(string(decodedData))) == 0 {
		return &Data{}, nil
	}

	kubernetesData := Data{}
	if err := json.Unmarshal(decodedData, &kubernetesData); err != nil {
		return nil, err
	}

	return &kubernetesData, nil
}
