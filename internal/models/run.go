package models

import (
	"slices"
	"strings"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/trn"
)

// runNodeGIDCode is the GID code for run-node id handles (plan, apply, policy check). Run nodes are
// not first-class models; this code exists only to encode/decode their ids as GIDs.
const runNodeGIDCode = "RN"

// RunNodeGID encodes a run node's raw node id as a RunNode GID handle.
func RunNodeGID(nodeID string) string {
	return gid.NewGlobalIDWithCode(runNodeGIDCode, nodeID).String()
}

// RunStatus represents the overall status of a run.
type RunStatus string

// RunStatus constants, listed in run execution order followed by the terminal statuses.
//
// Every workspace-gated node contributes two statuses, because the two waits it can be in are
// different things and the run status names both: *_queuing means the node is ready but has not been
// admitted to the workspace yet, and *_queued means it is admitted and its job is waiting for a
// runner. Collapsing them would leave the run reporting plan_queued while its plan node is still
// pending, and would erase the queuing -> queued transition, which is the moment the run won the
// workspace slot.
const (
	RunPending RunStatus = "pending"
	// The pre-plan policy stage. Gated on the workspace: the stage waits at pre_plan_queuing.
	RunPrePlanQueuing          RunStatus = "pre_plan_queuing"
	RunPrePlanRunning          RunStatus = "pre_plan_running"
	RunPrePlanAwaitingDecision RunStatus = "pre_plan_awaiting_decision"
	RunPrePlanCompleted        RunStatus = "pre_plan_completed"
	// The plan. Gated on the workspace, then on a runner.
	RunPlanQueuing RunStatus = "plan_queuing"
	RunPlanQueued  RunStatus = "plan_queued"
	RunPlanning    RunStatus = "planning"
	// The post-plan policy stage. Not separately gated: it inherits the plan's workspace slot.
	RunPostPlanRunning          RunStatus = "post_plan_running"
	RunPostPlanAwaitingDecision RunStatus = "post_plan_awaiting_decision"
	RunPostPlanCompleted        RunStatus = "post_plan_completed"
	RunPlanned                  RunStatus = "planned"
	// The pre-apply policy stage. Gated on the workspace, like the pre-plan stage.
	RunPreApplyQueuing          RunStatus = "pre_apply_queuing"
	RunPreApplyRunning          RunStatus = "pre_apply_running"
	RunPreApplyAwaitingDecision RunStatus = "pre_apply_awaiting_decision"
	RunPreApplyCompleted        RunStatus = "pre_apply_completed"
	// The apply. Gated on the workspace, then on a runner.
	RunApplyQueuing       RunStatus = "apply_queuing"
	RunApplyQueued        RunStatus = "apply_queued"
	RunApplying           RunStatus = "applying"
	RunPostApplyRunning   RunStatus = "post_apply_running"
	RunPostApplyCompleted RunStatus = "post_apply_completed"
	// Terminal statuses.
	RunApplied            RunStatus = "applied"
	RunPlannedAndFinished RunStatus = "planned_and_finished"
	RunCanceled           RunStatus = "canceled"
	RunDiscarded          RunStatus = "discarded"
	RunErrored            RunStatus = "errored"
)

// AllRunStatuses is every run status, in the order declared above. It exists so exhaustiveness can be
// asserted in tests — in particular that every status has an explicit mapping onto the TFE run status
// the Terraform CLI understands, rather than leaking a raw Tharsis string to the CLI.
var AllRunStatuses = []RunStatus{
	RunPending,
	RunPrePlanQueuing, RunPrePlanRunning, RunPrePlanAwaitingDecision, RunPrePlanCompleted,
	RunPlanQueuing, RunPlanQueued, RunPlanning,
	RunPostPlanRunning, RunPostPlanAwaitingDecision, RunPostPlanCompleted,
	RunPlanned,
	RunPreApplyQueuing, RunPreApplyRunning, RunPreApplyAwaitingDecision, RunPreApplyCompleted,
	RunApplyQueuing, RunApplyQueued, RunApplying,
	RunPostApplyRunning, RunPostApplyCompleted,
	RunApplied, RunPlannedAndFinished, RunCanceled, RunDiscarded, RunErrored,
}

// QueuingRunStatuses are the statuses a run holds while it is waiting to be admitted to its workspace,
// in run order — one per workspace-gated node. This is the single Go-side definition of "waiting for the
// workspace slot": the admitter, the work item consumer, the reconciler and the TFE run queue all read
// it, so adding a gated node means adding its status here and nowhere else.
//
// The partial indexes that make the admission queries cheap must list the same statuses in SQL, since a
// predicate cannot call into Go. They are created in the add_policy_enforcement migration and named
// index_runs_on_*_queuing; a new gated node needs a migration to extend them.
var QueuingRunStatuses = []RunStatus{
	RunPrePlanQueuing,
	RunPlanQueuing,
	RunPreApplyQueuing,
	RunApplyQueuing,
}

// IsQueuing reports whether a run in this status is waiting for the workspace slot, as opposed to
// waiting for a runner (the *_queued statuses) or running.
func (s RunStatus) IsQueuing() bool {
	return slices.Contains(QueuingRunStatuses, s)
}

// IsFinalStatus returns true if the status is a terminal state.
func (s RunStatus) IsFinalStatus() bool {
	return s == RunApplied || s == RunPlannedAndFinished || s == RunErrored || s == RunCanceled || s == RunDiscarded
}

// In reports whether s is one of statuses. An empty statuses means "no constraint" and always matches.
func (s RunStatus) In(statuses []RunStatus) bool {
	return len(statuses) == 0 || slices.Contains(statuses, s)
}

// PlanStatus represents the status of a plan node.
type PlanStatus string

// PlanStatus constants. The lifecycle is:
// created -> pending (ready, awaiting workspace admission) -> queued (job created)
// -> running -> finished/errored/canceled. A plan that never starts before the run reaches a final
// state (a pre-plan policy gate errored, the run was canceled, or it was discarded while awaiting a
// pre-plan override) moves from created to skipped instead, mirroring the apply.
const (
	PlanCreated  PlanStatus = "created"
	PlanPending  PlanStatus = "pending"
	PlanQueued   PlanStatus = "queued"
	PlanRunning  PlanStatus = "running"
	PlanFinished PlanStatus = "finished"
	PlanErrored  PlanStatus = "errored"
	PlanCanceled PlanStatus = "canceled"
	PlanSkipped  PlanStatus = "skipped"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s PlanStatus) IsFinalStatus() bool {
	return s == PlanFinished || s == PlanErrored || s == PlanCanceled || s == PlanSkipped
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

// RunTaskStageName identifies which stage of a run a node (policy check, future task, etc.) evaluates at.
type RunTaskStageName string

// RunTaskStageName constants, in run execution order.
const (
	RunTaskStageNamePrePlan   RunTaskStageName = "pre_plan"
	RunTaskStageNamePostPlan  RunTaskStageName = "post_plan"
	RunTaskStageNamePreApply  RunTaskStageName = "pre_apply"
	RunTaskStageNamePostApply RunTaskStageName = "post_apply"
)

// WorkspaceGatedStages are the stages that must acquire the workspace slot before they start, in run
// order. They evaluate against state another run could change underneath them: the pre-plan stage
// gates the plan, and the pre-apply stage evaluates the plan it is about to apply. The post-plan stage
// is not listed — it inherits the slot the plan already holds — and the post-apply stage runs after
// state is written, so it gates nothing.
var WorkspaceGatedStages = []RunTaskStageName{RunTaskStageNamePrePlan, RunTaskStageNamePreApply}

// IsWorkspaceGated reports whether a stage must be admitted to the workspace before it starts. A gated
// stage waits at RunTaskStagePending until the admitter acquires the slot; an ungated one goes straight
// from created to running on the slot the plan already holds.
func (s RunTaskStageName) IsWorkspaceGated() bool {
	for _, gated := range WorkspaceGatedStages {
		if s == gated {
			return true
		}
	}
	return false
}

// IsValid reports whether the stage name is one of the recognized run stages.
func (s RunTaskStageName) IsValid() bool {
	switch s {
	case RunTaskStageNamePrePlan, RunTaskStageNamePostPlan, RunTaskStageNamePreApply, RunTaskStageNamePostApply:
		return true
	default:
		return false
	}
}

// RunTaskStageStatus represents the status of a task stage node. A task stage aggregates the verdict
// of its child policy checks (and, in future, run task results): it is pending while it waits for the
// workspace slot, running while any child is evaluating, awaiting_override when a child is blocked on
// a human override, completed once every child has cleared, errored when a child failed a hard gate,
// canceled when the run is canceled or discarded, and skipped when the stage never ran (e.g. a
// post-plan stage on a no-change plan). awaiting_override mirrors go-tfe's TaskStageAwaitingOverride.
//
// pending is named to match PlanPending/ApplyPending: on every node it means "ready, but not yet
// admitted to the workspace". A stage has no queued status because the job-created state lives one
// level down, on its child policy checks (PolicyCheckQueued).
type RunTaskStageStatus string

// RunTaskStageStatus constants.
const (
	RunTaskStageCreated          RunTaskStageStatus = "created"
	RunTaskStagePending          RunTaskStageStatus = "pending"
	RunTaskStageRunning          RunTaskStageStatus = "running"
	RunTaskStageAwaitingOverride RunTaskStageStatus = "awaiting_override"
	RunTaskStageCompleted        RunTaskStageStatus = "completed"
	RunTaskStageErrored          RunTaskStageStatus = "errored"
	RunTaskStageCanceled         RunTaskStageStatus = "canceled"
	RunTaskStageSkipped          RunTaskStageStatus = "skipped"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s RunTaskStageStatus) IsFinalStatus() bool {
	return s == RunTaskStageCompleted || s == RunTaskStageErrored ||
		s == RunTaskStageCanceled || s == RunTaskStageSkipped
}

// PolicyCheckStatus represents the status of a policy check node. The lifecycle is:
// created -> queued -> running -> passed | soft_failed | errored | canceled, plus
// soft_failed -> overridden. The check passes through queued (its policy-eval job
// has been created and is awaiting a runner) before running. "soft_failed" is a
// check-node status (it blocks the run at post_plan_awaiting_decision), matching go-tfe's
// TaskStageAwaitingOverride.
//
// pending means "ready to run, but its stage is not running yet", and is how a retry re-enters the
// lifecycle: a retried check goes errored/canceled/soft_failed -> pending, and its stage reacts by
// restarting itself (waiting for the workspace slot first if the stage is gated). It matches
// PlanPending/ApplyPending/RunTaskStagePending — on every node, pending is the pre-admission state.
// A fresh check starts at created instead, because it has no stage to restart: the run reaches its
// stage in the normal forward flow. A soft_failed check is retryable so that a policy which has since
// been fixed can be re-evaluated rather than overridden; retrying discards the check's gate and any
// approvals collected against the previous verdict.
type PolicyCheckStatus string

// PolicyCheckStatus constants.
const (
	PolicyCheckCreated    PolicyCheckStatus = "created"
	PolicyCheckPending    PolicyCheckStatus = "pending"
	PolicyCheckQueued     PolicyCheckStatus = "queued"
	PolicyCheckRunning    PolicyCheckStatus = "running"
	PolicyCheckPassed     PolicyCheckStatus = "passed"
	PolicyCheckSoftFailed PolicyCheckStatus = "soft_failed"
	PolicyCheckOverridden PolicyCheckStatus = "overridden"
	PolicyCheckErrored    PolicyCheckStatus = "errored"
	PolicyCheckCanceled   PolicyCheckStatus = "canceled"
	PolicyCheckSkipped    PolicyCheckStatus = "skipped"
)

// IsFinalStatus returns true if the status is a terminal state.
func (s PolicyCheckStatus) IsFinalStatus() bool {
	return s == PolicyCheckPassed || s == PolicyCheckOverridden ||
		s == PolicyCheckErrored || s == PolicyCheckCanceled || s == PolicyCheckSkipped
}

// NotStarted returns true if the policy check has not begun evaluation yet.
func (s PolicyCheckStatus) NotStarted() bool {
	return s == PolicyCheckCreated || s == PolicyCheckPending
}

// PolicyCheckPolicyStatus is the result the policy evaluator reported for a policy set within a
// policy check. It starts as pending and transitions to passed or failed when the evaluator reports.
type PolicyCheckPolicyStatus string

// PolicyCheckPolicyStatus constants.
const (
	PolicyCheckPolicyPending PolicyCheckPolicyStatus = "pending"
	PolicyCheckPolicyPassed  PolicyCheckPolicyStatus = "passed"
	PolicyCheckPolicyFailed  PolicyCheckPolicyStatus = "failed"
)

// PolicyCheckPolicyProvenance records the group that owns a policy attachment and the stable TRN
// of the attachment itself. Policies are always group-scoped.
type PolicyCheckPolicyProvenance struct {
	GroupID   string `json:"groupId"`
	PolicyTRN string `json:"policyTrn"`
}

// PolicyCheckOPAData is the OPA-specific snapshot of a policy a check evaluates.
// PackageVersionConstraint is snapshotted unresolved (empty means the latest uploaded version): the
// policy-eval job resolves it to a concrete package version when the check runs.
type PolicyCheckOPAData struct {
	PackageSource            string  `json:"packageSource"`
	PackageVersionConstraint string  `json:"packageVersionConstraint"`
	PackageDigest            *string `json:"packageDigest,omitempty"`
}

// Copy creates a deep copy of the PolicyCheckOPAData. Nil copies to nil.
func (d *PolicyCheckOPAData) Copy() *PolicyCheckOPAData {
	if d == nil {
		return nil
	}
	return &PolicyCheckOPAData{
		PackageSource:            d.PackageSource,
		PackageVersionConstraint: d.PackageVersionConstraint,
		PackageDigest:            d.PackageDigest,
	}
}

func policyCheckOPADataEqual(a, b *PolicyCheckOPAData) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.PackageSource == b.PackageSource &&
		a.PackageVersionConstraint == b.PackageVersionConstraint &&
		ptrStringEqual(a.PackageDigest, b.PackageDigest)
}

// PolicyCheckModuleAttestationData is the module-attestation snapshot of a policy a check evaluates.
type PolicyCheckModuleAttestationData struct {
	PublicKey          string  `json:"publicKey"`
	PredicateType      *string `json:"predicateType,omitempty"`
	VerifyStateLineage bool    `json:"verifyStateLineage,omitempty"`
}

// Copy creates a deep copy of the PolicyCheckModuleAttestationData. Nil copies to nil.
func (d *PolicyCheckModuleAttestationData) Copy() *PolicyCheckModuleAttestationData {
	if d == nil {
		return nil
	}
	return &PolicyCheckModuleAttestationData{
		PublicKey:          d.PublicKey,
		PredicateType:      d.PredicateType,
		VerifyStateLineage: d.VerifyStateLineage,
	}
}

func policyCheckModuleAttestationDataEqual(a, b *PolicyCheckModuleAttestationData) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.PublicKey == b.PublicKey &&
		ptrStringEqual(a.PredicateType, b.PredicateType) &&
		a.VerifyStateLineage == b.VerifyStateLineage
}

// PolicyCheckPolicy is the snapshot of a single policy a policy check evaluates, together
// with its result. There is one entry per valid policy (no de-duplication), each with its own
// stable ID, EnforcementLevel, kind-specific data, and — for soft-mandatory policies — the approver
// snapshot (RequiredApprovals + allowed principal ids) used to gate an override.
// Name and Description are copied from the policy at run creation so the run keeps reporting the
// policy as it was evaluated, even after the policy is renamed, re-described, or deleted.
// Exactly one of OPAData and ModuleAttestationData is set, determined by the owning check's
// CheckType. Status is empty and MessagesObjectStoreKey nil until the check reports its result.
// Entries are stored as elements of the check's Policies JSONB column, so adding a field here needs
// no migration — it is simply absent (and therefore zero) on checks created before it existed.
type PolicyCheckPolicy struct {
	ID                       string                            `json:"id"`
	Name                     string                            `json:"name,omitempty"`
	Description              string                            `json:"description,omitempty"`
	EnforcementLevel         PolicyEnforcementLevel            `json:"enforcementLevel"`
	Provenance               PolicyCheckPolicyProvenance       `json:"provenance"`
	OPAData                  *PolicyCheckOPAData               `json:"opaData,omitempty"`
	ModuleAttestationData    *PolicyCheckModuleAttestationData `json:"moduleAttestationData,omitempty"`
	RequiredApprovals        int                               `json:"requiredApprovals,omitempty"`
	AllowedUserIDs           []string                          `json:"allowedUserIds,omitempty"`
	AllowedServiceAccountIDs []string                          `json:"allowedServiceAccountIds,omitempty"`
	AllowedTeamIDs           []string                          `json:"allowedTeamIds,omitempty"`
	Status                   PolicyCheckPolicyStatus           `json:"status"`
	MessagesObjectStoreKey   *string                           `json:"messagesObjectStoreKey,omitempty"`
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

// Run-relative node paths identifying the plan and apply nodes
// within a run. Used by RetryNode and the node GetPath implementations.
const (
	PlanNodePath  = "plan"
	ApplyNodePath = "apply"
)

// RunNode is the interface for typed run nodes (plan, apply, policy check)
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

// GetGlobalID returns the plan node's ID as a RunNode GID.
func (n *Plan) GetGlobalID() string {
	return RunNodeGID(n.ID)
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

// GetGlobalID returns the apply node's ID as a RunNode GID.
func (n *Apply) GetGlobalID() string {
	return RunNodeGID(n.ID)
}

// PolicyCheckMessagesSummary is a capped copy of the violation messages the check's policies
// reported, held on the check itself so a list of checks can show what failed without a read per
// policy from object storage. Truncated says Messages is not the whole set — the full list for a
// given policy is behind that policy's MessagesObjectStoreKey.
type PolicyCheckMessagesSummary struct {
	Messages  []string `json:"messages"`
	Truncated bool     `json:"truncated"`
}

// PolicyCheck represents a policy-evaluation task within a run — one per policy engine/type
// (only OPA today) at a given stage. It is a generic run node implementing the RunNode interface
// and owns the policy-eval job, the per-policy-set policies it evaluates (with their results), and
// its verdict.
type PolicyCheck struct {
	LatestJobID *string
	ID          string
	StageName   RunTaskStageName
	CheckType   PolicyKind
	Status      PolicyCheckStatus
	Policies    []*PolicyCheckPolicy
	// MessagesSummary previews the messages the policies above reported, and is nil until the check
	// has reported (or when it reported none). It exists so a view showing many checks does not have
	// to read every policy's messages out of object storage.
	MessagesSummary *PolicyCheckMessagesSummary
}

// GetID returns the node ID.
func (n *PolicyCheck) GetID() string { return n.ID }

// GetPath returns the run-relative path identifying this node within its run. It appends the
// check type so sibling checks under a run have distinct paths.
func (n *PolicyCheck) GetPath() string { return string(n.StageName) + "." + string(n.CheckType) }

// GetGlobalID returns the policy check node's ID as a RunNode GID.
func (n *PolicyCheck) GetGlobalID() string {
	return RunNodeGID(n.ID)
}

// Copy creates a deep copy of the PolicyCheck, including its child policies.
func (n *PolicyCheck) Copy() RunNode {
	cp := &PolicyCheck{
		LatestJobID:     n.LatestJobID,
		ID:              n.ID,
		StageName:       n.StageName,
		CheckType:       n.CheckType,
		Status:          n.Status,
		MessagesSummary: n.MessagesSummary.Copy(),
	}
	if n.Policies != nil {
		cp.Policies = make([]*PolicyCheckPolicy, len(n.Policies))
		for i, p := range n.Policies {
			cp.Policies[i] = p.Copy()
		}
	}
	return cp
}

// Copy creates a deep copy of the PolicyCheckMessagesSummary, including its message slice.
// Nil copies to nil.
func (s *PolicyCheckMessagesSummary) Copy() *PolicyCheckMessagesSummary {
	if s == nil {
		return nil
	}
	return &PolicyCheckMessagesSummary{
		Messages:  slices.Clone(s.Messages),
		Truncated: s.Truncated,
	}
}

// policyCheckMessagesSummaryEqual reports whether two summaries hold the same messages. A nil
// summary (never reported) is distinct from an empty one.
func policyCheckMessagesSummaryEqual(a, b *PolicyCheckMessagesSummary) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Truncated == b.Truncated && slices.Equal(a.Messages, b.Messages)
}

// Copy creates a deep copy of the PolicyCheckPolicy, including its approver id slices and
// kind-specific data. Nil copies to nil.
func (p *PolicyCheckPolicy) Copy() *PolicyCheckPolicy {
	if p == nil {
		return nil
	}
	return &PolicyCheckPolicy{
		ID:                       p.ID,
		Name:                     p.Name,
		Description:              p.Description,
		EnforcementLevel:         p.EnforcementLevel,
		Provenance:               p.Provenance,
		OPAData:                  p.OPAData.Copy(),
		ModuleAttestationData:    p.ModuleAttestationData.Copy(),
		RequiredApprovals:        p.RequiredApprovals,
		AllowedUserIDs:           slices.Clone(p.AllowedUserIDs),
		AllowedServiceAccountIDs: slices.Clone(p.AllowedServiceAccountIDs),
		AllowedTeamIDs:           slices.Clone(p.AllowedTeamIDs),
		Status:                   p.Status,
		MessagesObjectStoreKey:   p.MessagesObjectStoreKey,
	}
}

// ShallowCompare compares this PolicyCheck with another RunNode.
func (n *PolicyCheck) ShallowCompare(other RunNode) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	o, ok := other.(*PolicyCheck)
	if !ok {
		return false
	}
	return n.StageName == o.StageName &&
		n.CheckType == o.CheckType &&
		n.Status == o.Status &&
		ptrStringEqual(n.LatestJobID, o.LatestJobID) &&
		policyCheckMessagesSummaryEqual(n.MessagesSummary, o.MessagesSummary) &&
		slices.EqualFunc(n.Policies, o.Policies, policyCheckPolicyEqual)
}

// policyCheckPolicyEqual reports whether two PolicyCheckPolicy values are equal, including the
// owner source, kind-specific data, and approver snapshot.
func policyCheckPolicyEqual(a, b *PolicyCheckPolicy) bool {
	return a.ID == b.ID &&
		a.EnforcementLevel == b.EnforcementLevel &&
		a.Provenance == b.Provenance &&
		policyCheckOPADataEqual(a.OPAData, b.OPAData) &&
		policyCheckModuleAttestationDataEqual(a.ModuleAttestationData, b.ModuleAttestationData) &&
		a.RequiredApprovals == b.RequiredApprovals &&
		a.Status == b.Status &&
		ptrStringEqual(a.MessagesObjectStoreKey, b.MessagesObjectStoreKey) &&
		slices.Equal(a.AllowedUserIDs, b.AllowedUserIDs) &&
		slices.Equal(a.AllowedServiceAccountIDs, b.AllowedServiceAccountIDs) &&
		slices.Equal(a.AllowedTeamIDs, b.AllowedTeamIDs)
}

// RunTaskStage represents a stage of a run (pre_plan, post_plan, ...) as a generic run node. It
// owns the policy checks evaluated at that stage (and, in future, run task results), and its Status
// is the aggregate verdict of those children. There is at most one task stage per stage name on a run.
type RunTaskStage struct {
	ID           string
	StageName    RunTaskStageName
	Status       RunTaskStageStatus
	PolicyChecks []*PolicyCheck
}

// GetID returns the node ID.
func (n *RunTaskStage) GetID() string { return n.ID }

// GetPath returns the run-relative path identifying this node within its run. The stage is unique
// per run, so it doubles as the path.
func (n *RunTaskStage) GetPath() string { return string(n.StageName) }

// GetGlobalID returns the task stage node's ID as a RunNode GID.
func (n *RunTaskStage) GetGlobalID() string {
	return RunNodeGID(n.ID)
}

// Copy creates a deep copy of the RunTaskStage, including its child policy checks.
func (n *RunTaskStage) Copy() RunNode {
	cp := &RunTaskStage{
		ID:        n.ID,
		StageName: n.StageName,
		Status:    n.Status,
	}
	if n.PolicyChecks != nil {
		cp.PolicyChecks = make([]*PolicyCheck, len(n.PolicyChecks))
		for i, check := range n.PolicyChecks {
			cp.PolicyChecks[i] = check.Copy().(*PolicyCheck)
		}
	}
	return cp
}

// ShallowCompare compares this task stage's own fields (id, stage, status) with another RunNode. It
// deliberately excludes the child policy checks, which Diff compares separately and attributes to
// their own node IDs.
func (n *RunTaskStage) ShallowCompare(other RunNode) bool {
	if n == nil && other == nil {
		return true
	}
	if n == nil || other == nil {
		return false
	}
	o, ok := other.(*RunTaskStage)
	if !ok {
		return false
	}
	return n.ID == o.ID && n.StageName == o.StageName && n.Status == o.Status
}

// Annotation limits, matching Phobos pipeline annotations so the two behave identically.
const (
	maxRunAnnotations           = 10
	maxRunAnnotationValueLength = 256
	// maxRunAnnotationLinkLength bounds the stored link purely as a storage/DoS guard (not a trust
	// decision — link content safety is handled at render time by the client). Annotation links are
	// structured CI-origin URLs (commit/MR/pipeline), so 256 is ample.
	maxRunAnnotationLinkLength = 256
)

// RunAnnotation is an immutable key/value pair (with an optional link) attached to a run at creation
// time. Annotations let a run be traced back to what created it (e.g. commit, repository, triggering
// job). Multiple annotations may share the same key.
//
// Immutability is enforced by the absence of any update path — annotations are set only at creation
// and there is no mutation API. Do not add one: a future UpdateRun must not modify annotations.
type RunAnnotation struct {
	// Fields are ordered pointer-first for struct alignment (fieldalignment); the logical order is
	// Key, Value, Link, as reflected in the proto, GraphQL, and DB representations.
	Link  *string `json:"link,omitempty"`
	Key   string  `json:"key"`
	Value string  `json:"value"`
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
	Annotations             []*RunAnnotation
	Plan                    Plan
	Apply                   *Apply
	TaskStages              []*RunTaskStage
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
	// HasAdvisoryFailures says at least one advisory policy this run evaluated failed. Advisory
	// failures never block a run — the check still reports passed — so the run's status cannot express
	// them
	HasAdvisoryFailures bool
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

// Validate validates the model. Annotation validation lives here (on the model) rather than in the
// service-layer CreateRunInput.Validate() so it applies uniformly to every run-creation path —
// GraphQL and gRPC both build the run model and reach core/run.Create, which calls Validate() before
// persisting. Validating at a single service boundary would miss the gRPC path.
func (r *Run) Validate() error {
	return r.validateAnnotations()
}

// validateAnnotations enforces the run annotation rules, which match Phobos pipeline annotations:
// at most maxRunAnnotations entries; each key and value non-empty; each key a valid name; each value
// no longer than maxRunAnnotationValueLength. Duplicate keys are allowed. The optional link's content is
// not validated by design, matching Phobos — link safety is enforced at render time by the client (the UI
// only makes http(s) links clickable), so any non-UI consumer must treat the link as untrusted — but its
// length is bounded as a storage guard.
func (r *Run) validateAnnotations() error {
	if len(r.Annotations) > maxRunAnnotations {
		return errors.New("maximum of %d annotations allowed", maxRunAnnotations, errors.WithErrorCode(errors.EInvalid))
	}

	for _, annotation := range r.Annotations {
		if annotation.Key == "" {
			return errors.New("annotation key cannot be empty", errors.WithErrorCode(errors.EInvalid))
		}
		if annotation.Value == "" {
			return errors.New("annotation value cannot be empty", errors.WithErrorCode(errors.EInvalid))
		}
		if err := verifyValidName(annotation.Key); err != nil {
			return errors.Wrap(err, "invalid annotation key")
		}
		if len(annotation.Value) > maxRunAnnotationValueLength {
			return errors.New("annotation value cannot be longer than %d characters", maxRunAnnotationValueLength, errors.WithErrorCode(errors.EInvalid))
		}
		if annotation.Link != nil && len(*annotation.Link) > maxRunAnnotationLinkLength {
			return errors.New("annotation link cannot be longer than %d characters", maxRunAnnotationLinkLength, errors.WithErrorCode(errors.EInvalid))
		}
	}

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

// ComputeHasAdvisoryFailures reports whether any advisory policy this run evaluated failed. A caller that
// changes a policy's status assigns the result to HasAdvisoryFailures rather than deriving the flag from
// the check it just wrote: the flag summarizes every check on the run, so a check being retried can only
// clear it by recomputing across the others.
func (r *Run) ComputeHasAdvisoryFailures() bool {
	for _, check := range r.AllPolicyChecks() {
		for _, policy := range check.Policies {
			if policy.EnforcementLevel == PolicyEnforcementAdvisory && policy.Status == PolicyCheckPolicyFailed {
				return true
			}
		}
	}
	return false
}

// IsComplete returns true if the run is in a completed state
func (r *Run) IsComplete() bool {
	return r.Status.IsFinalStatus()
}

// AllPolicyChecks returns every policy check across all of the run's task stages. The task stages
// are kept in canonical stage order (pre_plan, post_plan, post_apply) by the DB layer at load time,
// so this flattens them in that order.
func (r *Run) AllPolicyChecks() []*PolicyCheck {
	var checks []*PolicyCheck
	for _, stage := range r.TaskStages {
		checks = append(checks, stage.PolicyChecks...)
	}
	return checks
}

// TaskStageByStageName returns the task stage for the given run stage, or nil if the run has none.
func (r *Run) TaskStageByStageName(stage RunTaskStageName) *RunTaskStage {
	for _, s := range r.TaskStages {
		if s.StageName == stage {
			return s
		}
	}
	return nil
}

// PolicyCheckByID returns the policy check with the given node ID, or nil.
func (r *Run) PolicyCheckByID(id string) *PolicyCheck {
	for _, check := range r.AllPolicyChecks() {
		if check.ID == id {
			return check
		}
	}
	return nil
}

// PolicyCheckByPath returns the policy check with the given run-relative path (stage.checktype).
func (r *Run) PolicyCheckByPath(path string) *PolicyCheck {
	for _, check := range r.AllPolicyChecks() {
		if check.GetPath() == path {
			return check
		}
	}
	return nil
}

// NodeByPath returns the run node addressed by the given run-relative path — "plan", "apply", a task
// stage ("post_plan") or a policy check within one ("post_plan.opa") — or nil when the run has no
// node at that path. It is the read counterpart of the nodePath RetryRunNode takes.
func (r *Run) NodeByPath(path string) RunNode {
	switch path {
	case PlanNodePath:
		return &r.Plan
	case ApplyNodePath:
		// A speculative run has no apply node.
		if r.Apply == nil {
			return nil
		}
		return r.Apply
	}

	for _, stage := range r.TaskStages {
		if stage.GetPath() == path {
			return stage
		}
	}

	if check := r.PolicyCheckByPath(path); check != nil {
		return check
	}

	return nil
}

// TaskStageByID returns the task stage with the given node ID, or nil.
func (r *Run) TaskStageByID(id string) *RunTaskStage {
	for _, s := range r.TaskStages {
		if s.ID == id {
			return s
		}
	}
	return nil
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
		HasAdvisoryFailures:    r.HasAdvisoryFailures,
	}
	if r.Apply != nil {
		applyCopy := r.Apply.Copy().(*Apply)
		cp.Apply = applyCopy
	}
	if r.TaskStages != nil {
		cp.TaskStages = make([]*RunTaskStage, len(r.TaskStages))
		for i, stage := range r.TaskStages {
			cp.TaskStages[i] = stage.Copy().(*RunTaskStage)
		}
	}
	if r.Annotations != nil {
		cp.Annotations = make([]*RunAnnotation, len(r.Annotations))
		for i, a := range r.Annotations {
			var link *string
			if a.Link != nil {
				l := *a.Link
				link = &l
			}
			cp.Annotations[i] = &RunAnnotation{Key: a.Key, Value: a.Value, Link: link}
		}
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

	// Compare task stage nodes by ID. A stage present now but not in the comparison target (or
	// whose own status changed) is flagged by its own ID — its status is a persisted row.
	otherStages := make(map[string]*RunTaskStage, len(other.TaskStages))
	for _, stage := range other.TaskStages {
		otherStages[stage.ID] = stage
	}
	for _, stage := range r.TaskStages {
		otherStage, ok := otherStages[stage.ID]
		if !ok || !stage.ShallowCompare(otherStage) {
			changedIDs = append(changedIDs, stage.ID)
		}
	}

	// Compare policy checks by ID (across all stages). A check present now but not in the
	// comparison target (or whose content changed) is flagged by its own ID.
	otherChecks := make(map[string]*PolicyCheck)
	for _, check := range other.AllPolicyChecks() {
		otherChecks[check.ID] = check
	}
	for _, check := range r.AllPolicyChecks() {
		otherCheck, ok := otherChecks[check.ID]
		if !ok || !check.ShallowCompare(otherCheck) {
			changedIDs = append(changedIDs, check.ID)
		}
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
		r.HasAdvisoryFailures == other.HasAdvisoryFailures &&
		ptrStringEqual(r.ConfigurationVersionID, other.ConfigurationVersionID) &&
		ptrStringEqual(r.ModuleSource, other.ModuleSource) &&
		ptrStringEqual(r.ModuleVersion, other.ModuleVersion) &&
		ptrStringEqual(r.ForceCanceledBy, other.ForceCanceledBy) &&
		ptrTimeEqual(r.ForceCancelAvailableAt, other.ForceCancelAvailableAt) &&
		slices.Equal(r.TargetAddresses, other.TargetAddresses) &&
		slices.Equal(r.ModuleDigest, other.ModuleDigest) &&
		ptrStringEqual(r.VariablesObjectStoreKey, other.VariablesObjectStoreKey) &&
		runAnnotationsEqual(r.Annotations, other.Annotations)
}

// runAnnotationsEqual reports whether two annotation slices are equal, comparing each entry's key,
// value, and optional link in order. Annotations are ordered and duplicate keys are allowed, so this
// is an ordered comparison rather than a set comparison.
func runAnnotationsEqual(a, b []*RunAnnotation) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Value != b[i].Value || !ptrStringEqual(a[i].Link, b[i].Link) {
			return false
		}
	}
	return true
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
