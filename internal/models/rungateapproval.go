package models

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

var _ Model = (*RunGateApproval)(nil)

// maxRunGateApprovalCommentLength is the maximum length for a run gate approval's comment.
const maxRunGateApprovalCommentLength = 1024

// RunGateDecision is the decision recorded on a run gate approval.
type RunGateDecision string

// RunGateDecision constants.
const (
	// RunGateDecisionApprove approves the gate.
	RunGateDecisionApprove RunGateDecision = "approve"
	// RunGateDecisionReject rejects the gate.
	RunGateDecisionReject RunGateDecision = "reject"
)

// RunGateApproval records a single approve/reject decision made against a run gate by a
// user or service account. CoveredRules lists the names of the gate's approval rules the decision
// counts toward — the rules the approver was eligible for at decision time — frozen here so a later
// change to the approver's team memberships does not retroactively alter the decision's effect.
type RunGateApproval struct {
	RunGateID        string
	UserID           *string
	ServiceAccountID *string
	CreatedBy        string
	Decision         RunGateDecision
	Comment          *string
	CoveredRules     []string
	Metadata         ResourceMetadata
}

// GetID returns the Metadata ID.
func (a *RunGateApproval) GetID() string {
	return a.Metadata.ID
}

// GetGlobalID returns the Metadata ID as a GID.
func (a *RunGateApproval) GetGlobalID() string {
	return gid.ToGlobalID(a.GetModelType(), a.Metadata.ID)
}

// GetModelType returns the model type.
func (a *RunGateApproval) GetModelType() types.ModelType {
	return types.RunGateApprovalModelType
}

// ResolveMetadata resolves the metadata fields for cursor-based pagination.
func (a *RunGateApproval) ResolveMetadata(key string) (*string, error) {
	return a.Metadata.resolveFieldValue(key)
}

// Validate returns an error if the model is not valid.
func (a *RunGateApproval) Validate() error {
	switch a.Decision {
	case RunGateDecisionApprove, RunGateDecisionReject:
	default:
		return errors.New("run gate decision %s is not supported", a.Decision, errors.WithErrorCode(errors.EInvalid))
	}
	// A decision is made by exactly one principal — a user or a service account, never both and never
	// neither. The table stores each in its own nullable column with a partial index that assumes it,
	// and principalKey (used by RunGate.Satisfied) relies on it to identify the approver.
	switch {
	case a.UserID != nil && a.ServiceAccountID != nil:
		return errors.New("run gate approval must be made by a user or a service account, not both",
			errors.WithErrorCode(errors.EInvalid))
	case a.UserID == nil && a.ServiceAccountID == nil:
		return errors.New("run gate approval must be made by a user or a service account",
			errors.WithErrorCode(errors.EInvalid))
	}
	if a.Comment != nil && len(*a.Comment) > maxRunGateApprovalCommentLength {
		return errors.New("Comment exceeds maximum length of %d characters", maxRunGateApprovalCommentLength, errors.WithErrorCode(errors.EInvalid))
	}
	return nil
}

// principalKey returns a stable key identifying the principal that made the decision, namespaced by
// principal kind so a user and a service account that share an ID string never collide. Validate
// enforces exactly one of UserID/ServiceAccountID; the CreatedBy fallback only guards a malformed
// row so de-duplication in RunGate.Satisfied still treats it as a single approver.
func (a *RunGateApproval) principalKey() string {
	switch {
	case a.UserID != nil:
		return "user:" + *a.UserID
	case a.ServiceAccountID != nil:
		return "sa:" + *a.ServiceAccountID
	default:
		return "createdBy:" + a.CreatedBy
	}
}
