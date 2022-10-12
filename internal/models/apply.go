package models

// ApplyStatus represents the various states for a Apply resource
type ApplyStatus string

// Apply Status Types
const (
	ApplyCanceled ApplyStatus = "canceled"
	ApplyCreated  ApplyStatus = "created"
	ApplyErrored  ApplyStatus = "errored"
	ApplyFinished ApplyStatus = "finished"
	ApplyPending  ApplyStatus = "pending"
	ApplyQueued   ApplyStatus = "queued"
	ApplyRunning  ApplyStatus = "running"
)

// Apply includes information related to running a terraform plan command
type Apply struct {
	WorkspaceID string
	Status      ApplyStatus
	TriggeredBy string
	Comment     string
	Metadata    ResourceMetadata
}
