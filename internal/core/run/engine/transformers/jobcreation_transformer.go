package transformers

import (
	"context"
	"strconv"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/namespace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// JobCreationTransformer creates a job for each plan/apply node that transitioned
// to the queued state and links it to the node via LatestJobID. It is a
// transformer (not a change handler) because it mutates the node's LatestJobID,
// which is then persisted with the rest of the run when the pipeline saves it.
type JobCreationTransformer struct {
	dbClient                  *db.Client
	inheritedSettingsResolver namespace.InheritedSettingResolver
}

// NewJobCreationTransformer creates a new JobCreationTransformer.
func NewJobCreationTransformer(dbClient *db.Client, inheritedSettingsResolver namespace.InheritedSettingResolver) *JobCreationTransformer {
	return &JobCreationTransformer{
		dbClient:                  dbClient,
		inheritedSettingsResolver: inheritedSettingsResolver,
	}
}

// Transform creates jobs for nodes that have transitioned to queued.
func (t *JobCreationTransformer) Transform(ctx context.Context, changeList []types.RunChange, _ types.RunStore) error {
	for _, change := range changeList {
		run := change.Run
		for _, sc := range change.NodeStatusChanges {
			switch c := sc.(type) {
			case statemachine.PlanStatusChange:
				if c.NewStatus != models.PlanQueued {
					continue
				}
				job, err := t.createJob(ctx, run, models.JobPlanType)
				if err != nil {
					return err
				}
				run.Plan.LatestJobID = &job.Metadata.ID
			case statemachine.ApplyStatusChange:
				if c.NewStatus != models.ApplyQueued || run.Apply == nil {
					continue
				}
				job, err := t.createJob(ctx, run, models.JobApplyType)
				if err != nil {
					return err
				}
				run.Apply.LatestJobID = &job.Metadata.ID
			}
		}
	}
	return nil
}

// createJob creates a job and its log stream for a queued node.
func (t *JobCreationTransformer) createJob(ctx context.Context, run *models.Run, jobType models.JobType) (*models.Job, error) {
	ws, err := t.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace")
	}
	if ws == nil {
		return nil, errors.New("workspace %s not found", run.WorkspaceID)
	}
	if ws.MaxJobDuration == nil {
		return nil, errors.New("workspace %s has no max job duration configured", ws.Metadata.ID)
	}

	runnerTagsSetting, err := t.inheritedSettingsResolver.GetRunnerTags(ctx, ws)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get runner tags")
	}

	providerMirrorSetting, err := t.inheritedSettingsResolver.GetProviderMirrorEnabled(ctx, ws)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get provider mirror enabled setting")
	}

	now := time.Now().UTC()
	job := &models.Job{
		Type:        jobType,
		WorkspaceID: run.WorkspaceID,
		RunID:       run.Metadata.ID,
		Timestamps: models.JobTimestamps{
			QueuedTimestamp: &now,
		},
		MaxJobDuration: *ws.MaxJobDuration,
		Tags:           runnerTagsSetting.Value,
		Properties: map[string]string{
			models.JobPropertyProviderMirrorEnabled: strconv.FormatBool(providerMirrorSetting.Value),
		},
	}
	if err := job.SetStatus(models.JobQueued); err != nil {
		return nil, errors.Wrap(err, "failed to set job status")
	}

	createdJob, err := t.dbClient.Jobs.CreateJob(ctx, job)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create job")
	}

	if _, err := t.dbClient.LogStreams.CreateLogStream(ctx, &models.LogStream{
		JobID: &createdJob.Metadata.ID,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to create log stream")
	}

	return createdJob, nil
}
