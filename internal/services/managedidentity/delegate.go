package managedidentity

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/apiserver/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/managedidentity/azurefederated"
)

// Delegate handles the logic for a specific type of managed identity
type Delegate interface {
	CreateCredentials(ctx context.Context, identity *models.ManagedIdentity, job *models.Job) ([]byte, error)
	SetManagedIdentityData(ctx context.Context, managedIdentity *models.ManagedIdentity, input []byte) error
}

// NewManagedIdentityDelegateMap creates a map containing a delegate for each managed identity type
func NewManagedIdentityDelegateMap(ctx context.Context, cfg *config.Config, pluginCatalog *plugin.Catalog) (map[models.ManagedIdentityType]Delegate, error) {
	azureHandler, err := azurefederated.New(ctx, pluginCatalog.JWSProvider, cfg.ServiceAccountIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s managed identity handler %v", models.ManagedIdentityAzureFederated, err)
	}
	awsHandler, err := awsfederated.New(ctx, pluginCatalog.JWSProvider, cfg.ServiceAccountIssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize %s managed identity handler %v", models.ManagedIdentityAWSFederated, err)
	}

	return map[models.ManagedIdentityType]Delegate{
		models.ManagedIdentityAzureFederated: azureHandler,
		models.ManagedIdentityAWSFederated:   awsHandler,
	}, nil
}
