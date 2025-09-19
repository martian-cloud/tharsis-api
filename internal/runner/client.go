package runner

import (
	"context"
	"sync"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/job"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/runner"
)

const (
	internalRunnerJWTType       = "internal_runner"
	internalRunnerJWTAudience   = "job-dispatcher"
	internalRunnerJWTExpiration = 5 * time.Minute
	expirationLeeway            = 30 * time.Second
)

// CreateRunnerSessionInput is the input for creating a runner session
type CreateRunnerSessionInput struct {
	RunnerPath string
}

// ClaimJobInput is the input for claiming the next available job
type ClaimJobInput struct {
	RunnerPath string
}

// ClaimJobResponse is the response when claiming a job
type ClaimJobResponse struct {
	JobID string
	Token string
}

// Client interface for claiming a job
type Client interface {
	CreateRunnerSession(ctx context.Context, input *CreateRunnerSessionInput) (string, error)
	SendRunnerSessionHeartbeat(ctx context.Context, sessionID string) error
	ClaimJob(ctx context.Context, input *ClaimJobInput) (*ClaimJobResponse, error)
	CreateRunnerSessionError(ctx context.Context, sessionID string, err error) error
}

// InternalTokenProvider is a token provider for internal runners
type InternalTokenProvider struct {
	runnerName string
	runnerGID  string
	signingKeyManager auth.SigningKeyManager
	expires    *time.Time
	token      string
	mutex      sync.RWMutex
}

// GetToken retrieves an ID token for the internal runner, generating a new one if the current one is expired.
func (i *InternalTokenProvider) GetToken(ctx context.Context) (string, error) {
	if !i.isTokenExpired() {
		i.mutex.RLock()
		defer i.mutex.RUnlock()
		return i.token, nil
	}

	expiration := time.Now().Add(internalRunnerJWTExpiration)
	token, err := i.signingKeyManager.GenerateToken(ctx, &auth.TokenInput{
		Expiration: &expiration,
		Subject:    i.runnerName,
		Audience:   internalRunnerJWTAudience,
		Claims: map[string]string{
			"runner_id": i.runnerGID,
			"type":      internalRunnerJWTType,
		},
	})
	if err != nil {
		return "", err
	}

	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.expires = &expiration
	i.token = string(token)

	return i.token, nil
}

// isTokenExpired checks if the current token is expired or not.
func (i *InternalTokenProvider) isTokenExpired() bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return i.expires == nil || !time.Now().Add(expirationLeeway).Before(*i.expires)
}

// NewInternalTokenProvider creates a new InternalTokenProvider for internal runners.
func NewInternalTokenProvider(runnerName string, runnerID string, signingKeyManager auth.SigningKeyManager) *InternalTokenProvider {
	return &InternalTokenProvider{
		runnerName:        runnerName,
		runnerGID:         gid.ToGlobalID(types.RunnerModelType, runnerID),
		signingKeyManager: signingKeyManager,
	}
}

// internalClient is the client for the internal system runner.
type internalClient struct {
	jobService    job.Service
	runnerService runner.Service
}

// NewInternalClient creates a new internal client
func NewInternalClient(runnerService runner.Service, jobService job.Service) Client {
	return &internalClient{
		jobService:    jobService,
		runnerService: runnerService,
	}
}

func (a *internalClient) CreateRunnerSessionError(ctx context.Context, sessionID string, err error) error {
	return a.runnerService.CreateRunnerSessionError(ctx, gid.FromGlobalID(sessionID), err.Error())
}

func (a *internalClient) CreateRunnerSession(ctx context.Context, input *CreateRunnerSessionInput) (string, error) {
	session, err := a.runnerService.CreateRunnerSession(ctx, &runner.CreateRunnerSessionInput{
		RunnerPath: input.RunnerPath,
		Internal:   true,
	})
	if err != nil {
		return "", err
	}
	return session.GetGlobalID(), nil
}

func (a *internalClient) SendRunnerSessionHeartbeat(ctx context.Context, sessionID string) error {
	return a.runnerService.AcceptRunnerSessionHeartbeat(ctx, gid.FromGlobalID(sessionID))
}

func (a *internalClient) ClaimJob(ctx context.Context, input *ClaimJobInput) (*ClaimJobResponse, error) {
	runner, err := a.runnerService.GetRunnerByTRN(ctx, types.RunnerModelType.BuildTRN(input.RunnerPath))
	if err != nil {
		return nil, err
	}

	resp, err := a.jobService.ClaimJob(ctx, runner.Metadata.ID)
	if err != nil {
		return nil, err
	}

	return &ClaimJobResponse{
		JobID: gid.ToGlobalID(types.JobModelType, resp.JobID),
		Token: resp.Token,
	}, nil
}
