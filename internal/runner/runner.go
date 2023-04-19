// Package runner package
package runner

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/docker"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/ecs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/local"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	checkRunsInterval       = 1 * time.Second
	checkRunsFailedInterval = 30 * time.Second
)

var (
	// runnerID will a constant UUID until runner registration is allowed.
	claimJobCount = metric.NewCounter("claim_job_count", "Amount of jobs claimed.")
	claimJobFails = metric.NewCounter("claim_job_fails_count", "Amount of jobs claims failed.")

	launchJobFails = metric.NewCounter("launch_job_fails", "Amount of launch jobs failed.")

	jobDispatchCount = metric.NewCounter("job_dispatch_count", "Amount of jobs dispatched.")
	jobDispatchTime  = metric.NewHistogram("job_dispatch_time", "Amount of time a job took to dispatch.", 1, 2, 8)
)

// ClaimJobInput is the input for claiming the next availble job
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
	ClaimJob(ctx context.Context, input *ClaimJobInput) (*ClaimJobResponse, error)
}

// JobDispatcherSettings defines the job dispatcher that'll be used for this runner
type JobDispatcherSettings struct {
	PluginData           map[string]string
	DispatcherType       string
	ServiceDiscoveryHost string
}

// Runner will claim the next available job and dispatch it using the configured job dispatcher
type Runner struct {
	jobDispatcher jobdispatcher.JobDispatcher
	logger        logger.Logger
	client        Client
	runnerPath    string
}

// NewRunner creates a new Runner
func NewRunner(
	ctx context.Context,
	runnerPath string,
	logger logger.Logger,
	client Client,
	jobDispatcherSettings *JobDispatcherSettings,
) (*Runner, error) {
	dispatcher, err := newJobDispatcherPlugin(ctx, logger, jobDispatcherSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create job dispatcher %v", err)
	}

	return &Runner{runnerPath: runnerPath, jobDispatcher: dispatcher, logger: logger, client: client}, nil
}

// Start will start the runner so it can begin picking up jobs
func (r *Runner) Start(ctx context.Context) {
	for {
		r.logger.Info("Waiting for next available run")

		resp, err := r.client.ClaimJob(ctx, &ClaimJobInput{
			RunnerPath: r.runnerPath,
		})
		claimJobCount.Inc()

		if err != nil {
			// Check if context has been canceled
			if ctx.Err() != nil {
				return
			}
			claimJobFails.Inc()
			r.logger.Errorf("Failed to request next available job %v", err)

			select {
			case <-ctx.Done():
			case <-time.After(checkRunsFailedInterval):
			}
		} else {
			r.logger.Infof("Claimed job with ID %s", resp.JobID)

			if err := r.launchJob(ctx, resp.JobID, resp.Token); err != nil {
				launchJobFails.Inc()
				r.logger.Errorf("Failed to launch job %v", err)
			}

			select {
			case <-ctx.Done():
			case <-time.After(checkRunsInterval):
			}
		}
	}
}

func (r *Runner) launchJob(ctx context.Context, jobID string, token string) error {
	// For measuring dispatch time in seconds.
	start := time.Now()
	executorID, err := r.jobDispatcher.DispatchJob(ctx, jobID, token)
	duration := time.Since(start)
	jobDispatchTime.Observe(float64(duration.Seconds()))
	jobDispatchCount.Inc()
	if err != nil {
		return err
	}

	r.logger.Infof("Job %s running in executor %s", jobID, executorID)

	return nil
}

func newJobDispatcherPlugin(ctx context.Context, logger logger.Logger, settings *JobDispatcherSettings) (jobdispatcher.JobDispatcher, error) {
	var (
		plugin jobdispatcher.JobDispatcher
		err    error
	)

	switch settings.DispatcherType {
	case "kubernetes":
		plugin, err = kubernetes.New(ctx, settings.PluginData, settings.ServiceDiscoveryHost, logger)
	case "ecs":
		plugin, err = ecs.New(ctx, settings.PluginData, settings.ServiceDiscoveryHost, logger)
	case "docker":
		plugin, err = docker.New(settings.PluginData, settings.ServiceDiscoveryHost, logger)
	case "local":
		plugin, err = local.New(settings.PluginData, settings.ServiceDiscoveryHost, logger)
	default:
		err = fmt.Errorf("the specified Job Executor plugin %s is not currently supported", settings.DispatcherType)
	}

	return plugin, err
}
