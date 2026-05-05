package jobexecutor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

func TestInitialize(t *testing.T) {
	const workspaceID = "workspaceID"
	var managedIdentityType = pb.ManagedIdentityType_tharsis_federated.String()

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
		jobLogger:      joblogger.NewMockLogger(t),
		client:         jobClient,
		authenticators: []managedidentity.Authenticator{},
		factoryMap:     buildFactoryMap(authenticator),
	}

	response, err := managedIdentities.initialize(ctx)

	assert.Nil(t, err)

	assert.Equal(t, authenticateResponse.Env, response.Env)
	assert.Equal(t, authenticateResponse.HostCredentialFileMapping, response.HostCredentialFileMapping)
}

func buildArrayOfManagedIdentities(managedIdentityType string) []*pb.ManagedIdentity {
	identities := []*pb.ManagedIdentity{
		{
			Metadata: &pb.ResourceMetadata{
				Id: "managedIdentity-1",
			},
			Type: managedIdentityType,
		},
	}
	return identities
}

func buildFactoryMap(authenticator managedidentity.Authenticator) map[string]authenticatorFactoryFunc {
	return map[string]authenticatorFactoryFunc{
		pb.ManagedIdentityType_tharsis_federated.String(): func() (managedidentity.Authenticator, error) {
			return authenticator, nil
		},
	}
}
