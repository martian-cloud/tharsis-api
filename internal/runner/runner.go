package runner

import (
	"context"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/run"
)

const (
	checkRunsInterval = 1 * time.Second
	runnerID          = "6ba7b812-9dad-11d1-80b4-00c04fd430c8"
)

// Runner is used to run terraform commands on a workspace
type Runner struct {
	runService          run.Service
	jobDispatcherPlugin jobdispatcher.JobDispatcher
	logger              logger.Logger
}

var (
	// runnerID will a constant UUID until runner registration is allowed.
	claimJobCount = metric.NewCounter("claim_job_count", "Amount of jobs claimed.")
	claimJobFails = metric.NewCounter("claim_job_fails_count", "Amount of jobs claims failed.")

	launchJobFails = metric.NewCounter("launch_job_fails", "Amount of launch jobs failed.")

	jobDispatchCount = metric.NewCounter("job_dispatch_count", "Amount of jobs dispatched.")
	jobDispatchTime  = metric.NewHistogram("job_dispatch_time", "Amount of time a job took to dispatch.", 1, 2, 8)
)

// NewRunner creates a new Runner
func NewRunner(
	runService run.Service,
	jobDispatcherPlugin jobdispatcher.JobDispatcher,
	logger logger.Logger,
) *Runner {
	return &Runner{runService, jobDispatcherPlugin, logger}
}

// Start will start the runner so it can begin picking up runs
func (r *Runner) Start(ctx context.Context) {
	go func() {
		for {
			r.logger.Info("Waiting for next available run")

			resp, err := r.runService.ClaimJob(ctx, runnerID)
			claimJobCount.Inc()
			if err != nil {
				claimJobFails.Inc()
				r.logger.Errorf("Failed to request next available job %v", err)
			} else {
				r.logger.Infof("Claimed job with ID %s", resp.Job.Metadata.ID)

				if err := r.launchJob(ctx, resp.Job, resp.Token); err != nil {
					launchJobFails.Inc()
					r.logger.Errorf("Failed to launch job %v", err)
				}
			}

			time.Sleep(checkRunsInterval)
		}
	}()
}

func (r *Runner) launchJob(ctx context.Context, job *models.Job, token string) error {
	// For measuring dispatch time in seconds.
	start := time.Now()
	executorID, err := r.jobDispatcherPlugin.DispatchJob(ctx, gid.ToGlobalID(gid.JobType, job.Metadata.ID), token)
	duration := time.Since(start)
	jobDispatchTime.Observe(float64(duration.Seconds()))
	jobDispatchCount.Inc()
	if err != nil {
		return err
	}

	r.logger.Infof("Job %s running in executor %s", job.Metadata.ID, executorID)

	return nil
}
