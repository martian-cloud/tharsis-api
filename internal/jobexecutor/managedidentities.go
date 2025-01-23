package jobexecutor

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/tharsisfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

type authenticatorFactoryFunc func() (managedidentity.Authenticator, error)

type managedIdentities struct {
	client         jobclient.Client
	jobLogger      joblogger.Logger
	factoryMap     map[types.ManagedIdentityType]authenticatorFactoryFunc
	workspaceID    string
	authenticators []managedidentity.Authenticator
}

type managedIdentityInitializeResponse struct {
	Env                       map[string]string
	HostCredentialFileMapping map[string]string
}

func newManagedIdentities(
	workspaceID string,
	jobLogger joblogger.Logger,
	client jobclient.Client,
	jobCfg *JobConfig,
) *managedIdentities {
	return &managedIdentities{
		workspaceID:    workspaceID,
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
				return tharsisfederated.New(client, jobLogger, jobCfg.APIEndpoint, &jobCfg.DiscoveryProtocolHost)
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

func (l *managedIdentities) initialize(ctx context.Context) (*managedIdentityInitializeResponse, error) {
	response := managedIdentityInitializeResponse{
		Env:                       map[string]string{},
		HostCredentialFileMapping: map[string]string{},
	}

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
			l.jobLogger.Infof("Loading credentials for %s managed identity: %s", managedIdentity.Type, managedIdentity.ResourcePath)
			return l.client.CreateManagedIdentityCredentials(ctx, managedIdentity.Metadata.ID)
		}

		authResponse, err := authenticator.Authenticate(ctx, identities, credsRetriever)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with managed identity %v", err)
		}

		for k, v := range authResponse.Env {
			response.Env[k] = v
		}

		for k, v := range authResponse.HostCredentialFileMapping {
			response.HostCredentialFileMapping[k] = v
		}
	}

	return &response, nil
}
