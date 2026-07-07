package eventhandlers

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/metric"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

var (
	planExecutionTime  = metric.NewHistogram("plan_execution_time", "Amount of time a plan took to execute.", 1, 2, 10)
	applyExecutionTime = metric.NewHistogram("apply_execution_time", "Amount of time an apply took to execute.", 1, 2, 10)

	planCompletedCount  = metric.NewCounter("plan_completed_count", "Amount of times a plan is completed.")
	applyCompletedCount = metric.NewCounter("apply_completed_count", "Amount of times an apply is completed.")
	runCompletedCount   = metric.NewCounter("run_completed_count", "Amount of times a run is completed.")
)

// RunMetricsHandler emits Prometheus metrics as runs and their plan/apply nodes
// reach a terminal state. It is registered as a stateless (post-commit) handler so
// metrics are emitted once per durable change rather than re-emitted on
// optimistic-lock retries.
type RunMetricsHandler struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewRunMetricsHandler creates a new RunMetricsHandler.
func NewRunMetricsHandler(logger logger.Logger, dbClient *db.Client) *RunMetricsHandler {
	return &RunMetricsHandler{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler. Metrics are best-effort: any failure
// is logged and never returned, so a metric error cannot fail an already-committed
// change.
func (h *RunMetricsHandler) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	for _, change := range changes {
		for _, statusChange := range change.NodeStatusChanges {
			switch statusChange.GetNodeType() {
			case statemachine.RunNodeType:
				if rc, ok := statusChange.(statemachine.RunStatusChange); ok && rc.NewStatus.IsFinalStatus() {
					runCompletedCount.Inc()
				}
			case statemachine.PlanNodeType:
				if pc, ok := statusChange.(statemachine.PlanStatusChange); ok && pc.NewStatus.IsFinalStatus() {
					planCompletedCount.Inc()
					h.observeExecutionTime(ctx, change.Run.Plan.LatestJobID, planExecutionTime)
				}
			case statemachine.ApplyNodeType:
				if ac, ok := statusChange.(statemachine.ApplyStatusChange); ok && ac.NewStatus.IsFinalStatus() {
					applyCompletedCount.Inc()
					if change.Run.Apply != nil {
						h.observeExecutionTime(ctx, change.Run.Apply.LatestJobID, applyExecutionTime)
					}
				}
			}
		}
	}
	return nil
}

// observeExecutionTime records the running->finished duration (in minutes) of the
// node's latest job. A job that never started running (no running timestamp) is
// skipped. Errors are logged, not returned.
func (h *RunMetricsHandler) observeExecutionTime(ctx context.Context, jobID *string, histogram prometheus.Histogram) {
	if jobID == nil {
		return
	}

	job, err := h.dbClient.Jobs.GetJobByID(ctx, *jobID)
	if err != nil {
		h.logger.WithContextFields(ctx).Errorf("run metrics handler failed to get job %s: %v", *jobID, err)
		return
	}
	if job == nil {
		return
	}

	running := job.Timestamps.RunningTimestamp
	finished := job.Timestamps.FinishedTimestamp
	if running == nil || finished == nil {
		return
	}

	histogram.Observe(finished.Sub(*running).Minutes())
}
