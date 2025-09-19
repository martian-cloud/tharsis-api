// Package managedidentity package
package managedidentity

//go:generate go tool mockery --name Delegate --inpackage --case underscore

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/tharsisfederated"
)

// Delegate handles the logic for a specific type of managed identity
type Delegate interface {
	CreateCredentials(ctx context.Context, identity *models.ManagedIdentity, job *models.Job) ([]byte, error)
	SetManagedIdentityData(ctx context.Context, managedIdentity *models.ManagedIdentity, input []byte) error
}

// NewManagedIdentityDelegateMap creates a map containing a delegate for each managed identity type
func NewManagedIdentityDelegateMap(ctx context.Context, signingKeyManager auth.SigningKeyManager) (map[models.ManagedIdentityType]Delegate, error) {
	azureHandler, err := azurefederated.New(ctx, signingKeyManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s managed identity handler %v", models.ManagedIdentityAzureFederated, err)
	}
	awsHandler, err := awsfederated.New(ctx, signingKeyManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s managed identity handler %v", models.ManagedIdentityAWSFederated, err)
	}
	tharsisHandler, err := tharsisfederated.New(ctx, signingKeyManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s managed identity handler %v", models.ManagedIdentityTharsisFederated, err)
	}

	return map[models.ManagedIdentityType]Delegate{
		models.ManagedIdentityAzureFederated:   azureHandler,
		models.ManagedIdentityAWSFederated:     awsHandler,
		models.ManagedIdentityTharsisFederated: tharsisHandler,
	}, nil
}
