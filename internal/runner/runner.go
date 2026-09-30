// Package runner package
package runner

import (
	"context"
	"fmt"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/docker"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/ecs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/kubernetes"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/local"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	checkRunsInterval       = 1 * time.Second
	checkRunsFailedInterval = 30 * time.Second
)

var (
	claimJobCount = metric.NewCounter("claim_job_count", "Amount of jobs claimed.")
	claimJobFails = metric.NewCounter("claim_job_fails_count", "Amount of jobs claims failed.")

	launchJobFails = metric.NewCounter("launch_job_fails", "Amount of launch jobs failed.")

	jobDispatchCount = metric.NewCounter("job_dispatch_count", "Amount of jobs dispatched.")
	jobDispatchTime  = metric.NewHistogram("job_dispatch_time", "Amount of time a job took to dispatch.", 1, 2, 8)
)

// JobDispatcherSettings defines the job dispatcher that'll be used for this runner
type JobDispatcherSettings struct {
	PluginData           map[string]string
	DispatcherType       string
	ServiceDiscoveryHost string
	TokenGetterFunc      types.TokenGetterFunc
}

// Runner will claim the next available job and dispatch it using the configured job dispatcher
type Runner struct {
	jobDispatcher jobdispatcher.JobDispatcher
	logger        logger.Logger
	client        Client
	runnerID      string
}

// NewRunner creates a new Runner
func NewRunner(
	ctx context.Context,
	runnerID string,
	logger logger.Logger,
	version string,
	client Client,
	jobDispatcherSettings *JobDispatcherSettings,
) (*Runner, error) {
	dispatcher, err := newJobDispatcherPlugin(ctx, logger, version, jobDispatcherSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create job dispatcher %v", err)
	}

	return &Runner{runnerID: runnerID, jobDispatcher: dispatcher, logger: logger, client: client}, nil
}

// Start will start the runner so it can begin picking up jobs
func (r *Runner) Start(ctx context.Context) {
	defer r.logger.Info("Runner session has ended")

	r.logger.Info("Creating new runner session")

	sessionID, err := r.client.CreateRunnerSession(ctx, &CreateRunnerSessionInput{
		RunnerID: r.runnerID,
	})
	if err != nil {
		r.logger.Errorf("Failed to create runner session %v", err)
		return
	}

	// Send keep alive
	go r.sendRunnerSessionHeartbeat(ctx, sessionID)

	// Start the cleanup worker to reap the runtimes of this runner's finished jobs.
	go newCleanupWorker(r.runnerID, sessionID, r.client, r.jobDispatcher, r.logger).start(ctx)

	for {
		r.logger.Info("Waiting for next available run")

		resp, err := r.client.ClaimJob(ctx, &ClaimJobInput{
			RunnerID: r.runnerID,
		})
		claimJobCount.Inc()

		if err != nil {
			// Check if context has been canceled
			if ctx.Err() != nil {
				return
			}
			claimJobFails.Inc()
			r.handleError(ctx, sessionID, fmt.Errorf("failed to request next available job %v", err))

			select {
			case <-ctx.Done():
			case <-time.After(checkRunsFailedInterval):
			}
		} else {
			r.logger.Infof("Claimed job with ID %s", resp.JobID)

			dispatcherData, err := r.launchJob(ctx, resp.JobID, resp.Token)
			if err != nil {
				launchJobFails.Inc()
				r.handleError(ctx, sessionID, fmt.Errorf("failed to launch job %v", err))
			} else {
				// Report the dispatch so limits are recorded and the cleanup loop can reap the runtime; a failure is tracked as a session error but doesn't fail the job.
				input := &JobDispatchedInput{JobID: resp.JobID, DispatcherData: dispatcherData, Limits: r.jobDispatcher.Limits()}
				if err := r.client.JobDispatched(ctx, input); err != nil {
					r.handleError(ctx, sessionID, fmt.Errorf("failed to report job dispatched for job %s: %v", resp.JobID, err))
				}
			}

			select {
			case <-ctx.Done():
			case <-time.After(checkRunsInterval):
			}
		}
	}
}

func (r *Runner) handleError(ctx context.Context, sessionID string, err error) {
	r.logger.Error(err)
	if sErr := r.client.CreateRunnerSessionError(ctx, sessionID, err); sErr != nil {
		r.logger.Errorf("failed to send error %v", sErr)
	}
}

func (r *Runner) launchJob(ctx context.Context, jobID string, token string) (map[string]string, error) {
	// For measuring dispatch time in seconds.
	start := time.Now()
	dispatcherData, err := r.jobDispatcher.DispatchJob(ctx, jobID, token)
	duration := time.Since(start)
	jobDispatchTime.Observe(float64(duration.Seconds()))
	jobDispatchCount.Inc()
	if err != nil {
		return nil, err
	}

	r.logger.Infof("Job %s dispatched with dispatcher data %v", jobID, dispatcherData)

	return dispatcherData, nil
}

func (r *Runner) sendRunnerSessionHeartbeat(ctx context.Context, sessionID string) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(models.RunnerSessionHeartbeatInterval):
			// Send heartbeat
			if err := r.client.SendRunnerSessionHeartbeat(ctx, sessionID); err != nil {
				r.logger.Errorf("failed to send runner session heartbeat: %v", err)
			}
		}
	}
}

func newJobDispatcherPlugin(ctx context.Context, logger logger.Logger, version string, settings *JobDispatcherSettings) (jobdispatcher.JobDispatcher, error) {
	var (
		plugin jobdispatcher.JobDispatcher
		err    error
	)

	switch settings.DispatcherType {
	case "kubernetes":
		plugin, err = kubernetes.New(ctx, settings.PluginData, settings.ServiceDiscoveryHost, settings.TokenGetterFunc, logger)
	case "ecs":
		plugin, err = ecs.New(ctx, settings.PluginData, settings.ServiceDiscoveryHost, logger)
	case "docker":
		plugin, err = docker.New(settings.PluginData, settings.ServiceDiscoveryHost, logger)
	case "local":
		plugin, err = local.New(settings.PluginData, settings.ServiceDiscoveryHost, logger, version)
	default:
		err = fmt.Errorf("the specified Job Executor plugin %s is not currently supported", settings.DispatcherType)
	}

	return plugin, err
}
