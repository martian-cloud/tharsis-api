package models

import (
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*Job)(nil)

// JobStatus type
type JobStatus string

// Job Status Constants
const (
	JobQueued   JobStatus = "queued"
	JobPending  JobStatus = "pending"
	JobRunning  JobStatus = "running"
	JobFinished JobStatus = "finished"
)

// JobType indicates the type of job
type JobType string

// Job Types Constants
const (
	JobPlanType  JobType = "plan"
	JobApplyType JobType = "apply"
)

// JobTimestamps includes the timestamp for each job state change
type JobTimestamps struct {
	QueuedTimestamp   *time.Time
	PendingTimestamp  *time.Time
	RunningTimestamp  *time.Time
	FinishedTimestamp *time.Time
}

// Job represents a unit of work that needs to be completed
type Job struct {
	Timestamps               JobTimestamps
	CancelRequestedTimestamp *time.Time
	Status                   JobStatus
	Type                     JobType
	WorkspaceID              string
	RunID                    string
	RunnerID                 *string
	RunnerPath               *string
	Metadata                 ResourceMetadata
	MaxJobDuration           int32
	CancelRequested          bool
	Tags                     []string
}

// GetID returns the Metadata ID.
func (j *Job) GetID() string {
	return j.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (j *Job) GetGlobalID() string {
	return gid.ToGlobalID(j.GetModelType(), j.Metadata.ID)
}

// GetModelType returns the Model's type.
func (j *Job) GetModelType() types.ModelType {
	return types.JobModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (j *Job) ResolveMetadata(key string) (string, error) {
	return j.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (j *Job) Validate() error {
	return nil
}
