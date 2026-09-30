package runner

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/runner/jobdispatcher"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	// cleanupPollMinInterval and cleanupPollMaxInterval bound the jittered interval between cleanup
	// polls; jitter spreads DB load when many runners poll the same API. Jobs don't need to be reaped
	// immediately, so the interval is kept long to keep load on the API and DB low.
	cleanupPollMinInterval = 5 * time.Minute
	cleanupPollMaxInterval = 10 * time.Minute
	// cleanupBatchLimit bounds how many jobs are claimed for cleanup per poll.
	cleanupBatchLimit = 100
)

var (
	jobsCleanedUpCount = metric.NewCounter("jobs_cleaned_up_count", "Number of jobs whose runtime was cleaned up.")
	jobCleanupFails    = metric.NewCounter("job_cleanup_fails_count", "Number of job cleanup failures.")
)

// cleanupWorker periodically reaps the runtimes of this runner's final jobs, claiming them from the
// API and delegating teardown to the dispatcher.
type cleanupWorker struct {
	client     Client
	dispatcher jobdispatcher.JobDispatcher
	logger     logger.Logger
	runnerID   string
	sessionID  string
}

// newCleanupWorker creates a cleanupWorker for the given runner and dispatcher.
func newCleanupWorker(runnerID string, sessionID string, client Client, dispatcher jobdispatcher.JobDispatcher, logger logger.Logger) *cleanupWorker {
	return &cleanupWorker{
		client:     client,
		dispatcher: dispatcher,
		logger:     logger,
		runnerID:   runnerID,
		sessionID:  sessionID,
	}
}

// start runs the cleanup loop until ctx is canceled.
func (w *cleanupWorker) start(ctx context.Context) {
	w.logger.Info("Runner job cleanup worker started")
	defer w.logger.Info("Runner job cleanup worker stopped")

	for {
		select {
		case <-ctx.Done():
			return
		// Jitter the poll interval so runners don't all hit the API simultaneously.
		case <-time.After(cleanupPollMinInterval + time.Duration(rand.Int64N(int64(cleanupPollMaxInterval-cleanupPollMinInterval)+1))):
			if err := w.cleanupJobs(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				w.handleError(ctx, fmt.Errorf("failed to clean up jobs: %v", err))
			}
		}
	}
}

// cleanupJobs claims a batch of jobs needing cleanup and tears each one down via the dispatcher.
// Cleanup is best-effort: every attempted job is marked done in a single call regardless of the
// outcome, so a resource that can't be deleted (already gone, or a persistent error) doesn't cause
// the job to be re-claimed and retried forever. Failures are recorded as runner session errors and
// counted for alerting.
func (w *cleanupWorker) cleanupJobs(ctx context.Context) error {
	jobs, err := w.client.ClaimJobsForCleanup(ctx, &ClaimJobsForCleanupInput{
		RunnerID: w.runnerID,
		Limit:    cleanupBatchLimit,
	})
	if err != nil {
		return fmt.Errorf("failed to claim jobs for cleanup: %w", err)
	}

	cleanedJobIDs := make([]string, 0, len(jobs))
	for _, j := range jobs {
		if err := w.dispatcher.CleanupJob(ctx, j.JobID, j.DispatcherData); err != nil {
			// Abandon the batch on shutdown rather than marking runtimes that were never deleted as cleaned up.
			if ctx.Err() != nil {
				return ctx.Err()
			}

			jobCleanupFails.Inc()
			w.handleError(ctx, fmt.Errorf("failed to clean up job %s: %v", j.JobID, err))
		} else {
			jobsCleanedUpCount.Inc()
		}

		// Mark the job done even on failure; retrying a persistent failure every pass never drains.
		cleanedJobIDs = append(cleanedJobIDs, j.JobID)
	}

	if len(cleanedJobIDs) > 0 {
		if err := w.client.MarkJobsCleanedUp(ctx, w.runnerID, cleanedJobIDs); err != nil {
			jobCleanupFails.Inc()
			return fmt.Errorf("failed to mark jobs cleaned up: %w", err)
		}
	}

	return nil
}

// handleError logs err and records it against the runner session so cleanup failures are visible.
func (w *cleanupWorker) handleError(ctx context.Context, err error) {
	w.logger.Error(err)
	if sErr := w.client.CreateRunnerSessionError(ctx, w.sessionID, err); sErr != nil {
		w.logger.Errorf("failed to send error %v", sErr)
	}
}
