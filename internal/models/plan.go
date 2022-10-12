package models

// PlanStatus represents the various states for a Plan resource
type PlanStatus string

// Run Status Types
const (
	PlanCanceled PlanStatus = "canceled"
	PlanQueued   PlanStatus = "queued"
	PlanErrored  PlanStatus = "errored"
	PlanFinished PlanStatus = "finished"
	PlanPending  PlanStatus = "pending"
	PlanRunning  PlanStatus = "running"
)

// Plan includes information related to running a terraform plan command
type Plan struct {
	WorkspaceID          string
	Status               PlanStatus
	Metadata             ResourceMetadata
	ResourceAdditions    int
	ResourceChanges      int
	ResourceDestructions int
	HasChanges           bool
}
