package commands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// jobWithStatus builds a job already in the given status (the initial assignment
// from the zero value is always allowed).
func jobWithStatus(id string, status models.JobStatus) *models.Job {
	j := &models.Job{Metadata: models.ResourceMetadata{ID: id}}
	_ = j.SetStatus(status)
	return j
}

func TestCancelLatestJob(t *testing.T) {
	ctx := context.Background()

	t.Run("nil job id cancels the node immediately", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs}, nil, false)
		require.NoError(t, err)
		assert.True(t, cancelNode)
	})

	t.Run("already-final job cancels the node immediately", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		id := "job-1"
		mockJobs.On("GetJobByID", ctx, id).Return(jobWithStatus(id, models.JobFinished), nil)

		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs}, &id, false)
		require.NoError(t, err)
		assert.True(t, cancelNode)
	})

	t.Run("graceful cancel of a running job requests cancellation but leaves the node", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		id := "job-1"
		job := jobWithStatus(id, models.JobRunning)
		mockJobs.On("GetJobByID", ctx, id).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs}, &id, false)
		require.NoError(t, err)
		// The runner must confirm the cancellation, so the node is not transitioned now.
		assert.False(t, cancelNode)
		assert.Equal(t, models.JobCanceling, job.GetStatus())
		assert.NotNil(t, job.CancelRequestedTimestamp)
		assert.False(t, job.ForceCanceled)
	})

	t.Run("force cancel of a running job stops it and cancels the node", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		id := "job-1"
		job := jobWithStatus(id, models.JobRunning)
		mockJobs.On("GetJobByID", ctx, id).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		mockLogStreams := db.NewMockLogStreams(t)
		mockLogStreams.On("GetLogStreamByJobID", ctx, id).Return(&models.LogStream{}, nil)
		mockLogStreams.On("UpdateLogStream", ctx, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Completed
		})).Return(&models.LogStream{}, nil)

		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs, LogStreams: mockLogStreams}, &id, true)
		require.NoError(t, err)
		assert.True(t, cancelNode)
		assert.Equal(t, models.JobCanceled, job.GetStatus())
		assert.True(t, job.ForceCanceled)
	})

	t.Run("graceful cancel of a queued job stops it immediately, cancels the node, and completes the log stream", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		id := "job-1"
		job := jobWithStatus(id, models.JobQueued)
		mockJobs.On("GetJobByID", ctx, id).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		// A queued job is canceled without a runner ever reporting on it, so the
		// cancel must mark the job's log stream completed itself.
		mockLogStreams := db.NewMockLogStreams(t)
		mockLogStreams.On("GetLogStreamByJobID", ctx, id).Return(&models.LogStream{}, nil)
		mockLogStreams.On("UpdateLogStream", ctx, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Completed
		})).Return(&models.LogStream{}, nil)

		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs, LogStreams: mockLogStreams}, &id, false)
		require.NoError(t, err)
		assert.True(t, cancelNode)
		assert.Equal(t, models.JobCanceled, job.GetStatus())
		assert.NotNil(t, job.CancelRequestedTimestamp)
		mockLogStreams.AssertExpectations(t)
	})

	t.Run("queued job cancel tolerates a missing log stream", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		id := "job-1"
		job := jobWithStatus(id, models.JobQueued)
		mockJobs.On("GetJobByID", ctx, id).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		mockLogStreams := db.NewMockLogStreams(t)
		mockLogStreams.On("GetLogStreamByJobID", ctx, id).Return(nil, nil)

		cancelNode, err := cancelLatestJob(ctx, &db.Client{Jobs: mockJobs, LogStreams: mockLogStreams}, &id, false)
		require.NoError(t, err)
		assert.True(t, cancelNode)
		mockLogStreams.AssertNotCalled(t, "UpdateLogStream", mock.Anything, mock.Anything)
	})
}

func TestCancelActivePhase(t *testing.T) {
	ctx := context.Background()

	t.Run("acts on the plan when the plan is not final", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		planJobID := "plan-job"
		job := jobWithStatus(planJobID, models.JobQueued)
		mockJobs.On("GetJobByID", ctx, planJobID).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		mockLogStreams := db.NewMockLogStreams(t)
		mockLogStreams.On("GetLogStreamByJobID", ctx, planJobID).Return(&models.LogStream{}, nil)
		mockLogStreams.On("UpdateLogStream", ctx, mock.Anything).Return(&models.LogStream{}, nil)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanQueued,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued, LatestJobID: &planJobID},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
		}

		changes, err := cancelActivePhase(ctx, &db.Client{Jobs: mockJobs, LogStreams: mockLogStreams}, run, false)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
	})

	t.Run("acts on the apply when the plan is final and apply is not", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		applyJobID := "apply-job"
		job := jobWithStatus(applyJobID, models.JobQueued)
		mockJobs.On("GetJobByID", ctx, applyJobID).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		mockLogStreams := db.NewMockLogStreams(t)
		mockLogStreams.On("GetLogStreamByJobID", ctx, applyJobID).Return(&models.LogStream{}, nil)
		mockLogStreams.On("UpdateLogStream", ctx, mock.Anything).Return(&models.LogStream{}, nil)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunApplyQueued,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
			Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyQueued, LatestJobID: &applyJobID},
		}

		changes, err := cancelActivePhase(ctx, &db.Client{Jobs: mockJobs, LogStreams: mockLogStreams}, run, false)
		require.NoError(t, err)
		assert.NotEmpty(t, changes)
	})

	t.Run("graceful cancel of a running plan job returns no node changes", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		planJobID := "plan-job"
		job := jobWithStatus(planJobID, models.JobRunning)
		mockJobs.On("GetJobByID", ctx, planJobID).Return(job, nil)
		mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlanning,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanRunning, LatestJobID: &planJobID},
		}

		changes, err := cancelActivePhase(ctx, &db.Client{Jobs: mockJobs}, run, false)
		require.NoError(t, err)
		assert.Nil(t, changes)
	})

	t.Run("no active phase returns no changes", func(t *testing.T) {
		mockJobs := db.NewMockJobs(t)
		run := &models.Run{
			Metadata: models.ResourceMetadata{ID: "run-1"},
			Status:   models.RunPlannedAndFinished,
			Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished},
		}

		changes, err := cancelActivePhase(ctx, &db.Client{Jobs: mockJobs}, run, false)
		require.NoError(t, err)
		assert.Nil(t, changes)
	})
}

func TestCancelRun_Execute_InvalidStates(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name   string
		status models.RunStatus
	}{
		{"planned and finished", models.RunPlannedAndFinished},
		{"applied", models.RunApplied},
		{"errored", models.RunErrored},
		{"already canceled", models.RunCanceled},
		{"discarded", models.RunDiscarded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			run := &models.Run{Metadata: models.ResourceMetadata{ID: "run-1"}, Status: tt.status}
			runStore := store.NewRunStore(&db.Client{})
			runStore.AddRun(run)

			cmd := &CancelRun{in: &CancelRunInput{RunID: "run-1"}}
			err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
			require.Error(t, err)
			assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
		})
	}
}

func TestCancelRun_Execute_ForceWithoutPriorGraceful(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanning,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanRunning},
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &CancelRun{in: &CancelRunInput{RunID: "run-1", Force: true}}
	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	assert.Contains(t, err.Error(), "graceful")
}

func TestCancelRun_Execute_ForceBeforeWaitElapsed(t *testing.T) {
	ctx := context.Background()

	future := time.Now().UTC().Add(time.Hour)
	run := &models.Run{
		Metadata:               models.ResourceMetadata{ID: "run-1"},
		Status:                 models.RunPlanning,
		Plan:                   models.Plan{ID: "plan-1", Status: models.PlanRunning},
		ForceCancelAvailableAt: &future,
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &CancelRun{in: &CancelRunInput{RunID: "run-1", Force: true}}
	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
	assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
}

func TestCancelRun_Execute_GracefulSetsForceCancelAvailableAt(t *testing.T) {
	ctx := auth.WithCaller(context.Background(), auth.NewServiceAccountCaller("sa-1", "sa/path", nil, nil, nil))

	// Graceful cancel of a running plan job: the job moves to canceling, the node is
	// left running, and force-cancel becomes available after the delay.
	mockJobs := db.NewMockJobs(t)
	planJobID := "plan-job"
	job := jobWithStatus(planJobID, models.JobRunning)
	mockJobs.On("GetJobByID", ctx, planJobID).Return(job, nil)
	mockJobs.On("UpdateJob", ctx, job).Return(job, nil)

	mockActivityEvents := db.NewMockActivityEvents(t)
	// The cancel is recorded as an UPDATE activity event whose payload identifies
	// the run-update sub-action (cancel).
	mockActivityEvents.On("CreateActivityEvent", mock.Anything, mock.MatchedBy(func(in *models.ActivityEvent) bool {
		var payload models.ActivityEventUpdateRunPayload
		if err := json.Unmarshal(in.Payload, &payload); err != nil {
			return false
		}
		return in.Action == models.ActionUpdate &&
			in.TargetType == models.TargetRun &&
			payload.Type == string(models.RunUpdateTypeCancel)
	})).Return(&models.ActivityEvent{}, nil)

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanning,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanRunning, LatestJobID: &planJobID},
	}
	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	cmd := &CancelRun{
		dbClient:      &db.Client{Jobs: mockJobs, ActivityEvents: mockActivityEvents},
		in:            &CancelRunInput{RunID: "run-1", CanceledBy: "user@example.com"},
		namespacePath: "groupA/ws",
	}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	require.NotNil(t, cmd.Updated)
	assert.NotNil(t, run.ForceCancelAvailableAt)
	assert.False(t, run.ForceCanceled)
	// Plan node still running (runner must confirm), job requested to cancel.
	assert.Equal(t, models.PlanRunning, run.Plan.Status)
	assert.Equal(t, models.JobCanceling, job.GetStatus())
}
