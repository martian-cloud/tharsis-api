package jobexecutor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-sdk-go/pkg/types"
)

const (
	// waitCancelError is the duration to sleep if cancel wait gets error
	waitCancelError = 10 * time.Second

	// waitForcedCancel is the duration to sleep between polls looking for forced cancel.
	waitForcedCancel = 30 * time.Second
)

// JobHandler contains the job lifecycle functions
type JobHandler interface {
	Execute(ctx context.Context) error
	OnError(ctx context.Context, err error) error
	OnSuccess(ctx context.Context) error
}

// JobConfig is used to configure the job
type JobConfig struct {
	JobID                 string
	APIEndpoint           string
	JobToken              string
	DiscoveryProtocolHost string
}

// JobExecutor executes a job
type JobExecutor struct {
	cfg    *JobConfig
	client Client
	logger logger.Logger
}

// NewJobExecutor creates a new JobExecutor
func NewJobExecutor(
	cfg *JobConfig,
	client Client,
	logger logger.Logger,
) *JobExecutor {
	return &JobExecutor{cfg, client, logger}
}

// Execute executes the job associated with the JobExecutor instance
func (j *JobExecutor) Execute(ctx context.Context) error {
	jobLogger, err := newJobLogger(j.cfg.JobID, j.client, j.logger)
	if err != nil {
		return fmt.Errorf("failed to create job logger %v", err)
	}

	defer jobLogger.Close()

	jobLogger.start()

	// Get the memory limit if one has been passed in.
	memoryLimit := uint64(0)
	sLimit := os.Getenv("MEMORY_LIMIT")
	if sLimit != "" {
		var pErr error
		memoryLimit, pErr = humanize.ParseBytes(sLimit)
		if pErr != nil {
			return fmt.Errorf("invalid memory limit: MEMORY_LIMIT was %s: %w", sLimit, pErr)
		}
	}

	// If there is a defined memory limit, create a memory monitor and launch it.
	var memoryMonitor MemoryMonitor
	if memoryLimit > 0 {
		memoryMonitor, err = NewMemoryMonitor(jobLogger, memoryLimit)
		if err != nil {
			return err
		}
		memoryMonitor.Start(ctx)
		defer memoryMonitor.Stop()
	}

	workspaceDir, err := os.MkdirTemp("", "tfworkspace")
	if err != nil {
		return fmt.Errorf("failed to create temp workspace dir %v", err)
	}
	defer os.RemoveAll(workspaceDir)

	jobLogger.Infof("Starting job %s \n", j.cfg.JobID)

	// Build job
	handler, err := j.buildJobHandler(ctx, workspaceDir, jobLogger)
	if err != nil {
		return err
	}

	// Execute job
	if eErr := handler.Execute(ctx); eErr != nil {
		j.onError(ctx, jobLogger, handler, eErr)
		return nil
	}

	if err := handler.OnSuccess(ctx); err != nil {
		jobLogger.Infof("Error occurred while calling OnSuccess lifecycle function: %v\n", err)
	}

	return nil
}

func (j *JobExecutor) buildJobHandler(ctx context.Context, workspaceDir string, jobLogger *jobLogger) (JobHandler, error) {
	// Get Job
	job, err := j.client.GetJob(ctx, j.cfg.JobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job %v", err)
	}

	// Get Run
	run, err := j.client.GetRun(ctx, job.RunID)
	if err != nil {
		return nil, fmt.Errorf("failed to get run %v", err)
	}

	// Get workspace
	ws, err := j.client.GetWorkspace(ctx, job.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace %v", err)
	}

	cancellableCtx := j.createCancellableContext(ctx, jobLogger, run.Metadata.ID, job.MaxJobDuration)

	var handler JobHandler

	switch job.Type {
	case types.JobPlanType:
		handler = NewPlanHandler(cancellableCtx, j.cfg, workspaceDir, ws, run, jobLogger, j.client)
	case types.JobApplyType:
		handler = NewApplyHandler(cancellableCtx, j.cfg, workspaceDir, ws, run, jobLogger, j.client)
	default:
		j.logger.Infof("Invalid job type %s", job.Type)
	}

	return handler, err
}

func (j *JobExecutor) onError(ctx context.Context, jobLogger *jobLogger, jobImpl JobHandler, err error) {
	jobLogger.Errorf("Error occurred while executing run: %s\n", err)
	jobLogger.Flush()

	if err = jobImpl.OnError(ctx, err); err != nil {
		jobLogger.Errorf("Error occurred while handling job error: %v\n", err)
	}
}

func (j *JobExecutor) createCancellableContext(ctx context.Context, jobLogger *jobLogger, runID string, maxJobDuration int32) context.Context {
	// This will gracefully cancel the job after the job timeout is reached.
	cancellableCtx, cancelFunc := context.WithTimeout(ctx, time.Duration(maxJobDuration)*time.Minute)

	// Listen for cancellation
	go func() {

		// First stage: wait for graceful cancel request.
		for {
			// Check if context is cancelled
			if ctx.Err() != nil {
				return
			}

			cancelled, err := j.waitForJobCancellation(ctx)
			if err != nil {
				jobLogger.Infof("Received error when listening for job cancellation: %v \n", err)
				time.Sleep(waitCancelError)
				continue
			}
			if cancelled {
				jobLogger.Infof("Received job cancellation request\n")

				cancelFunc()
				// After a non-forced cancellation request, keep waiting but in the next loop.
				break
			}
		}

		// Second stage: wait for switch to forced cancel.
		for {
			run, err := j.client.GetRun(ctx, runID)
			if err != nil {
				if ctx.Err() != nil {
					// If the context is canceled, it means the run was already gracefully cancelled,
					// so there is no need to take any additional forced cancel action.
					return
				}

				jobLogger.Infof("Received error when listening for forced run cancellation: %v \n", err)
				time.Sleep(waitCancelError)
				continue
			}

			// If the cancellation was forced, this should kill the main process and force the run to terminate.
			if run.ForceCanceled {
				jobLogger.Errorf("Force canceled run ID %s", run.Metadata.ID)
				os.Exit(1)
			}

			time.Sleep(waitForcedCancel)
		}

	}()

	return cancellableCtx
}

func (j *JobExecutor) waitForJobCancellation(ctx context.Context) (bool, error) {
	eventChannel, err := j.client.SubscribeToJobCancellationEvent(ctx, j.cfg.JobID)
	if err == context.DeadlineExceeded || te.IsContextCanceledError(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	event := <-eventChannel
	if event == nil {
		return false, errors.New("channel closed")
	}
	return event.Job.CancelRequested, nil
}
