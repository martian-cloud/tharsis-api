package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/store"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

func TestMapJobStatusToPlanStatus(t *testing.T) {
	tests := []struct {
		jobStatus  models.JobStatus
		wantStatus models.PlanStatus
		wantOK     bool
	}{
		{models.JobQueued, models.PlanQueued, true},
		{models.JobRunning, models.PlanRunning, true},
		{models.JobFinished, models.PlanFinished, true},
		{models.JobFailed, models.PlanErrored, true},
		{models.JobCanceled, models.PlanCanceled, true},
		// JobPending maps to no node change.
		{models.JobPending, "", false},
		{models.JobCanceling, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.jobStatus), func(t *testing.T) {
			got, ok := mapJobStatusToPlanStatus(tt.jobStatus)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantStatus, got)
		})
	}
}

func TestMapJobStatusToApplyStatus(t *testing.T) {
	tests := []struct {
		jobStatus  models.JobStatus
		wantStatus models.ApplyStatus
		wantOK     bool
	}{
		{models.JobQueued, models.ApplyQueued, true},
		{models.JobRunning, models.ApplyRunning, true},
		{models.JobFinished, models.ApplyFinished, true},
		{models.JobFailed, models.ApplyErrored, true},
		{models.JobCanceled, models.ApplyCanceled, true},
		{models.JobPending, "", false},
		{models.JobCanceling, "", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.jobStatus), func(t *testing.T) {
			got, ok := mapJobStatusToApplyStatus(tt.jobStatus)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantStatus, got)
		})
	}
}

func TestSyncJobStatus_Execute_PlanRunning(t *testing.T) {
	ctx := context.Background()

	// Plan starts queued so that the JobRunning -> PlanRunning transition is legal.
	jobID := "job-1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued, LatestJobID: &jobID},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyCreated},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobPlanType, JobID: jobID, NewStatus: models.JobRunning, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.PlanRunning, run.Plan.Status)
	assert.Equal(t, models.RunPlanning, run.Status)
}

func TestSyncJobStatus_Execute_ApplyRunning(t *testing.T) {
	ctx := context.Background()

	jobID := "job-1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunApplyQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanFinished, HasChanges: true},
		Apply:    &models.Apply{ID: "apply-1", Status: models.ApplyQueued, LatestJobID: &jobID},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobApplyType, JobID: jobID, NewStatus: models.JobRunning, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.ApplyRunning, run.Apply.Status)
	assert.Equal(t, models.RunApplying, run.Status)
}

func TestSyncJobStatus_Execute_PendingIsNoOp(t *testing.T) {
	ctx := context.Background()

	// JobPending maps to no node status, so Execute should leave the run unchanged
	// and not attempt an (illegal) transition. The job matches the node's current job so
	// the no-op is driven by the pending mapping, not the staleness check.
	jobID := "job-1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued, LatestJobID: &jobID},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobPlanType, JobID: jobID, NewStatus: models.JobPending, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.PlanQueued, run.Plan.Status)
	assert.Equal(t, models.RunPlanQueued, run.Status)
}

func TestSyncJobStatus_Execute_StaleJobIgnored(t *testing.T) {
	ctx := context.Background()

	// The plan node's current job is "job-2" (e.g. after a retry), but a status
	// report arrives for the superseded "job-1". It must not be projected onto the node.
	latest := "job-2"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued, LatestJobID: &latest},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobPlanType, JobID: "job-1", NewStatus: models.JobRunning, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.PlanQueued, run.Plan.Status, "stale job must not transition the node")
	assert.Equal(t, models.RunPlanQueued, run.Status)
}

func TestSyncJobStatus_Execute_FinalStatusIgnored(t *testing.T) {
	ctx := context.Background()

	// The plan was force-canceled, so the node is already in a final status while its
	// LatestJobID still points at the running job. A late JobFinished report for that
	// same job must be a no-op rather than an illegal canceled -> finished transition.
	jobID := "job-1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunCanceled,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanCanceled, LatestJobID: &jobID},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobPlanType, JobID: jobID, NewStatus: models.JobFinished, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}), "late job status for a final node must be a no-op, not an error")
	assert.Equal(t, models.PlanCanceled, run.Plan.Status, "final node status must be unchanged")
	assert.Equal(t, models.RunCanceled, run.Status)
}

func TestSyncJobStatus_Execute_MatchingJobTransitions(t *testing.T) {
	ctx := context.Background()

	// The reporting job matches the node's current job, so the transition applies.
	latest := "job-1"
	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued, LatestJobID: &latest},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobPlanType, JobID: "job-1", NewStatus: models.JobRunning, PersistJob: func(context.Context) error { return nil }}

	require.NoError(t, cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore}))
	assert.Equal(t, models.PlanRunning, run.Plan.Status)
	assert.Equal(t, models.RunPlanning, run.Status)
}

func TestSyncJobStatus_Execute_UnknownJobType(t *testing.T) {
	ctx := context.Background()

	run := &models.Run{
		Metadata: models.ResourceMetadata{ID: "run-1"},
		Status:   models.RunPlanQueued,
		Plan:     models.Plan{ID: "plan-1", Status: models.PlanQueued},
	}

	runStore := store.NewRunStore(&db.Client{})
	runStore.AddRun(run)

	logr, _ := logger.NewForTest()
	cmd := &SyncJobStatus{logger: logr, RunID: "run-1", JobType: models.JobType("bogus"), NewStatus: models.JobRunning, PersistJob: func(context.Context) error { return nil }}

	err := cmd.Execute(ctx, &types.ExecuteInput{RunStore: runStore})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown job type")
}
