package jobexecutor

import (
	"context"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/awsfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/azurefederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/kubernetesfederated"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/managedidentity/tharsisfederated"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authenticatorFactoryFunc func() (managedidentity.Authenticator, error)

type managedIdentities struct {
	client     jobclient.Client
	jobLogger  joblogger.Logger
	factoryMap map[string]authenticatorFactoryFunc
	// managedIdentityIDs are the identities the job's run node was authorized to use. The API only
	// creates credentials for these, so identities assigned to the workspace later are not loaded.
	managedIdentityIDs []string
	authenticators     []managedidentity.Authenticator
}

type managedIdentityInitializeResponse struct {
	Env                       map[string]string
	HostCredentialFileMapping map[string]string
}

func newManagedIdentities(
	managedIdentityIDs []string,
	jobLogger joblogger.Logger,
	client jobclient.Client,
	jobCfg *JobConfig,
) *managedIdentities {
	return &managedIdentities{
		managedIdentityIDs: managedIdentityIDs,
		jobLogger:          jobLogger,
		client:             client,
		authenticators:     []managedidentity.Authenticator{},
		factoryMap: map[string]authenticatorFactoryFunc{
			pb.ManagedIdentityType_aws_federated.String(): func() (managedidentity.Authenticator, error) {
				return awsfederated.New()
			},
			pb.ManagedIdentityType_azure_federated.String(): func() (managedidentity.Authenticator, error) {
				return azurefederated.New()
			},
			pb.ManagedIdentityType_tharsis_federated.String(): func() (managedidentity.Authenticator, error) {
				return tharsisfederated.New(client, jobLogger, jobCfg.APIEndpoint, jobCfg.DiscoveryProtocolHosts)
			},
			pb.ManagedIdentityType_kubernetes_federated.String(): func() (managedidentity.Authenticator, error) {
				return kubernetesfederated.New()
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

	identitiesMap := map[string][]*pb.ManagedIdentity{}
	for _, id := range l.managedIdentityIDs {
		identity, err := l.client.GetManagedIdentity(ctx, id)
		if status.Code(err) == codes.NotFound || (err == nil && identity == nil) {
			// The identity was deleted after the run node was authorized to use it. Skip it so the
			// job still runs; anything that needed its credentials will fail on its own.
			l.jobLogger.Warningf("Managed identity %s was not found and will be skipped; it may have been deleted", id)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get managed identity %s: %v", id, err)
		}
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

		credsRetriever := func(ctx context.Context, managedIdentity *pb.ManagedIdentity) ([]byte, error) {
			l.jobLogger.Infof("Loading credentials for managed identity: %s", managedIdentity.Metadata.Trn)
			return l.client.CreateManagedIdentityCredentials(ctx, managedIdentity.Metadata.Id)
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
