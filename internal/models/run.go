package models

import "time"

// RunStatus represents the various states for a Run resource
type RunStatus string

// Run Status Types
const (
	RunApplied            RunStatus = "applied"
	RunApplyQueued        RunStatus = "apply_queued"
	RunApplying           RunStatus = "applying"
	RunCanceled           RunStatus = "canceled"
	RunErrored            RunStatus = "errored"
	RunPending            RunStatus = "pending"
	RunPlanQueued         RunStatus = "plan_queued"
	RunPlanned            RunStatus = "planned"
	RunPlannedAndFinished RunStatus = "planned_and_finished"
	RunPlanning           RunStatus = "planning"
)

// Run represents a terraform run
// Only one of ConfigurationVersionID, ModuleSource/ModuleVersion can be non-nil.
// The ModuleVersion field is optional: blank if non-registry or want latest version
type Run struct {
	ConfigurationVersionID *string
	ForceCancelAvailableAt *time.Time
	ForceCanceledBy        *string
	ModuleVersion          *string
	ModuleSource           *string
	TargetAddresses        []string
	ModuleDigest           []byte // This is only set for modules stored in the Tharsis module registry
	CreatedBy              string
	PlanID                 string
	ApplyID                string
	WorkspaceID            string
	Status                 RunStatus
	Comment                string
	TerraformVersion       string
	Metadata               ResourceMetadata
	HasChanges             bool
	IsDestroy              bool
	IsAssessmentRun        bool
	ForceCanceled          bool
	AutoApply              bool
	Refresh                bool
	RefreshOnly            bool
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Run) ResolveMetadata(key string) (string, error) {
	return r.Metadata.resolveFieldValue(key)
}

// Speculative returns whether this run is speculative.
func (r *Run) Speculative() bool {
	return r.ApplyID == ""
}

// IsComplete returns true if the run is in a completed state
func (r *Run) IsComplete() bool {
	switch r.Status {
	case RunApplied, RunCanceled, RunErrored, RunPlannedAndFinished:
		return true
	default:
		return false
	}
}
