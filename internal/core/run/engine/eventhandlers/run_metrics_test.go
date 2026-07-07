package eventhandlers

import (
	"context"
	"testing"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// counterValue reads the current value of a prometheus counter. It avoids a
// dependency on prometheus/client_golang/prometheus/testutil so the test does not
// pull new transitive modules into go.mod.
func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}

func TestRunMetricsHandler_HandleRunChanges_IncrementsCounters(t *testing.T) {
	logr, _ := logger.NewForTest()

	// A job with both timestamps so observeExecutionTime records a value.
	running := time.Now().UTC()
	finished := running.Add(2 * time.Minute)
	job := &models.Job{
		Metadata: models.ResourceMetadata{ID: "job-1"},
		Timestamps: models.JobTimestamps{
			RunningTimestamp:  &running,
			FinishedTimestamp: &finished,
		},
	}

	mockJobs := db.NewMockJobs(t)
	mockJobs.On("GetJobByID", mock.Anything, "job-1").Return(job, nil)

	dbClient := &db.Client{Jobs: mockJobs}
	handler := NewRunMetricsHandler(logr, dbClient)

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Plan:     models.Plan{LatestJobID: ptr.String("job-1")},
		Apply:    &models.Apply{LatestJobID: ptr.String("job-1")},
	}

	changes := []types.RunChange{{
		Run: run,
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.RunStatusChange{NewStatus: models.RunApplied},
			statemachine.PlanStatusChange{NewStatus: models.PlanFinished},
			statemachine.ApplyStatusChange{NewStatus: models.ApplyFinished},
		},
	}}

	beforeRun := counterValue(t, runCompletedCount)
	beforePlan := counterValue(t, planCompletedCount)
	beforeApply := counterValue(t, applyCompletedCount)

	require.NoError(t, handler.HandleRunChanges(context.Background(), changes))

	assert.Equal(t, beforeRun+1, counterValue(t, runCompletedCount))
	assert.Equal(t, beforePlan+1, counterValue(t, planCompletedCount))
	assert.Equal(t, beforeApply+1, counterValue(t, applyCompletedCount))
}

func TestRunMetricsHandler_HandleRunChanges_NonFinalStatusesDoNotIncrement(t *testing.T) {
	logr, _ := logger.NewForTest()

	// No job lookups should occur, so a bare mock with no expectations suffices.
	dbClient := &db.Client{Jobs: db.NewMockJobs(t)}
	handler := NewRunMetricsHandler(logr, dbClient)

	changes := []types.RunChange{{
		Run: &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Apply: &models.Apply{}},
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.RunStatusChange{NewStatus: models.RunPlanning},
			statemachine.PlanStatusChange{NewStatus: models.PlanRunning},
			statemachine.ApplyStatusChange{NewStatus: models.ApplyRunning},
		},
	}}

	beforeRun := counterValue(t, runCompletedCount)
	beforePlan := counterValue(t, planCompletedCount)
	beforeApply := counterValue(t, applyCompletedCount)

	require.NoError(t, handler.HandleRunChanges(context.Background(), changes))

	assert.Equal(t, beforeRun, counterValue(t, runCompletedCount))
	assert.Equal(t, beforePlan, counterValue(t, planCompletedCount))
	assert.Equal(t, beforeApply, counterValue(t, applyCompletedCount))
}

func TestRunMetricsHandler_HandleRunChanges_ApplyFinalWithNilApplyDoesNotPanic(t *testing.T) {
	logr, _ := logger.NewForTest()
	dbClient := &db.Client{Jobs: db.NewMockJobs(t)}
	handler := NewRunMetricsHandler(logr, dbClient)

	// Apply node reports a final status but the run has no Apply struct: the handler
	// guards on change.Run.Apply != nil before dereferencing, so this must not panic
	// and must still increment the apply counter.
	changes := []types.RunChange{{
		Run: &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}},
		NodeStatusChanges: []statemachine.NodeStatusChange{
			statemachine.ApplyStatusChange{NewStatus: models.ApplyFinished},
		},
	}}

	before := counterValue(t, applyCompletedCount)
	require.NotPanics(t, func() {
		require.NoError(t, handler.HandleRunChanges(context.Background(), changes))
	})
	assert.Equal(t, before+1, counterValue(t, applyCompletedCount))
}

func TestRunMetricsHandler_observeExecutionTime(t *testing.T) {
	logr, _ := logger.NewForTest()

	running := time.Now().UTC()
	finished := running.Add(3 * time.Minute)

	type testCase struct {
		name      string
		jobID     *string
		job       *models.Job
		jobErr    error
		expectGet bool
	}

	testCases := []testCase{
		{
			name:      "nil job id is skipped without a db call",
			jobID:     nil,
			expectGet: false,
		},
		{
			name:      "job with running and finished timestamps observes",
			jobID:     ptr.String("job-1"),
			expectGet: true,
			job: &models.Job{
				Metadata: models.ResourceMetadata{ID: "job-1"},
				Timestamps: models.JobTimestamps{
					RunningTimestamp:  &running,
					FinishedTimestamp: &finished,
				},
			},
		},
		{
			name:      "job missing timestamps does not error",
			jobID:     ptr.String("job-1"),
			expectGet: true,
			job: &models.Job{
				Metadata:   models.ResourceMetadata{ID: "job-1"},
				Timestamps: models.JobTimestamps{},
			},
		},
		{
			name:      "nil job does not error",
			jobID:     ptr.String("job-1"),
			expectGet: true,
			job:       nil,
		},
		{
			name:      "db error is swallowed",
			jobID:     ptr.String("job-1"),
			expectGet: true,
			jobErr:    assert.AnError,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			mockJobs := db.NewMockJobs(t)
			if test.expectGet {
				mockJobs.On("GetJobByID", mock.Anything, "job-1").Return(test.job, test.jobErr)
			}
			handler := NewRunMetricsHandler(logr, &db.Client{Jobs: mockJobs})

			require.NotPanics(t, func() {
				handler.observeExecutionTime(context.Background(), test.jobID, planExecutionTime)
			})
		})
	}
}
