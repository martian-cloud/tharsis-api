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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestInitialize(t *testing.T) {
	var managedIdentityType = pb.ManagedIdentityType_tharsis_federated.String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobClient := jobclient.NewMockClient(t)

	identities := buildArrayOfManagedIdentities(managedIdentityType)

	// Only the identities authorized on the job's run node are loaded.
	jobClient.On("GetManagedIdentity", ctx, "managedIdentity-1").Return(identities[0], nil)

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
		managedIdentityIDs: []string{"managedIdentity-1"},
		jobLogger:          joblogger.NewMockLogger(t),
		client:             jobClient,
		authenticators:     []managedidentity.Authenticator{},
		factoryMap:         buildFactoryMap(authenticator),
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

func TestInitialize_NoAuthorizedIdentities(t *testing.T) {
	// The mock client fails the test on any call, so no identity is looked up.
	managedIdentities := &managedIdentities{
		jobLogger:      joblogger.NewMockLogger(t),
		client:         jobclient.NewMockClient(t),
		authenticators: []managedidentity.Authenticator{},
		factoryMap:     buildFactoryMap(managedidentity.NewMockAuthenticator(t)),
	}

	response, err := managedIdentities.initialize(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, response.Env)
	assert.Empty(t, response.HostCredentialFileMapping)
}

func TestJobManagedIdentityIDs(t *testing.T) {
	run := &pb.Run{
		Plan:  &pb.Plan{ManagedIdentityIds: []string{"plan-mi"}},
		Apply: &pb.Apply{ManagedIdentityIds: []string{"apply-mi"}},
	}

	assert.Equal(t, []string{"plan-mi"}, jobManagedIdentityIDs(run, &pb.Job{Type: "plan"}))
	assert.Equal(t, []string{"apply-mi"}, jobManagedIdentityIDs(run, &pb.Job{Type: "apply"}))
	assert.Nil(t, jobManagedIdentityIDs(run, &pb.Job{Type: "opa"}))
	// A speculative run has no apply node.
	assert.Nil(t, jobManagedIdentityIDs(&pb.Run{Plan: &pb.Plan{}}, &pb.Job{Type: "apply"}))
}

func TestInitialize_DeletedIdentityIsSkipped(t *testing.T) {
	var managedIdentityType = pb.ManagedIdentityType_tharsis_federated.String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobClient := jobclient.NewMockClient(t)
	identities := buildArrayOfManagedIdentities(managedIdentityType)

	// managedIdentity-2 was deleted after the run node was authorized to use it.
	jobClient.On("GetManagedIdentity", ctx, "managedIdentity-1").Return(identities[0], nil)
	jobClient.On("GetManagedIdentity", ctx, "managedIdentity-2").Return(nil, status.Error(codes.NotFound, "not found"))

	jobLogger := joblogger.NewMockLogger(t)
	jobLogger.On("Warningf", mock.Anything, "managedIdentity-2").Once()

	authenticator := managedidentity.NewMockAuthenticator(t)
	authenticator.On("Authenticate", ctx, identities, mock.Anything).Return(&managedidentity.AuthenticateResponse{
		Env: map[string]string{"TF_TOKEN_env": "test"},
	}, nil)

	managedIdentities := &managedIdentities{
		managedIdentityIDs: []string{"managedIdentity-1", "managedIdentity-2"},
		jobLogger:          jobLogger,
		client:             jobClient,
		authenticators:     []managedidentity.Authenticator{},
		factoryMap:         buildFactoryMap(authenticator),
	}

	response, err := managedIdentities.initialize(ctx)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"TF_TOKEN_env": "test"}, response.Env)
}

func TestInitialize_LookupErrorFailsJob(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Errors other than not-found still fail the job rather than running without its credentials.
	jobClient := jobclient.NewMockClient(t)
	jobClient.On("GetManagedIdentity", ctx, "managedIdentity-1").Return(nil, status.Error(codes.Unavailable, "unavailable"))

	managedIdentities := &managedIdentities{
		managedIdentityIDs: []string{"managedIdentity-1"},
		jobLogger:          joblogger.NewMockLogger(t),
		client:             jobClient,
		authenticators:     []managedidentity.Authenticator{},
		factoryMap:         buildFactoryMap(managedidentity.NewMockAuthenticator(t)),
	}

	_, err := managedIdentities.initialize(ctx)
	assert.ErrorContains(t, err, "failed to get managed identity managedIdentity-1")
}
