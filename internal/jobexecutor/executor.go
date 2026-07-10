package jobexecutor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	humanize "github.com/dustin/go-humanize"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/jobclient"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/jobexecutor/joblogger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// waitCancelError is the duration to sleep if cancel wait gets error
	waitCancelError = 10 * time.Second

	// waitForcedCancel is the duration to sleep between polls looking for forced cancel.
	waitForcedCancel = 30 * time.Second

	// successIcon is the icon used for success messages
	successIcon = "\u2705"

	// failureIcon is the icon used for failure messages
	failureIcon = "\u274c"
)

var errJobAlreadyCanceled = errors.New("job already canceled")

// JobHandler contains the job lifecycle functions
type JobHandler interface {
	Execute(ctx context.Context) error
	OnError(ctx context.Context, err error)
	Cleanup(ctx context.Context) error
}

// JobConfig is used to configure the job
type JobConfig struct {
	JobID                  string
	APIEndpoint            string
	JobToken               string
	DiscoveryProtocolHosts []string
}

// JobExecutor executes a job
type JobExecutor struct {
	cfg            *JobConfig
	client         jobclient.Client
	logger         logger.Logger
	cancellableCtx context.Context
	cancelFunc     context.CancelFunc
	version        string
}

// NewJobExecutor creates a new JobExecutor
func NewJobExecutor(
	ctx context.Context,
	cfg *JobConfig,
	client jobclient.Client,
	logger logger.Logger,
	version string,
) *JobExecutor {

	ctx, cancel := context.WithCancel(ctx)

	return &JobExecutor{
		cfg:            cfg,
		client:         client,
		logger:         logger,
		version:        version,
		cancellableCtx: ctx,
		cancelFunc:     cancel,
	}
}

// Execute executes the job associated with the JobExecutor instance
func (j *JobExecutor) Execute(ctx context.Context) error {
	jobLogger, err := joblogger.NewLogger(j.cfg.JobID, j.client, j.logger)
	if err != nil {
		return fmt.Errorf("failed to create job logger %v", err)
	}

	defer jobLogger.Close()

	jobLogger.Start()

	// Add a defer to handle any panics that may occur during job execution
	defer func() {
		if rErr := recover(); rErr != nil {
			j.handleJobFailureWithError(ctx, jobLogger, fmt.Errorf("job panic: %v", rErr))
		}
	}()

	err = j.execute(ctx, jobLogger)

	if err == errJobAlreadyCanceled {
		return nil
	} else if j.cancellableCtx.Err() != nil {
		j.handleJobCanceled(ctx, jobLogger)
	} else if err != nil {
		j.handleJobFailureWithError(ctx, jobLogger, err)
	} else {
		jobLogger.Flush()
		if _, err := j.client.SetJobStatus(ctx, j.cfg.JobID, pb.JobStatus_finished, models.CurrentJobProtocolVersion); err != nil {
			return fmt.Errorf("failed to set job status to succeeded: %v", err)
		}
	}

	return nil
}

func (j *JobExecutor) handleJobFailureWithError(ctx context.Context, jobLogger joblogger.Logger, err error) {
	jobLogger.Errorf("Error occurred while executing job: %v", err)

	j.handleJobFailure(ctx, jobLogger)
}

func (j *JobExecutor) handleJobFailure(ctx context.Context, jobLogger joblogger.Logger) {
	jobLogger.Flush()

	if _, err := j.client.SetJobStatus(ctx, j.cfg.JobID, pb.JobStatus_failed, models.CurrentJobProtocolVersion); err != nil {
		j.logger.Errorf("failed to set job status to failed: %v", err)
	}
}

func (j *JobExecutor) handleJobCanceled(ctx context.Context, jobLogger joblogger.Logger) {
	jobLogger.Flush()

	if _, err := j.client.SetJobStatus(ctx, j.cfg.JobID, pb.JobStatus_canceled, models.CurrentJobProtocolVersion); err != nil {
		j.logger.Errorf("failed to set job status to canceled: %v", err)
	}
}

func (j *JobExecutor) execute(ctx context.Context, jobLogger joblogger.Logger) error {
	// Set job status to running
	if _, err := j.client.SetJobStatus(ctx, j.cfg.JobID, pb.JobStatus_running, models.CurrentJobProtocolVersion); err != nil {
		if status.Code(err) == codes.InvalidArgument {
			// Check if job is already canceled
			job, err := j.client.GetJob(ctx, j.cfg.JobID)
			if err != nil {
				return fmt.Errorf("failed to get job %v", err)
			}
			if job.Status == pb.JobStatus_canceled {
				return errJobAlreadyCanceled
			}
		}
		return fmt.Errorf("failed to set job status to running: %v", err)
	}

	jobLogger.Infof("Job executor version %s", j.version)
	jobLogger.Infof("Starting job %s", j.cfg.JobID)

	// Get Job
	job, err := j.client.GetJob(ctx, j.cfg.JobID)
	if err != nil {
		return fmt.Errorf("failed to get job %v", err)
	}

	j.startCancellationMonitor(ctx, jobLogger, job)

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
	if memoryLimit > 0 {
		memoryMonitor, err := NewMemoryMonitor(jobLogger, memoryLimit)
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

	// Build job
	handler, err := j.buildJobHandler(ctx, workspaceDir, jobLogger, job)
	if err != nil {
		return err
	}
	defer func() {
		if err = handler.Cleanup(ctx); err != nil {
			j.logger.Infof("Error occurred while cleaning up job: %v\n", err)
		}
	}()

	// Execute job
	eErr := handler.Execute(ctx)

	if eErr != nil && j.cancellableCtx.Err() == nil {
		handler.OnError(ctx, eErr)
	}

	return eErr
}

func (j *JobExecutor) buildJobHandler(ctx context.Context, workspaceDir string, jobLogger joblogger.Logger, job *pb.Job) (JobHandler, error) {
	// Get Run
	run, err := j.client.GetRun(ctx, job.RunId)
	if err != nil {
		return nil, fmt.Errorf("failed to get run %v", err)
	}

	// Get workspace
	ws, err := j.client.GetWorkspace(ctx, job.WorkspaceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace %v", err)
	}

	var handler JobHandler

	switch job.Type {
	case pb.JobType_plan.String():
		handler, err = NewPlanHandler(j.cancellableCtx, j.cfg, workspaceDir, ws, run, job, j.logger, jobLogger, j.client)
		if err != nil {
			return nil, err
		}
	case pb.JobType_apply.String():
		handler, err = NewApplyHandler(j.cancellableCtx, j.cfg, workspaceDir, ws, run, job, j.logger, jobLogger, j.client)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid job type %s", job.Type)
	}

	return handler, err
}

func (j *JobExecutor) startCancellationMonitor(
	ctx context.Context,
	jobLogger joblogger.Logger,
	job *pb.Job,
) {
	// Listen for job timeout
	go func() {
		select {
		case <-j.cancellableCtx.Done():
		case <-time.After(time.Duration(job.MaxJobDuration) * time.Minute):
			jobLogger.Warningf("Max job duration exceeded. Job will gracefully cancel now...")
			// Cancel job since job timeout has been reached
			j.cancelFunc()
		}
	}()

	// Listen for graceful cancel
	go func() {
		for {
			if j.cancellableCtx.Err() != nil {
				// Context canceled
				return
			}

			canceled, err := j.waitForJobCancellation(ctx)
			if err != nil {
				if ctx.Err() != nil {
					// Job is shutting down.
					return
				}

				jobLogger.Errorf("Received error when listening for graceful job cancellation: %v", err)
				time.Sleep(waitCancelError)
				continue
			}

			if canceled {
				jobLogger.Warningf("Received a graceful job cancel request")

				// Cancel the context
				j.cancelFunc()

				return
			}
		}
	}()

	// Listen for force cancel
	go func() {
		// Wait for graceful cancel before checking for force cancel
		<-j.cancellableCtx.Done()

		for {
			if ctx.Err() != nil {
				// If the context is canceled, it means the job is already shutting down
				return
			}

			job, err := j.client.GetJob(ctx, job.Metadata.Id)
			if err != nil {
				if ctx.Err() != nil {
					// If the context is canceled, it means the run was already gracefully cancelled,
					// so there is no need to take any additional forced cancel action.
					return
				}

				jobLogger.Errorf("Received error when listening for forced job cancellation: %v \n", err)
				time.Sleep(waitCancelError)
				continue
			}

			// If the cancellation was forced, this should kill the main process and force the run to terminate.
			if job.ForceCanceled {
				jobLogger.Warningf("Received force cancel request for this job")
				os.Exit(1)
			}

			time.Sleep(waitForcedCancel)
		}
	}()
}

func (j *JobExecutor) waitForJobCancellation(ctx context.Context) (bool, error) {
	canceled := false

	// StreamWithReconnect keeps the subscription alive across server drains, so no outer
	// timeout/re-subscribe loop is needed here.
	err := client.StreamWithReconnect(ctx,
		func(streamCtx context.Context) (grpc.ServerStreamingClient[pb.JobCancellationEvent], error) {
			return j.client.SubscribeToJobCancellationEvent(streamCtx, j.cfg.JobID)
		},
		func(event *pb.JobCancellationEvent) (bool, error) {
			canceled = event.Job.Status == pb.JobStatus_canceling || event.Job.Status == pb.JobStatus_canceled
			return canceled, nil
		},
	)

	return canceled, err
}
