// Package jobclient package
package jobclient

//go:generate go tool mockery --name Client --inpackage --case underscore

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client/token"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ Client = (*jobClient)(nil)

// UpdateApplyInput is the input for updating an apply.
type UpdateApplyInput struct {
	ID           string
	ErrorMessage *string
}

// UpdatePlanInput is the input for updating a plan.
type UpdatePlanInput struct {
	ID           string
	HasChanges   bool
	ErrorMessage *string
}

// RunPolicyOutcomeInput is a single policy's outcome reported for a run stage, keyed by the
// PolicyCheckPolicy id it was evaluated from. Messages holds one entry per violation, in display
// order, and is empty when the policy passed.
type RunPolicyOutcomeInput struct {
	PolicyID string
	Messages []string
	Passed   bool
}

// Client interface is used by the Job Executor to interface with the Tharsis API
type Client interface {
	GetRun(ctx context.Context, id string) (*pb.Run, error)
	GetJob(ctx context.Context, id string) (*pb.Job, error)
	GetWorkspace(ctx context.Context, id string) (*pb.Workspace, error)
	GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]*pb.RunVariable, error)
	GetPackageVersion(ctx context.Context, source, versionConstraint string) (*pb.PackageVersion, error)
	DownloadPackage(ctx context.Context, packageVersionID string) (io.ReadCloser, error)
	GetAssignedManagedIdentities(ctx context.Context, workspaceID string) ([]*pb.ManagedIdentity, error)
	GetConfigurationVersion(ctx context.Context, id string) (*pb.ConfigurationVersion, error)
	CreateStateVersion(ctx context.Context, runID string, body io.Reader) (*pb.StateVersion, error)
	GetRunStateVersion(ctx context.Context, runID string) (*pb.StateVersion, error)
	CreateManagedIdentityCredentials(ctx context.Context, managedIdentityID string) ([]byte, error)
	CreateTerraformCLIDownloadURL(ctx context.Context, version, os, architecture string) (string, error)
	SaveJobLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error
	SubscribeToJobCancellationEvent(ctx context.Context, jobID string) (pb.Jobs_SubscribeToJobCancellationEventClient, error)
	UpdateApply(ctx context.Context, input *UpdateApplyInput) (*pb.Apply, error)
	UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*pb.Plan, error)
	ReportRunPolicyOutcomes(ctx context.Context, policyCheckID string, outcomes []RunPolicyOutcomeInput) error
	SetJobStatus(ctx context.Context, jobID string, status pb.JobStatus, jobProtocolVersion string) (*pb.Job, error)
	UploadPlanCache(ctx context.Context, planID string, body io.Reader) error
	UploadPlanData(ctx context.Context, planID string, tfPlan *tfjson.Plan, tfProviderSchemas *tfjson.ProviderSchemas) error
	UploadStateVersionJSON(ctx context.Context, stateVersionID string, tfState *tfjson.State) error
	DownloadConfigurationVersion(ctx context.Context, configVersionID string, writer io.Writer) error
	DownloadStateVersion(ctx context.Context, stateVersionID string, writer io.Writer) error
	DownloadStateVersionJSON(ctx context.Context, stateVersionID string, writer io.Writer) error
	DownloadPlanCache(ctx context.Context, planID string, writer io.Writer) error
	DownloadPlanJSON(ctx context.Context, planID string, writer io.Writer) error
	Close() error
	CreateServiceAccountToken(ctx context.Context, serviceAccountPath string, token string) (string, *time.Duration, error)
	SetVariablesIncludedInTFConfig(ctx context.Context, runID string, variableKeys []string) error
	CreateFederatedRegistryTokens(ctx context.Context, jobID string) ([]*pb.FederatedRegistryToken, error)
	CreateProviderVersionMirror(ctx context.Context, input *pb.CreateTerraformProviderVersionMirrorRequest) (*pb.TerraformProviderVersionMirror, error)
	GetProviderPlatformMirror(ctx context.Context, id string) (*pb.TerraformProviderPlatformMirror, error)
	UploadProviderPlatformPackageToMirror(ctx context.Context, versionMirrorID, os, arch string, reader io.Reader) error
	GetProviderPlatformPackageDownloadURL(ctx context.Context, input *pb.GetProviderPlatformPackageDownloadURLRequest) (*pb.GetProviderPlatformPackageDownloadURLResponse, error)
	GetAvailableProviderVersions(ctx context.Context, input *pb.GetAvailableProviderVersionsRequest) (map[string]struct{}, error)
}

type jobClient struct {
	grpcClient *client.GRPCClient
	restClient client.RESTClient
}

// ClientConfig holds configuration for creating a new job client.
type ClientConfig struct {
	Logger        client.LeveledLogger
	APIEndpoint   string
	Token         string
	UserAgent     string
	TLSSkipVerify bool
}

// NewClient creates an instance of the Client interface
func NewClient(ctx context.Context, cfg *ClientConfig) (Client, error) {
	// Parse the address to make sure its a valid URL.
	baseURL, err := url.Parse(cfg.APIEndpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid address: %v", err)
	}

	tokenResolver, err := token.NewStatic(func() (string, error) {
		return cfg.Token, nil
	})
	if err != nil {
		return nil, err
	}

	grpcClient, err := client.NewGRPCClient(ctx, &client.GRPCClientConfig{
		HTTPEndpoint:  baseURL.String(),
		TokenResolver: tokenResolver,
		TLSSkipVerify: cfg.TLSSkipVerify,
		UserAgent:     cfg.UserAgent,
		Logger:        cfg.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	restClient, err := client.NewRESTClient(&client.RESTClientConfig{
		Endpoint:           baseURL.String(),
		TokenResolver:      tokenResolver,
		UserAgent:          &cfg.UserAgent,
		InsecureSkipVerify: cfg.TLSSkipVerify,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	return &jobClient{grpcClient: grpcClient, restClient: restClient}, nil
}

// Close will close any open connections
func (c *jobClient) Close() error {
	return c.grpcClient.Close()
}

// CreateManagedIdentityCredentials creates credentials for a managed identity and returns its contents
func (c *jobClient) CreateManagedIdentityCredentials(ctx context.Context, managedIdentityID string) ([]byte, error) {
	resp, err := c.grpcClient.ManagedIdentitiesClient.CreateManagedIdentityCredentials(ctx, &pb.CreateManagedIdentityCredentialsRequest{
		ManagedIdentityId: managedIdentityID,
	})
	if err != nil {
		return nil, err
	}

	return []byte(resp.Data), nil
}

// GetAssignedManagedIdentities returns a list of assigned managed identities for a workspace
func (c *jobClient) GetAssignedManagedIdentities(ctx context.Context, workspaceID string) ([]*pb.ManagedIdentity, error) {
	resp, err := c.grpcClient.ManagedIdentitiesClient.GetManagedIdentitiesForWorkspace(ctx, &pb.GetManagedIdentitiesForWorkspaceRequest{
		WorkspaceId: workspaceID,
	})
	if err != nil {
		return nil, err
	}

	return resp.ManagedIdentities, nil
}

// GetRun returns a run by ID
func (c *jobClient) GetRun(ctx context.Context, id string) (*pb.Run, error) {
	return c.grpcClient.RunsClient.GetRunByID(ctx, &pb.GetRunByIDRequest{Id: id})
}

// GetRunVariables gets RunVariables for a run. includeSensitiveValues controls whether sensitive
// variable values are resolved and returned; callers that don't use the values themselves (e.g.
// policy evaluation) should pass false.
func (c *jobClient) GetRunVariables(ctx context.Context, runID string, includeSensitiveValues bool) ([]*pb.RunVariable, error) {
	resp, err := c.grpcClient.RunsClient.GetRunVariables(ctx, &pb.GetRunVariablesRequest{
		Id:                     runID,
		IncludeSensitiveValues: includeSensitiveValues,
	})
	if err != nil {
		return nil, err
	}

	return resp.Variables, nil
}

// GetPackageVersion resolves a package source and version constraint to the uploaded package version
// satisfying it, returning that version (including its global id) so the caller can download it. An
// empty constraint resolves to the latest uploaded version. A missing package, or a constraint no
// uploaded version satisfies, surfaces as a NotFound gRPC error (status.Code(err) == codes.NotFound).
func (c *jobClient) GetPackageVersion(ctx context.Context, source, versionConstraint string) (*pb.PackageVersion, error) {
	return c.grpcClient.PackagesClient.GetPackageVersion(ctx, &pb.GetPackageVersionRequest{
		Source:            source,
		VersionConstraint: versionConstraint,
	})
}

// DownloadPackage returns a reader over the tar.gz package for the given package version id. A
// deleted or unavailable package version surfaces as a NotFound gRPC error, which the caller can
// detect via status.Code(err) == codes.NotFound. The caller must Close the returned reader.
func (c *jobClient) DownloadPackage(ctx context.Context, packageVersionID string) (io.ReadCloser, error) {
	var buf bytes.Buffer
	err := c.restClient.DownloadPackage(ctx, &client.DownloadPackageInput{
		PackageVersionID: packageVersionID,
		Writer:           &buf,
	})
	if err != nil {
		if errors.Is(err, client.ErrPackageNotFound) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, fmt.Errorf("failed to download package: %w", err)
	}

	return io.NopCloser(&buf), nil
}

// GetJob returns a job by ID
func (c *jobClient) GetJob(ctx context.Context, id string) (*pb.Job, error) {
	return c.grpcClient.JobsClient.GetJobByID(ctx, &pb.GetJobByIDRequest{Id: id})
}

// SubscribeToJobCancellationEvent returns job cancellation events for a job
func (c *jobClient) SubscribeToJobCancellationEvent(ctx context.Context, jobID string) (pb.Jobs_SubscribeToJobCancellationEventClient, error) {
	stream, err := c.grpcClient.JobsClient.SubscribeToJobCancellationEvent(ctx, &pb.SubscribeToJobCancellationEventRequest{
		JobId: jobID,
	})
	if err != nil {
		return nil, err
	}

	return stream, nil
}

// SaveJobLogs saves job logs and returns any errors
func (c *jobClient) SaveJobLogs(ctx context.Context, jobID string, startOffset int, buffer []byte) error {
	_, err := c.grpcClient.JobsClient.SaveJobLogs(ctx, &pb.SaveJobLogsRequest{
		JobId:       jobID,
		StartOffset: int32(startOffset),
		Logs:        string(buffer),
	})

	return err
}

// GetWorkspace returns a workspace by ID
func (c *jobClient) GetWorkspace(ctx context.Context, id string) (*pb.Workspace, error) {
	return c.grpcClient.WorkspacesClient.GetWorkspaceByID(ctx, &pb.GetWorkspaceByIDRequest{Id: id})
}

// UpdateApply updates an apply and returns its contents
func (c *jobClient) UpdateApply(ctx context.Context, input *UpdateApplyInput) (*pb.Apply, error) {
	return c.grpcClient.RunsClient.UpdateApply(ctx, &pb.UpdateApplyRequest{
		Id:           input.ID,
		ErrorMessage: input.ErrorMessage,
	})
}

// UpdatePlan updates a plan and returns its contents
func (c *jobClient) UpdatePlan(ctx context.Context, input *UpdatePlanInput) (*pb.Plan, error) {
	return c.grpcClient.RunsClient.UpdatePlan(ctx, &pb.UpdatePlanRequest{
		Id:           input.ID,
		HasChanges:   input.HasChanges,
		ErrorMessage: input.ErrorMessage,
	})
}

// ReportRunPolicyOutcomes reports the policy-set outcomes for a run's policy check node.
func (c *jobClient) ReportRunPolicyOutcomes(ctx context.Context, policyCheckID string, outcomes []RunPolicyOutcomeInput) error {
	pbOutcomes := make([]*pb.RunPolicyOutcomeInput, len(outcomes))
	for i, o := range outcomes {
		pbOutcomes[i] = &pb.RunPolicyOutcomeInput{
			PolicyId: o.PolicyID,
			Messages: o.Messages,
			Passed:   o.Passed,
		}
	}
	_, err := c.grpcClient.RunsClient.ReportRunPolicyOutcomes(ctx, &pb.ReportRunPolicyOutcomesRequest{
		PolicyCheckId: policyCheckID,
		Outcomes:      pbOutcomes,
	})
	return err
}

// SetJobStatus sets the status of a job via gRPC.
func (c *jobClient) SetJobStatus(ctx context.Context, jobID string, status pb.JobStatus, jobProtocolVersion string) (*pb.Job, error) {
	return c.grpcClient.JobsClient.SetJobStatus(ctx, &pb.SetJobStatusInput{
		JobId:              jobID,
		Status:             status,
		JobProtocolVersion: jobProtocolVersion,
	})
}

// CreateStateVersion creates a new state version and returns its contents
func (c *jobClient) CreateStateVersion(ctx context.Context, runID string, body io.Reader) (*pb.StateVersion, error) {
	// Base64 encode state data
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(body); err != nil {
		return nil, err
	}

	state := base64.StdEncoding.EncodeToString(buf.Bytes())

	return c.grpcClient.StateVersionsClient.CreateStateVersion(ctx, &pb.CreateStateVersionRequest{
		RunId: &runID,
		State: state,
	})
}

// GetRunStateVersion returns the state version a run's apply created. A run that wrote no state has
// none, which the server reports as a NotFound status; callers that only need the state version when
// it exists should check for codes.NotFound rather than treating it as a failure.
func (c *jobClient) GetRunStateVersion(ctx context.Context, runID string) (*pb.StateVersion, error) {
	return c.grpcClient.StateVersionsClient.GetRunStateVersion(ctx, &pb.GetRunStateVersionRequest{
		RunId: runID,
	})
}

// GetConfigurationVersion returns a configuration version by ID
func (c *jobClient) GetConfigurationVersion(ctx context.Context, id string) (*pb.ConfigurationVersion, error) {
	return c.grpcClient.ConfigurationVersionsClient.GetConfigurationVersionByID(ctx, &pb.GetConfigurationVersionByIDRequest{Id: id})
}

// DownloadConfigurationVersion downloads a configuration version and returns any errors
func (c *jobClient) DownloadConfigurationVersion(ctx context.Context, configVersionID string, writer io.Writer) error {
	return c.restClient.DownloadConfigurationVersion(ctx, &client.DownloadConfigurationVersionInput{
		ConfigVersionID: configVersionID,
		Writer:          writer,
	})
}

// DownloadStateVersion downloads a state version and returns any errors
func (c *jobClient) DownloadStateVersion(ctx context.Context, stateVersionID string, writer io.Writer) error {
	return c.restClient.DownloadStateVersion(ctx, &client.DownloadStateVersionInput{
		StateVersionID: stateVersionID,
		Writer:         writer,
	})
}

// UploadStateVersionJSON uploads the "terraform show -json" rendering of a state version that has
// already been created. Sent separately from CreateStateVersion because that call carries the raw
// state base64-encoded inside a single gRPC message, which the rendering would not reliably fit
// alongside; this goes over the streaming REST endpoint instead.
func (c *jobClient) UploadStateVersionJSON(ctx context.Context, stateVersionID string, tfState *tfjson.State) error {
	data, err := json.Marshal(tfState)
	if err != nil {
		return fmt.Errorf("failed to marshal state json: %w", err)
	}

	return c.restClient.UploadStateVersionJSON(ctx, &client.UploadStateVersionJSONInput{
		StateVersionID: stateVersionID,
		Reader:         bytes.NewReader(data),
	})
}

// DownloadStateVersionJSON downloads a state version's JSON rendering. A state version with no
// rendering yields client.ErrStateVersionJSONNotFound.
func (c *jobClient) DownloadStateVersionJSON(ctx context.Context, stateVersionID string, writer io.Writer) error {
	return c.restClient.DownloadStateVersionJSON(ctx, &client.DownloadStateVersionJSONInput{
		StateVersionID: stateVersionID,
		Writer:         writer,
	})
}

// DownloadPlanCache downloads a plan cache and returns any errors
func (c *jobClient) DownloadPlanCache(ctx context.Context, planID string, writer io.Writer) error {
	return c.restClient.DownloadPlanCache(ctx, &client.DownloadPlanCacheInput{
		PlanID: planID,
		Writer: writer,
	})
}

// DownloadPlanJSON downloads a plan's JSON representation and returns any errors
func (c *jobClient) DownloadPlanJSON(ctx context.Context, planID string, writer io.Writer) error {
	return c.restClient.DownloadPlanJSON(ctx, &client.DownloadPlanJSONInput{
		PlanID: planID,
		Writer: writer,
	})
}

// UploadPlanCache uploads a plan cache and returns any errors
func (c *jobClient) UploadPlanCache(ctx context.Context, planID string, body io.Reader) error {
	return c.restClient.UploadPlanCache(ctx, &client.UploadPlanCacheInput{
		PlanID: planID,
		Reader: body,
	})
}

// UploadPlanData uploads the json plan and provider schemas and returns any errors
func (c *jobClient) UploadPlanData(ctx context.Context, planID string, tfPlan *tfjson.Plan, tfProviderSchemas *tfjson.ProviderSchemas) error {
	data, err := json.Marshal(struct {
		Plan            *tfjson.Plan            `json:"plan"`
		ProviderSchemas *tfjson.ProviderSchemas `json:"provider_schemas"`
	}{
		Plan:            tfPlan,
		ProviderSchemas: tfProviderSchemas,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal plan data: %w", err)
	}

	return c.restClient.UploadPlanData(ctx, &client.UploadPlanDataInput{
		PlanID: planID,
		Reader: bytes.NewReader(data),
	})
}

// CreateTerraformCLIDownloadURL creates a download URL which can be used to
// download a Terraform CLI binary directly.
func (c *jobClient) CreateTerraformCLIDownloadURL(ctx context.Context, version, os, architecture string) (string, error) {
	resp, err := c.grpcClient.TerraformCLIVersionsClient.CreateTerraformCLIDownloadURL(ctx, &pb.CreateTerraformCLIDownloadURLRequest{
		Version:      version,
		Os:           os,
		Architecture: architecture,
	})
	if err != nil {
		return "", err
	}

	return resp.Url, nil
}

// CreateServiceAccountToken creates a service account token from the given token
func (c *jobClient) CreateServiceAccountToken(ctx context.Context, serviceAccountPath string, managedIdentityToken string) (string, *time.Duration, error) {
	resp, err := c.grpcClient.ServiceAccountsClient.CreateOIDCToken(ctx, &pb.CreateOIDCTokenRequest{
		ServiceAccountId: serviceAccountPath,
		Token:            managedIdentityToken,
	})
	if err != nil {
		return "", nil, err
	}

	expiresIn := time.Duration(resp.ExpiresIn) * time.Second

	return resp.Token, &expiresIn, nil
}

// SetVariablesIncludedInTFConfig updates run variables usage.
func (c *jobClient) SetVariablesIncludedInTFConfig(ctx context.Context, runID string, variableKeys []string) error {
	_, err := c.grpcClient.RunsClient.SetVariablesIncludedInTFConfig(ctx, &pb.SetVariablesIncludedInTFConfigRequest{
		RunId:        runID,
		VariableKeys: variableKeys,
	})

	return err
}

// CreateFederatedRegistryTokens creates one or more federated registry tokens pursuant to the federated registry feature.
func (c *jobClient) CreateFederatedRegistryTokens(ctx context.Context, jobID string) ([]*pb.FederatedRegistryToken, error) {
	resp, err := c.grpcClient.FederatedRegistriesClient.CreateFederatedRegistryTokens(ctx, &pb.CreateFederatedRegistryTokensRequest{
		JobId: jobID,
	})
	if err != nil {
		return nil, err
	}

	return resp.Tokens, nil
}

// CreateProviderVersionMirror creates a new provider version mirror.
func (c *jobClient) CreateProviderVersionMirror(ctx context.Context, input *pb.CreateTerraformProviderVersionMirrorRequest) (*pb.TerraformProviderVersionMirror, error) {
	return c.grpcClient.TerraformProviderMirrorsClient.CreateTerraformProviderVersionMirror(ctx, input)
}

// GetProviderPlatformMirror returns a platform mirror by ID. Returns nil, nil if not found.
func (c *jobClient) GetProviderPlatformMirror(ctx context.Context, id string) (*pb.TerraformProviderPlatformMirror, error) {
	resp, err := c.grpcClient.TerraformProviderMirrorsClient.GetTerraformProviderPlatformMirrorByID(ctx, &pb.GetTerraformProviderPlatformMirrorByIDRequest{Id: id})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}

		return nil, err
	}

	return resp, nil
}

// UploadProviderPlatformPackageToMirror uploads a provider platform package to the mirror.
func (c *jobClient) UploadProviderPlatformPackageToMirror(ctx context.Context, versionMirrorID, os, arch string, reader io.Reader) error {
	return c.restClient.UploadProviderPlatformPackageToMirror(ctx, &client.UploadProviderPlatformPackageToMirrorInput{
		VersionMirrorID: versionMirrorID,
		OS:              os,
		Arch:            arch,
		Reader:          reader,
	})
}

// GetProviderPlatformPackageDownloadURL returns the download URL and hashes for a provider platform package.
func (c *jobClient) GetProviderPlatformPackageDownloadURL(ctx context.Context, input *pb.GetProviderPlatformPackageDownloadURLRequest) (*pb.GetProviderPlatformPackageDownloadURLResponse, error) {
	return c.grpcClient.TerraformProviderMirrorsClient.GetProviderPlatformPackageDownloadURL(ctx, input)
}

// GetAvailableProviderVersions returns all cached versions for a provider.
func (c *jobClient) GetAvailableProviderVersions(ctx context.Context, input *pb.GetAvailableProviderVersionsRequest) (map[string]struct{}, error) {
	resp, err := c.grpcClient.TerraformProviderMirrorsClient.GetAvailableProviderVersions(ctx, input)
	if err != nil {
		return nil, err
	}

	versions := make(map[string]struct{}, len(resp.Versions))
	for _, v := range resp.Versions {
		versions[v] = struct{}{}
	}

	return versions, nil
}
