package models

import (
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

var _ Model = (*Run)(nil)

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

// GetID returns the Metadata ID.
func (r *Run) GetID() string {
	return r.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (r *Run) GetGlobalID() string {
	return gid.ToGlobalID(r.GetModelType(), r.Metadata.ID)
}

// GetModelType returns the type of the model.
func (r *Run) GetModelType() types.ModelType {
	return types.RunModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination
func (r *Run) ResolveMetadata(key string) (*string, error) {
	return r.Metadata.resolveFieldValue(key)
}

// Validate validates the model.
func (r *Run) Validate() error {
	return nil
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

// GetGroupPath returns the group path
func (r *Run) GetGroupPath() string {
	path := strings.Split(r.Metadata.TRN[len(types.TRNPrefix):], ":")[1]
	pathSegments := strings.Split(path, "/")
	groupPath := strings.Join(pathSegments[:len(pathSegments)-2], "/")
	return groupPath
}
