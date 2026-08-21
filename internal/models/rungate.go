package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*RunGate)(nil)

// maxOverrideCommentLength is the maximum length for a run gate's override comment.
const maxOverrideCommentLength = 1024

// RunGateStatus represents the status of a run gate.
type RunGateStatus string

// RunGateStatus constants.
const (
	// RunGatePending indicates the gate has not yet received enough approvals.
	RunGatePending RunGateStatus = "pending"
	// RunGateApproved indicates the gate has received the required number of approvals.
	RunGateApproved RunGateStatus = "approved"
	// RunGateOverridden indicates an admin bypassed the gate's approval requirements.
	RunGateOverridden RunGateStatus = "overridden"
	// RunGateCanceled indicates the gate was closed out without a decision because its run reached
	// a terminal status (e.g. discarded, canceled, or errored) while the gate was still pending.
	RunGateCanceled RunGateStatus = "canceled"
)

// RunGateSubjectType identifies the kind of principal allowed to approve a gate.
type RunGateSubjectType string

// RunGateSubjectType constants.
const (
	// RunGateSubjectUser is an allowed user.
	RunGateSubjectUser RunGateSubjectType = "user"
	// RunGateSubjectServiceAccount is an allowed service account.
	RunGateSubjectServiceAccount RunGateSubjectType = "service_account"
	// RunGateSubjectTeam is an allowed team.
	RunGateSubjectTeam RunGateSubjectType = "team"
)

// RunGateAllowedSubject is a single principal allowed to approve an approval rule, captured at
// gate creation so the gate can display who was allowed even after the subject is deleted.
type RunGateAllowedSubject struct {
	ID   string             `json:"id"`
	TRN  string             `json:"trn"`
	Type RunGateSubjectType `json:"type"`
}

// RunGateType identifies the kind of policy check a gate governs.
type RunGateType string

// RunGateType constants.
const (
	// RunGateTypeOPAPolicy is a gate governing an OPA policy check.
	RunGateTypeOPAPolicy RunGateType = "opa_policy"
)

// RunGateApprovalRule is the durable approval requirement for a single policy that soft-failed on
// the gate's policy check. Name is the run's PolicyCheckPolicy ID (the join key the UI uses to look
// up the policy's display info from the run); RequiredApprovals and AllowedSubjects are snapshotted
// at gate creation so the rule survives policy/subject deletion.
type RunGateApprovalRule struct {
	Name              string                   `json:"name"`
	RequiredApprovals int                      `json:"requiredApprovals"`
	AllowedSubjects   []*RunGateAllowedSubject `json:"allowedSubjects"`
}

// RunGate represents the approval gate for a single run policy check that parked at
// awaiting_override because one or more soft-mandatory policies failed. It carries one
// RunGateApprovalRule per soft-failed policy and is satisfied once every rule has reached its
// required number of approvals from its allowed subjects.
type RunGate struct {
	RunID         string
	WorkspaceID   string
	PolicyCheckID string
	Type          RunGateType
	ApprovalRules []*RunGateApprovalRule
	Status        RunGateStatus
	// OverriddenBy is the subject that bypassed the gate's approval requirements — a user's email
	// or a service account's resource path — and OverrideComment the reason they gave. Both are set
	// only when Status is RunGateOverridden; a gate that reached RunGateApproved by collecting its
	// approvals leaves them nil, which is what distinguishes the two ways a gate is cleared.
	OverriddenBy    *string
	OverrideComment *string
	Metadata        ResourceMetadata
}

// Satisfied reports whether every approval rule on the gate has reached its required number of
// approvals. An approve decision counts toward a rule when its CoveredRules includes the rule's
// name; reject decisions are advisory and never count. A rule with RequiredApprovals <= 0 is
// considered met.
//
// Approvals are counted by distinct principal, not by row: a principal approves at most once — the
// DB enforces one approval row per (gate, principal) and the decision command replaces a prior
// decision rather than adding a second — but de-duplicating here too means a stray duplicate row can
// never inflate a rule past its ceiling and clear a gate that should still be waiting.
func (g *RunGate) Satisfied(approvals []RunGateApproval) bool {
	for i := range g.ApprovalRules {
		rule := g.ApprovalRules[i]
		if rule.RequiredApprovals <= 0 {
			continue
		}
		approvers := make(map[string]struct{})
		for j := range approvals {
			a := &approvals[j]
			if a.Decision != RunGateDecisionApprove {
				continue
			}
			for _, name := range a.CoveredRules {
				if name == rule.Name {
					approvers[a.principalKey()] = struct{}{}
					break
				}
			}
		}
		if len(approvers) < rule.RequiredApprovals {
			return false
		}
	}
	return true
}

// GetID returns the Metadata ID.
func (g *RunGate) GetID() string {
	return g.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (g *RunGate) GetGlobalID() string {
	return gid.ToGlobalID(g.GetModelType(), g.Metadata.ID)
}

// GetModelType returns the model type.
func (g *RunGate) GetModelType() types.ModelType {
	return types.RunGateModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (g *RunGate) ResolveMetadata(key string) (*string, error) {
	return g.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid: Type and Status must be recognized enum
// values, the override comment must be within the length limit, and the OverriddenBy/OverrideComment
// fields must obey the invariant that they are set only on an overridden gate (an overridden gate
// must name the subject that overrode it; a gate cleared any other way leaves both nil).
func (g *RunGate) Validate() error {
	switch g.Type {
	case RunGateTypeOPAPolicy:
	default:
		return errors.New("run gate type %s is not supported", g.Type, errors.WithErrorCode(errors.EInvalid))
	}

	switch g.Status {
	case RunGatePending, RunGateApproved, RunGateOverridden, RunGateCanceled:
	default:
		return errors.New("run gate status %s is not supported", g.Status, errors.WithErrorCode(errors.EInvalid))
	}

	if g.OverrideComment != nil && len(*g.OverrideComment) > maxOverrideCommentLength {
		return errors.New("Override comment exceeds maximum length of %d characters", maxOverrideCommentLength, errors.WithErrorCode(errors.EInvalid))
	}

	// OverriddenBy/OverrideComment record who bypassed the gate and why, so they are meaningful only
	// for an overridden gate. A gate cleared by collecting approvals (approved) or closed out
	// (pending/canceled) must leave both nil; an overridden gate must record the overriding subject.
	if g.Status == RunGateOverridden {
		if g.OverriddenBy == nil {
			return errors.New("an overridden run gate must record the subject that overrode it",
				errors.WithErrorCode(errors.EInvalid))
		}
	} else if g.OverriddenBy != nil || g.OverrideComment != nil {
		return errors.New("overriddenBy and overrideComment may only be set on an overridden run gate",
			errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}
