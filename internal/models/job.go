package models

import (
	"fmt"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*Job)(nil)

// JobStatus type
type JobStatus string

// Job Status Constants
const (
	JobQueued    JobStatus = "queued"
	JobPending   JobStatus = "pending"
	JobRunning   JobStatus = "running"
	JobFailed    JobStatus = "failed"
	JobCanceled  JobStatus = "canceled"
	JobCanceling JobStatus = "canceling"
	JobFinished  JobStatus = "finished"
)

// IsFinal returns true if this is a final status for the job
func (s JobStatus) IsFinal() bool {
	switch s {
	case JobFailed, JobCanceled, JobFinished:
		return true
	}
	return false
}

// jobStatusTransitions lists the statuses each job status may legally transition to.
// Final statuses are absent (no outgoing transitions).
var jobStatusTransitions = map[JobStatus][]JobStatus{
	JobQueued:    {JobPending, JobCanceled},
	JobPending:   {JobRunning, JobCanceled},
	JobRunning:   {JobFinished, JobFailed, JobCanceled, JobCanceling},
	JobCanceling: {JobCanceled, JobFinished, JobFailed},
}

// canTransitionTo reports whether the job may move from its current status to next.
func (s JobStatus) canTransitionTo(next JobStatus) bool {
	for _, allowed := range jobStatusTransitions[s] {
		if allowed == next {
			return true
		}
	}
	return false
}

// JobType indicates the type of job
type JobType string

// Job Types Constants
const (
	JobPlanType  JobType = "plan"
	JobApplyType JobType = "apply"
)

// Job Property Keys
const (
	JobPropertyProviderMirrorEnabled = "providerMirrorEnabled"
)

// CurrentJobProtocolVersion is the current version of the job protocol.
// This should only be incremented when the protocol between the job executor and the API changes.
const CurrentJobProtocolVersion = "1.0.0"

// JobTimestamps includes the timestamp for each job state change
type JobTimestamps struct {
	QueuedTimestamp   *time.Time
	PendingTimestamp  *time.Time
	RunningTimestamp  *time.Time
	FinishedTimestamp *time.Time
}

// Job represents a unit of work that needs to be completed
type Job struct {
	Timestamps                 JobTimestamps
	CancelRequestedTimestamp   *time.Time
	status                     JobStatus
	Type                       JobType
	WorkspaceID                string
	RunID                      string
	RunnerID                   *string
	RunnerPath                 *string
	Metadata                   ResourceMetadata
	MaxJobDuration             int32
	ForceCanceled              bool
	OutdatedJobProtocolVersion bool
	Tags                       []string
	Properties                 map[string]string
}

// GetStatus returns the job's current status.
func (j *Job) GetStatus() JobStatus {
	return j.status
}

// SetStatus transitions the job to a new status. The initial assignment from the
// zero value — constructing a new job or hydrating one from the database — is always
// allowed; any later change must be a valid job lifecycle transition.
func (j *Job) SetStatus(status JobStatus) error {
	if j.status != "" && !j.status.canTransitionTo(status) {
		return fmt.Errorf("invalid job status transition from %q to %q", j.status, status)
	}
	j.status = status
	return nil
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
func (j *Job) ResolveMetadata(key string) (*string, error) {
	return j.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (j *Job) Validate() error {
	return nil
}
