package jobexecutor

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

type authenticatorFactoryFunc func() (managedidentity.Authenticator, error)

type managedIdentities struct {
	client         Client
	jobLogger      *jobLogger
	factoryMap     map[types.ManagedIdentityType]authenticatorFactoryFunc
	workspacePath  string
	workspaceDir   string
	authenticators []managedidentity.Authenticator
}

func newManagedIdentities(
	workspacePath string,
	workspaceDir string,
	jobLogger *jobLogger,
	client Client,
) *managedIdentities {
	return &managedIdentities{
		workspacePath:  workspacePath,
		workspaceDir:   workspaceDir,
		jobLogger:      jobLogger,
		client:         client,
		authenticators: []managedidentity.Authenticator{},
		factoryMap: map[types.ManagedIdentityType]authenticatorFactoryFunc{
			types.ManagedIdentityAWSFederated: func() (managedidentity.Authenticator, error) {
				return awsfederated.New()
			},
			types.ManagedIdentityAzureFederated: func() (managedidentity.Authenticator, error) {
				return azurefederated.New(jobLogger), nil
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

	identities, err := l.client.GetAssignedManagedIdentities(ctx, l.workspacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned managed identities for workspace %v", err)
	}

	for _, identity := range identities {
		l.jobLogger.Infof("Loading credentials for %s managed identity: %s\n", identity.Type, identity.ResourcePath)

		creds, err := l.client.CreateManagedIdentityCredentials(ctx, identity.Metadata.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to create managed identity credentials %v", err)
		}

		factoryFunc, ok := l.factoryMap[identity.Type]
		if !ok {
			return nil, fmt.Errorf("managed identity type %s is not supported", identity.Type)
		}

		authenticator, err := factoryFunc()
		if err != nil {
			return nil, fmt.Errorf("error creating authenticator: %s", err)
		}

		l.authenticators = append(l.authenticators, authenticator)
		id := identity

		env, err := authenticator.Authenticate(ctx, &id, creds)
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate with managed identity %v", err)
		}

		for k, v := range env {
			allEnvVars[k] = v
		}
	}

	return allEnvVars, nil
}
