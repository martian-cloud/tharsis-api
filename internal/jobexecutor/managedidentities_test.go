package jobexecutor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

func TestInitialize(t *testing.T) {
	const workspaceID = "workspaceID"
	const managedIdentityType = types.ManagedIdentityTharsisFederated

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobClient := jobclient.NewMockClient(t)

	identities := buildArrayOfManagedIdentities(managedIdentityType)

	jobClient.On("GetAssignedManagedIdentities", ctx, workspaceID).Return(identities, nil)

	authenticator := managedidentity.NewMockAuthenticator(t)

	authenticateResponse := managedidentity.AuthenticateResponse{
		Env: map[string]string{
			"TF_TOKEN_env": "test",
		},
		HostCredentialFileMapping: map[string]string{
			"localhost:6556": "token-file",
		},
	}
	authenticator.On("Authenticate", ctx, identities, mock.Anything).Return(&authenticateResponse, nil)

	managedIdentities := &managedIdentities{
		workspaceID:    workspaceID,
		jobLogger:      joblogger.NewMockJobLogger(t),
		client:         jobClient,
		authenticators: []managedidentity.Authenticator{},
		factoryMap:     buildFactoryMap(authenticator),
	}

	response, err := managedIdentities.initialize(ctx)

	assert.Nil(t, err)

	assert.Equal(t, authenticateResponse.Env, response.Env)
	assert.Equal(t, authenticateResponse.HostCredentialFileMapping, response.HostCredentialFileMapping)
}

func buildArrayOfManagedIdentities(managedIdentityType types.ManagedIdentityType) []types.ManagedIdentity {
	identities := []types.ManagedIdentity{
		{
			Metadata: types.ResourceMetadata{
				ID: "managedIdentity-1",
			},
			Type: managedIdentityType,
		},
	}
	return identities
}

func buildFactoryMap(authenticator managedidentity.Authenticator) map[types.ManagedIdentityType]authenticatorFactoryFunc {
	return map[types.ManagedIdentityType]authenticatorFactoryFunc{
		types.ManagedIdentityTharsisFederated: func() (managedidentity.Authenticator, error) {
			return authenticator, nil
		},
	}
}
