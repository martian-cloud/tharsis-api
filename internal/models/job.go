package models

import "time"

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
	RunnerID                 string
	Metadata                 ResourceMetadata
	MaxJobDuration           int32
	CancelRequested          bool
}

// JobLogDescriptor contains metadata for job logs
type JobLogDescriptor struct {
	JobID    string
	Metadata ResourceMetadata
	Size     int
}
