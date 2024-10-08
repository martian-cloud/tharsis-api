package jobexecutor

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

type authenticatorFactoryFunc func() (managedidentity.Authenticator, error)

type managedIdentities struct {
	client         Client
	jobLogger      *jobLogger
	factoryMap     map[types.ManagedIdentityType]authenticatorFactoryFunc
	workspaceID    string
	workspaceDir   string
	authenticators []managedidentity.Authenticator
}

func newManagedIdentities(
	workspaceID string,
	workspaceDir string,
	jobLogger *jobLogger,
	client Client,
) *managedIdentities {
	return &managedIdentities{
		workspaceID:    workspaceID,
		workspaceDir:   workspaceDir,
		jobLogger:      jobLogger,
		client:         client,
		authenticators: []managedidentity.Authenticator{},
		factoryMap: map[types.ManagedIdentityType]authenticatorFactoryFunc{
			types.ManagedIdentityAWSFederated: func() (managedidentity.Authenticator, error) {
				return awsfederated.New()
			},
			types.ManagedIdentityAzureFederated: func() (managedidentity.Authenticator, error) {
				return azurefederated.New()
			},
			types.ManagedIdentityTharsisFederated: func() (managedidentity.Authenticator, error) {
				return tharsisfederated.New(), nil
			},
		},
	}
}

func (l *managedIdentities) close(ctx context.Context) error {
	for _, authenticator := range l.authenticators {
		if err := authenticator.Close(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l *managedIdentities) initialize(ctx context.Context) (map[string]string, error) {
	allEnvVars := map[string]string{}

	identities, err := l.client.GetAssignedManagedIdentities(ctx, l.workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned managed identities for workspace %v", err)
	}

	identitiesMap := map[types.ManagedIdentityType][]types.ManagedIdentity{}
	for _, identity := range identities {
		identitiesMap[identity.Type] = append(identitiesMap[identity.Type], identity)
	}

	for identityType, identities := range identitiesMap {
		factoryFunc, ok := l.factoryMap[identityType]
		if !ok {
			return nil, fmt.Errorf("managed identity type %s is not supported", identityType)
		}

		authenticator, err := factoryFunc()
		if err != nil {
			return nil, fmt.Errorf("error creating authenticator: %s", err)
		}

		l.authenticators = append(l.authenticators, authenticator)

		credsRetriever := func(ctx context.Context, managedIdentity *types.ManagedIdentity) ([]byte, error) {
			l.jobLogger.Infof("Loading credentials for %s managed identity: %s\n", managedIdentity.Type, managedIdentity.ResourcePath)
			return l.client.CreateManagedIdentityCredentials(ctx, managedIdentity.Metadata.ID)
		}

		env, err := authenticator.Authenticate(ctx, identities, credsRetriever)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with managed identity %v", err)
		}

		for k, v := range env {
			allEnvVars[k] = v
		}
	}

	return allEnvVars, nil
}
