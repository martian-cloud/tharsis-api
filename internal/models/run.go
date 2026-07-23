package models

import (
	"slices"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// RunStatus represents the overall status of a run.
type RunStatus string

// RunStatus constants. The queuing/queuing_apply statuses mirror the TFE (go-tfe)
// run statuses of the same names: queuing means the plan node is pending (waiting
// to be queued/admitted), and queuing_apply means the apply node is pending.
const (
	RunApplied            RunStatus = "applied"
	RunApplyQueued        RunStatus = "apply_queued"
	RunApplying           RunStatus = "applying"
	RunCanceled           RunStatus = "canceled"
	RunDiscarded          RunStatus = "discarded"
	RunErrored            RunStatus = "errored"
	RunPending            RunStatus = "pending"
	RunPlanQueued         RunStatus = "plan_queued"
	RunPlanned            RunStatus = "planned"
	RunPlannedAndFinished RunStatus = "planned_and_finished"
	RunPlanning           RunStatus = "planning"
	RunQueuing            RunStatus = "queuing"
	RunQueuingApply       RunStatus = "queuing_apply"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s RunStatus) IsFinalStatus() bool {
	return s == RunApplied || s == RunPlannedAndFinished || s == RunErrored || s == RunCanceled || s == RunDiscarded
}

// PlanStatus represents the status of a plan node.
type PlanStatus string

// PlanStatus constants. The lifecycle is:
// created -> pending (ready, awaiting workspace admission) -> queued (job created)
// -> running -> finished/errored/canceled.
const (
	PlanCreated  PlanStatus = "created"
	PlanPending  PlanStatus = "pending"
	PlanQueued   PlanStatus = "queued"
	PlanRunning  PlanStatus = "running"
	PlanFinished PlanStatus = "finished"
	PlanErrored  PlanStatus = "errored"
	PlanCanceled PlanStatus = "canceled"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s PlanStatus) IsFinalStatus() bool {
	return s == PlanFinished || s == PlanErrored || s == PlanCanceled
}

// ApplyStatus represents the status of an apply node.
type ApplyStatus string

// ApplyStatus constants. The lifecycle mirrors the plan:
// created -> pending (approved, awaiting workspace admission) -> queued (job created)
// -> running -> finished/errored/canceled. An apply that never starts before the run
// reaches a final state (plan errored/canceled, plan finished without changes, run
// discarded) moves from created to skipped instead.
const (
	ApplyCreated  ApplyStatus = "created"
	ApplyPending  ApplyStatus = "pending"
	ApplyQueued   ApplyStatus = "queued"
	ApplyRunning  ApplyStatus = "running"
	ApplyFinished ApplyStatus = "finished"
	ApplyErrored  ApplyStatus = "errored"
	ApplyCanceled ApplyStatus = "canceled"
	ApplySkipped  ApplyStatus = "skipped"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s ApplyStatus) IsFinalStatus() bool {
	return s == ApplyFinished || s == ApplyErrored || s == ApplyCanceled || s == ApplySkipped
}

var _ Model = (*Run)(nil)

// PlanSummary contains a summary of the types of changes this plan includes
type PlanSummary struct {
	ResourceAdditions    int32
	ResourceChanges      int32
	ResourceDestructions int32
	ResourceImports      int32
	ResourceDrift        int32
	OutputAdditions      int32
	OutputChanges        int32
	OutputDestructions   int32
}

// Run-relative node paths identifying the plan and apply nodes within a run.
// Used by RetryNode and the node GetPath implementations.
const (
	PlanNodePath  = "plan"
	ApplyNodePath = "apply"
)

// RunNode is the interface for typed run nodes (plan, apply)
type RunNode interface {
	GetID() string
	GetPath() string
	Copy() RunNode
	ShallowCompare(other RunNode) bool
}

// Plan represents a plan task within a run
type Plan struct {
	ErrorMessage        *string
	LatestJobID         *string
	ID                  string
	CacheObjectStoreKey *string
	JSONObjectStoreKey  *string
	DiffObjectStoreKey  *string
	Status              PlanStatus
	DiffSize            int
	Summary             PlanSummary
	HasChanges          bool
}

// GetID returns the node ID
func (n *Plan) GetID() string { return n.ID }

// GetPath returns the run-relative path identifying this node within its run.
func (n *Plan) GetPath() string { return PlanNodePath }

// GetGlobalID returns the ID as a GID.
func (n *Plan) GetGlobalID() string {
	return gid.ToGlobalID(types.PlanModelType, n.ID)
}

// Metadata returns the resource metadata for this plan node derived from the parent run.
func (n *Plan) Metadata(run *Run) *ResourceMetadata {
	return &ResourceMetadata{
		ID:                   n.ID,
		TRN:                  trn.TypePlan.Build(run.GetWorkspacePath(), run.GetGlobalID(), "plan"),
		Version:              run.Metadata.Version,
		CreationTimestamp:    run.Metadata.CreationTimestamp,
		LastUpdatedTimestamp: run.Metadata.LastUpdatedTimestamp,
	}
}

// Apply represents an apply task within a run
type Apply struct {
	ErrorMessage *string
	LatestJobID  *string
	ID           string
	Status       ApplyStatus
	TriggeredBy  string
	Comment      string
}

// GetID returns the node ID
func (n *Apply) GetID() string { return n.ID }

// GetPath returns the run-relative path identifying this node within its run.
func (n *Apply) GetPath() string { return ApplyNodePath }

// Metadata returns the resource metadata for this apply node derived from the parent run.
func (n *Apply) Metadata(run *Run) *ResourceMetadata {
	return &ResourceMetadata{
		ID:                   n.ID,
		TRN:                  trn.TypeApply.Build(run.GetWorkspacePath(), run.GetGlobalID(), "apply"),
		Version:              run.Metadata.Version,
		CreationTimestamp:    run.Metadata.CreationTimestamp,
		LastUpdatedTimestamp: run.Metadata.LastUpdatedTimestamp,
	}
}

// GetGlobalID returns the ID as a GID.
func (n *Apply) GetGlobalID() string {
	return gid.ToGlobalID(types.ApplyModelType, n.ID)
}

// Run represents a terraform run
// Only one of ConfigurationVersionID, ModuleSource/ModuleVersion can be non-nil.
// The ModuleVersion field is optional: blank if non-registry or want latest version
type Run struct {
	ConfigurationVersionID  *string
	ForceCancelAvailableAt  *time.Time
	ForceCanceledBy         *string
	ModuleVersion           *string
	ModuleSource            *string
	TargetAddresses         []string
	Plan                    Plan
	Apply                   *Apply
	ModuleDigest            []byte // This is only set for modules stored in the Tharsis module registry
	CreatedBy               string
	WorkspaceID             string
	VariablesObjectStoreKey *string
	Status                  RunStatus
	Comment                 string
	TerraformVersion        string
	Metadata                ResourceMetadata
	IsDestroy               bool
	IsAssessmentRun         bool
	ForceCanceled           bool
	AutoApply               bool
	Refresh                 bool
	RefreshOnly             bool
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
	return r.Apply == nil
}

// HasChanges returns whether the run's plan produced changes. It is derived from the
// plan node rather than stored, so it can never drift out of sync.
func (r *Run) HasChanges() bool {
	return r.Plan.HasChanges
}

// IsComplete returns true if the run is in a completed state
func (r *Run) IsComplete() bool {
	return r.Status.IsFinalStatus()
}

// GetGroupPath returns the group path
func (r *Run) GetGroupPath() string {
	parts := trn.MustParseAny(r.Metadata.TRN).PathParts()
	return strings.Join(parts[:len(parts)-2], "/")
}

// GetWorkspacePath returns the workspace path
func (r *Run) GetWorkspacePath() string {
	parts := trn.MustParseAny(r.Metadata.TRN).PathParts()
	return strings.Join(parts[:len(parts)-1], "/")
}

// Copy creates a deep copy of the Run.
func (r *Run) Copy() *Run {
	cp := &Run{
		ConfigurationVersionID: r.ConfigurationVersionID,
		ForceCancelAvailableAt: r.ForceCancelAvailableAt,
		ForceCanceledBy:        r.ForceCanceledBy,
		ModuleVersion:          r.ModuleVersion,
		ModuleSource:           r.ModuleSource,
		TargetAddresses:        slices.Clone(r.TargetAddresses),
		Plan:                   *r.Plan.Copy().(*Plan),
		ModuleDigest:           slices.Clone(r.ModuleDigest),
		CreatedBy:              r.CreatedBy,
		WorkspaceID:            r.WorkspaceID,
		Status:                 r.Status,
		Comment:                r.Comment,
		TerraformVersion:       r.TerraformVersion,
		Metadata:               r.Metadata,
		IsDestroy:              r.IsDestroy,
		IsAssessmentRun:        r.IsAssessmentRun,
		ForceCanceled:          r.ForceCanceled,
		AutoApply:              r.AutoApply,
		Refresh:                r.Refresh,
		RefreshOnly:            r.RefreshOnly,
	}
	if r.Apply != nil {
		applyCopy := r.Apply.Copy().(*Apply)
		cp.Apply = applyCopy
	}
	return cp
}

// Diff compares this run with another and returns the IDs of nodes that have changed.
// The run's own Metadata.ID is included when run-level fields change. Plan and apply
// nodes are identified by their own IDs.
func (r *Run) Diff(other *Run) []string {
	var changedIDs []string

	if !r.ShallowCompare(other) {
		changedIDs = append(changedIDs, r.Metadata.ID)
	}

	if !r.Plan.ShallowCompare(&other.Plan) {
		changedIDs = append(changedIDs, r.Plan.ID)
	}

	switch {
	case r.Apply != nil && other.Apply != nil:
		if !r.Apply.ShallowCompare(other.Apply) {
			changedIDs = append(changedIDs, r.Apply.ID)
		}
	case r.Apply != nil:
		// Apply node exists now but not in the comparison target: its presence
		// changed, so flag it.
		changedIDs = append(changedIDs, r.Apply.ID)
	}

	return changedIDs
}

// ShallowCompare compares all run-level content fields so Diff detects any change, even to
// fields that are immutable today. It deliberately excludes:
//   - the Plan and Apply nodes: Diff compares those separately and attributes a change to the
//     node's own ID, so comparing them here would wrongly flag the run row too;
//   - Metadata: resource identity plus the DB-managed optimistic-lock version and timestamps,
//     which are not run content.
func (r *Run) ShallowCompare(other *Run) bool {
	if r == nil && other == nil {
		return true
	}
	if r == nil || other == nil {
		return false
	}
	return r.Status == other.Status &&
		r.CreatedBy == other.CreatedBy &&
		r.WorkspaceID == other.WorkspaceID &&
		r.Comment == other.Comment &&
		r.TerraformVersion == other.TerraformVersion &&
		r.IsDestroy == other.IsDestroy &&
		r.IsAssessmentRun == other.IsAssessmentRun &&
		r.ForceCanceled == other.ForceCanceled &&
		r.AutoApply == other.AutoApply &&
		r.Refresh == other.Refresh &&
		r.RefreshOnly == other.RefreshOnly &&
		ptrStringEqual(r.ConfigurationVersionID, other.ConfigurationVersionID) &&
		ptrStringEqual(r.ModuleSource, other.ModuleSource) &&
		ptrStringEqual(r.ModuleVersion, other.ModuleVersion) &&
		ptrStringEqual(r.ForceCanceledBy, other.ForceCanceledBy) &&
		ptrTimeEqual(r.ForceCancelAvailableAt, other.ForceCancelAvailableAt) &&
		slices.Equal(r.TargetAddresses, other.TargetAddresses) &&
		slices.Equal(r.ModuleDigest, other.ModuleDigest) &&
		ptrStringEqual(r.VariablesObjectStoreKey, other.VariablesObjectStoreKey)
}

// Copy creates a deep copy of the Plan.
func (n *Plan) Copy() RunNode {
	return &Plan{
		ErrorMessage:        n.ErrorMessage,
		LatestJobID:         n.LatestJobID,
		ID:                  n.ID,
		Status:              n.Status,
		DiffSize:            n.DiffSize,
		Summary:             n.Summary,
		HasChanges:          n.HasChanges,
		CacheObjectStoreKey: n.CacheObjectStoreKey,
		JSONObjectStoreKey:  n.JSONObjectStoreKey,
		DiffObjectStoreKey:  n.DiffObjectStoreKey,
	}
}

// ShallowCompare compares this Plan with another RunNode.
func (n *Plan) ShallowCompare(other RunNode) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	o, ok := other.(*Plan)
	if !ok {
		return false
	}
	return n.Status == o.Status &&
		n.HasChanges == o.HasChanges &&
		n.DiffSize == o.DiffSize &&
		n.Summary == o.Summary &&
		ptrStringEqual(n.CacheObjectStoreKey, o.CacheObjectStoreKey) &&
		ptrStringEqual(n.JSONObjectStoreKey, o.JSONObjectStoreKey) &&
		ptrStringEqual(n.DiffObjectStoreKey, o.DiffObjectStoreKey) &&
		ptrStringEqual(n.LatestJobID, o.LatestJobID) &&
		ptrStringEqual(n.ErrorMessage, o.ErrorMessage)
}

// Copy creates a deep copy of the Apply.
func (n *Apply) Copy() RunNode {
	return &Apply{
		ErrorMessage: n.ErrorMessage,
		LatestJobID:  n.LatestJobID,
		ID:           n.ID,
		Status:       n.Status,
		TriggeredBy:  n.TriggeredBy,
		Comment:      n.Comment,
	}
}

// ShallowCompare compares this Apply with another RunNode.
func (n *Apply) ShallowCompare(other RunNode) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	o, ok := other.(*Apply)
	if !ok {
		return false
	}
	return n.Status == o.Status &&
		n.TriggeredBy == o.TriggeredBy &&
		n.Comment == o.Comment &&
		ptrStringEqual(n.LatestJobID, o.LatestJobID) &&
		ptrStringEqual(n.ErrorMessage, o.ErrorMessage)
}

func ptrStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrTimeEqual(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
