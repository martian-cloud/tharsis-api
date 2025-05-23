// Package jobclient package
package jobclient

//go:generate go tool mockery --name Client --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	tharsis "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/config"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

// Client interface is used by the Job Executor to interface with the Tharsis API
type Client interface {
	GetRun(ctx context.Context, id string) (*types.Run, error)
	GetJob(ctx context.Context, id string) (*types.Job, error)
	GetWorkspace(ctx context.Context, id string) (*types.Workspace, error)
	GetRunVariables(ctx context.Context, runID string) ([]types.RunVariable, error)
	GetAssignedManagedIdentities(ctx context.Context, workspaceID string) ([]types.ManagedIdentity, error)
	GetConfigurationVersion(ctx context.Context, id string) (*types.ConfigurationVersion, error)
	CreateStateVersion(ctx context.Context, runID string, body io.Reader) (*types.StateVersion, error)
	CreateManagedIdentityCredentials(ctx context.Context, managedIdentityID string) ([]byte, error)
	CreateTerraformCLIDownloadURL(ctx context.Context, input *types.CreateTerraformCLIDownloadURLInput) (string, error)
	SaveJobLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error
	SubscribeToJobCancellationEvent(ctx context.Context, jobID string) (<-chan *types.CancellationEvent, error)
	UpdateApply(ctx context.Context, apply *types.Apply) (*types.Apply, error)
	UpdatePlan(ctx context.Context, apply *types.Plan) (*types.Plan, error)
	UploadPlanCache(ctx context.Context, plan *types.Plan, body io.Reader) error
	UploadPlanData(ctx context.Context, plan *types.Plan, tfPlan *tfjson.Plan, tfProviderScheams *tfjson.ProviderSchemas) error
	DownloadConfigurationVersion(ctx context.Context, configurationVersion *types.ConfigurationVersion, writer io.WriterAt) error
	DownloadStateVersion(ctx context.Context, stateVersion *types.StateVersion, writer io.WriterAt) error
	DownloadPlanCache(ctx context.Context, planID string, writer io.WriterAt) error
	Close() error
	CreateServiceAccountToken(ctx context.Context, serviceAccountPath string, token string) (string, *time.Duration, error)
	SetVariablesIncludedInTFConfig(ctx context.Context, runID string, variableKeys []string) error
	CreateFederatedRegistryTokens(ctx context.Context, input *types.CreateFederatedRegistryTokensInput) ([]types.FederatedRegistryToken, error)
}

var _ Client = (*client)(nil)

type client struct {
	tharsisClient *tharsis.Client
}

// NewClient creates an instance of the Client interface
func NewClient(apiURL string, token string) (Client, error) {
	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	staticToken, err := auth.NewStaticTokenProvider(token)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(config.WithEndpoint(baseURL.String()), config.WithTokenProvider(staticToken))
	if err != nil {
		return nil, err
	}

	c, err := tharsis.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &client{tharsisClient: c}, nil
}

// Close will close any open connections
func (c *client) Close() error {
	return c.tharsisClient.Close()
}

// CreateManagedIdentityCredentials creates credentials for a managed identity and returns its contents
func (c *client) CreateManagedIdentityCredentials(ctx context.Context, managedIdentityID string) ([]byte, error) {
	credentialsOpts := types.CreateManagedIdentityCredentialsInput{ID: managedIdentityID}

	credentials, err := c.tharsisClient.ManagedIdentity.CreateManagedIdentityCredentials(ctx, &credentialsOpts)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}

// GetAssignedManagedIdentities returns a list of assigned managed identities for a workspace
func (c *client) GetAssignedManagedIdentities(ctx context.Context, workspaceID string) ([]types.ManagedIdentity, error) {
	identitiesOpts := &types.GetAssignedManagedIdentitiesInput{ID: &workspaceID}

	identities, err := c.tharsisClient.Workspaces.GetAssignedManagedIdentities(ctx, identitiesOpts)
	if err != nil {
		return nil, err
	}

	return identities, nil
}

// GetRun returns a run by ID
func (c *client) GetRun(ctx context.Context, id string) (*types.Run, error) {
	run, err := c.tharsisClient.Run.GetRun(ctx, &types.GetRunInput{ID: id})
	if err != nil {
		return nil, err
	}

	return run, nil
}

// GetRunVariables gets RunVariables for a run
func (c *client) GetRunVariables(ctx context.Context, runID string) ([]types.RunVariable, error) {
	// Get run variables and include sensitive values since they will be needed to run the job
	runVariables, err := c.tharsisClient.Run.GetRunVariables(ctx, &types.GetRunVariablesInput{
		RunID:                  runID,
		IncludeSensitiveValues: true,
	})
	if err != nil {
		return nil, err
	}

	return runVariables, nil
}

// GetJob returns a job by ID
func (c *client) GetJob(ctx context.Context, id string) (*types.Job, error) {
	job, err := c.tharsisClient.Job.GetJob(ctx, &types.GetJobInput{ID: id})
	if err != nil {
		return nil, err
	}

	return job, nil
}

// SubscribeToJobCancellationEvent returns job cancellation events for a job
func (c *client) SubscribeToJobCancellationEvent(ctx context.Context, jobID string) (<-chan *types.CancellationEvent, error) {
	eventChannel, err := c.tharsisClient.Job.SubscribeToJobCancellationEvent(ctx, &types.JobCancellationEventSubscriptionInput{JobID: jobID})
	if err != nil {
		return nil, err
	}

	return eventChannel, nil
}

// SaveJobLogs saves job logs and returns any errors
func (c *client) SaveJobLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error {
	return c.tharsisClient.Job.SaveJobLogs(ctx,
		&types.SaveJobLogsInput{
			Logs:        string(buffer),
			StartOffset: int32(startOffset),
			JobID:       jobID,
		},
	)
}

// GetWorkspace returns a workspace by ID
func (c *client) GetWorkspace(ctx context.Context, id string) (*types.Workspace, error) {
	workspace, err := c.tharsisClient.Workspaces.GetWorkspace(ctx, &types.GetWorkspaceInput{ID: &id})
	if err != nil {
		return nil, err
	}

	return workspace, nil
}

// UpdateApply updates an apply and returns its contents
func (c *client) UpdateApply(ctx context.Context, apply *types.Apply) (*types.Apply, error) {
	updatedApply, err := c.tharsisClient.Apply.UpdateApply(ctx,
		&types.UpdateApplyInput{
			ID:           apply.Metadata.ID,
			Status:       apply.Status,
			ErrorMessage: apply.ErrorMessage,
		},
	)
	if err != nil {
		return nil, err
	}

	return updatedApply, nil
}

// UpdatePlan updates a plan and returns its contents
func (c *client) UpdatePlan(ctx context.Context, plan *types.Plan) (*types.Plan, error) {
	updatedPlan, err := c.tharsisClient.Plan.UpdatePlan(ctx,
		&types.UpdatePlanInput{
			ID:           plan.Metadata.ID,
			Status:       plan.Status,
			HasChanges:   plan.HasChanges,
			ErrorMessage: plan.ErrorMessage,
		},
	)
	if err != nil {
		return nil, err
	}

	return updatedPlan, nil
}

// CreateStateVersion creates a new state version and returns its contents
func (c *client) CreateStateVersion(ctx context.Context, runID string, body io.Reader) (*types.StateVersion, error) {

	// Base64 encode state data
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(body); err != nil {
		return nil, err
	}
	state := base64.StdEncoding.EncodeToString(buf.Bytes())

	stateVersionOpts := types.CreateStateVersionInput{
		State: state,
		RunID: runID,
	}

	stateVersion, err := c.tharsisClient.StateVersion.CreateStateVersion(ctx, &stateVersionOpts)
	if err != nil {
		return nil, err
	}

	return stateVersion, nil
}

// GetConfigurationVersion returns a configuration version by ID
func (c *client) GetConfigurationVersion(ctx context.Context, id string) (*types.ConfigurationVersion, error) {
	configurationVersion, err := c.tharsisClient.ConfigurationVersion.GetConfigurationVersion(ctx,
		&types.GetConfigurationVersionInput{
			ID: id,
		},
	)
	if err != nil {
		return nil, err
	}

	return configurationVersion, nil
}

// DownloadConfigurationVersion downloads a configuration version and returns any errors
func (c *client) DownloadConfigurationVersion(ctx context.Context, configurationVersion *types.ConfigurationVersion, writer io.WriterAt) error {
	err := c.tharsisClient.ConfigurationVersion.DownloadConfigurationVersion(ctx,
		&types.GetConfigurationVersionInput{
			ID: configurationVersion.Metadata.ID,
		},
		writer,
	)
	if err != nil {
		return err
	}

	return nil
}

// DownloadStateVersion downloads a state version and returns any errors
func (c *client) DownloadStateVersion(ctx context.Context, stateVersion *types.StateVersion, writer io.WriterAt) error {
	err := c.tharsisClient.StateVersion.DownloadStateVersion(ctx,
		&types.DownloadStateVersionInput{
			ID: stateVersion.Metadata.ID,
		},
		writer,
	)
	if err != nil {
		return err
	}

	return nil
}

// DownloadPlanCache downloads a plan cache and returns any errors
func (c *client) DownloadPlanCache(ctx context.Context, planID string, writer io.WriterAt) error {
	err := c.tharsisClient.Plan.DownloadPlanCache(ctx, planID, writer)
	if err != nil {
		return err
	}

	return nil
}

// UploadPlanCache uploads a plan cache and returns any errors
func (c *client) UploadPlanCache(ctx context.Context, plan *types.Plan, body io.Reader) error {
	return c.tharsisClient.Plan.UploadPlanCache(ctx, plan.Metadata.ID, body)
}

// UploadPlanData uploads the json plan and provider schemas and returns any errors
func (c *client) UploadPlanData(ctx context.Context, plan *types.Plan, tfPlan *tfjson.Plan, tfProviderScheams *tfjson.ProviderSchemas) error {
	return c.tharsisClient.Plan.UploadPlanData(ctx, plan.Metadata.ID, tfPlan, tfProviderScheams)
}

// CreateTerraformCLIDownloadURL creates a download URL which can be used to
// download a Terraform CLI binary directly.
func (c *client) CreateTerraformCLIDownloadURL(ctx context.Context,
	input *types.CreateTerraformCLIDownloadURLInput) (string, error) {
	downloadURL, err := c.tharsisClient.TerraformCLIVersions.CreateTerraformCLIDownloadURL(ctx, input)
	if err != nil {
		return "", err
	}

	return downloadURL, nil
}

// CreateServiceAccountToken Creates a service account token from the given token
func (c *client) CreateServiceAccountToken(ctx context.Context, serviceAccountPath string, managedIdentityToken string) (string, *time.Duration, error) {
	input := &types.ServiceAccountCreateTokenInput{
		ServiceAccountPath: serviceAccountPath,
		Token:              managedIdentityToken,
	}

	response, err := c.tharsisClient.ServiceAccount.CreateToken(ctx, input)
	if err != nil {
		return "", nil, err
	}

	return response.Token, &response.ExpiresIn, nil
}

// SetVariablesIncludedInTFConfig updates run variables usage.
func (c *client) SetVariablesIncludedInTFConfig(ctx context.Context, runID string, variableKeys []string) error {
	return c.tharsisClient.Run.SetVariablesIncludedInTFConfig(ctx, &types.SetVariablesIncludedInTFConfigInput{
		RunID:        runID,
		VariableKeys: variableKeys,
	})
}

// CreateFederatedRegistryTokens creates one or more federated registry tokens pursuant to the federated registry feature.
func (c *client) CreateFederatedRegistryTokens(ctx context.Context,
	input *types.CreateFederatedRegistryTokensInput,
) ([]types.FederatedRegistryToken, error) {
	tokens, err := c.tharsisClient.FederatedRegistry.CreateFederatedRegistryTokens(ctx, input)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
