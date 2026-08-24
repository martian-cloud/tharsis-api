package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// GateDecision is the decision type for a run gate operation.
type GateDecision string

const (
	// GateDecisionApprove approves a run gate, allowing the run to proceed.
	GateDecisionApprove GateDecision = "approve"
	// GateDecisionReject rejects a run gate, blocking further progress.
	GateDecisionReject GateDecision = "reject"
	// GateDecisionOverride overrides a failed policy check on a run gate.
	GateDecisionOverride GateDecision = "override"
)

// SetRunGateDecisionInput carries everything SetRunGateDecision needs.
type SetRunGateDecisionInput struct {
	GateID   string
	Decision GateDecision
	Comment  *string
	Caller   auth.Caller
}

// SetRunGateDecision is a unified command for approve, reject, and override decisions on a
// run gate. It owns the gate pending check, covered-rules computation, and override
// authorization — the service resolves the caller and dispatches.
//
// Approve: records the approval, advances the gate to approved once every rule reaches its
// required number of approvals, and auto-overrides the blocked post-plan stage in the same
// transaction. Reject: records the decision and leaves the gate pending (advisory only).
// Override: bypasses approvals, setting the gate to overridden — recording the overriding subject
// and their comment on the gate, since no approval row is written — and immediately advancing the
// blocked post-plan stage.
type SetRunGateDecision struct {
	dbClient *db.Client
	in       *SetRunGateDecisionInput

	// Populated once Execute succeeds.
	UpdatedGate *models.RunGate
	Updated     *models.Run
}

// Execute dispatches to the approve/reject path or the override path based on the decision.
func (c *SetRunGateDecision) Execute(ctx context.Context, input *types.ExecuteInput) error {
	gate, err := c.dbClient.RunGates.GetRunGateByID(ctx, c.in.GateID)
	if err != nil {
		return errors.Wrap(err, "failed to get run gate")
	}
	if gate == nil {
		return errors.New("run gate with id %s not found", c.in.GateID, errors.WithErrorCode(errors.ENotFound))
	}
	if gate.Status != models.RunGatePending {
		return errors.New("run gate is not pending a decision", errors.WithErrorCode(errors.EConflict))
	}

	switch c.in.Decision {
	case GateDecisionApprove, GateDecisionReject:
		return c.executeDecision(ctx, gate, input)
	case GateDecisionOverride:
		return c.executeOverride(ctx, gate, input)
	default:
		return errors.New("unsupported gate decision: %s", c.in.Decision, errors.WithErrorCode(errors.EInvalid))
	}
}

func (c *SetRunGateDecision) executeDecision(ctx context.Context, gate *models.RunGate, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, gate.RunID)
	if err != nil {
		return err
	}

	check := run.PolicyCheckByID(gate.PolicyCheckID)
	if check == nil || check.Status != models.PolicyCheckSoftFailed {
		return errors.New("run does not have a policy check awaiting override", errors.WithErrorCode(errors.EConflict))
	}

	// Determine which of the gate's approval rules the caller is eligible for. Eligibility is
	// based on direct identity match or live team membership, and is frozen on the approval row
	// so future membership changes don't alter the decision's effect.
	coveredRules := []string{}
	for i := range gate.ApprovalRules {
		rule := gate.ApprovalRules[i]
		userIDs, saIDs, teamIDs := splitGateAllowedSubjects(rule.AllowedSubjects)
		eligible, eErr := rules.EligiblePrincipal(ctx, userIDs, saIDs, teamIDs)
		if eErr != nil {
			return eErr
		}
		if eligible {
			coveredRules = append(coveredRules, rule.Name)
		}
	}
	if len(coveredRules) == 0 {
		return errors.New("caller is not an eligible approver for this run gate", errors.WithErrorCode(errors.EForbidden))
	}

	// Resolve the caller's principal FK for the approval row (nullable so a later deletion is
	// tolerated); CreatedBy always carries the subject string.
	var userID, serviceAccountID *string
	if err = auth.HandleCaller(
		ctx,
		func(_ context.Context, c *auth.UserCaller) error {
			id := c.User.Metadata.ID
			userID = &id
			return nil
		},
		func(_ context.Context, c *auth.ServiceAccountCaller) error {
			id := c.ServiceAccountID
			serviceAccountID = &id
			return nil
		},
	); err != nil {
		return err
	}

	modelDecision := models.RunGateDecisionApprove
	if c.in.Decision == GateDecisionReject {
		modelDecision = models.RunGateDecisionReject
	}

	existingApprovals, err := c.dbClient.RunGateApprovals.GetRunGateApprovalsByGateID(ctx, gate.Metadata.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get existing run gate approvals")
	}

	var priorApproval *models.RunGateApproval
	for i := range existingApprovals {
		a := &existingApprovals[i]
		if (userID != nil && a.UserID != nil && *a.UserID == *userID) ||
			(serviceAccountID != nil && a.ServiceAccountID != nil && *a.ServiceAccountID == *serviceAccountID) {
			priorApproval = a
			break
		}
	}

	if priorApproval != nil {
		priorApproval.Decision = modelDecision
		priorApproval.Comment = c.in.Comment
		priorApproval.CoveredRules = coveredRules
		if err := priorApproval.Validate(); err != nil {
			return err
		}
		if _, err := c.dbClient.RunGateApprovals.UpdateRunGateApproval(ctx, priorApproval); err != nil {
			return errors.Wrap(err, "failed to update run gate approval")
		}
	} else {
		newApproval := &models.RunGateApproval{
			RunGateID:        gate.Metadata.ID,
			UserID:           userID,
			ServiceAccountID: serviceAccountID,
			CreatedBy:        c.in.Caller.GetSubject(),
			Decision:         modelDecision,
			Comment:          c.in.Comment,
			CoveredRules:     coveredRules,
		}
		if err := newApproval.Validate(); err != nil {
			return err
		}
		if _, err := c.dbClient.RunGateApprovals.CreateRunGateApproval(ctx, newApproval); err != nil {
			return errors.Wrap(err, "failed to create run gate approval")
		}
	}

	if c.in.Decision == GateDecisionReject {
		// Advisory rejection: gate stays pending, run unchanged.
		c.UpdatedGate = gate
		c.Updated = run
		return nil
	}

	// Approve path: advance the gate to approved once every rule reaches its required count.
	approvals, err := c.dbClient.RunGateApprovals.GetRunGateApprovalsByGateID(ctx, gate.Metadata.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get run gate approvals")
	}

	if !gate.Satisfied(approvals) {
		c.UpdatedGate = gate
		c.Updated = run
		return nil
	}

	gate.Status = models.RunGateApproved
	updatedGate, err := c.dbClient.RunGates.UpdateRunGate(ctx, gate)
	if err != nil {
		return errors.Wrap(err, "failed to update run gate status")
	}
	c.UpdatedGate = updatedGate
	c.Updated = run

	// The gate is fully approved: auto-override the blocked post-plan stage in the same
	// transaction so no separate OverrideRun call is needed.
	changes, err := statemachine.SetPolicyCheckStatus(run, check.GetPath(), models.PolicyCheckOverridden)
	if err != nil {
		return errors.Wrap(err, "failed to transition policy check to overridden")
	}
	return input.RunStore.AddRunChanges(run, changes...)
}

func (c *SetRunGateDecision) executeOverride(ctx context.Context, gate *models.RunGate, input *types.ExecuteInput) error {
	run, err := input.RunStore.GetRunByID(ctx, gate.RunID)
	if err != nil {
		return err
	}

	check := run.PolicyCheckByID(gate.PolicyCheckID)
	if check == nil || check.Status != models.PolicyCheckSoftFailed {
		return errors.New("run does not have a policy check awaiting override", errors.WithErrorCode(errors.EConflict))
	}

	// Record who bypassed the approvals and why. Unlike approve/reject, an override writes no
	// approval row, so the gate itself is the only durable trace of the decision.
	subject := c.in.Caller.GetSubject()
	gate.OverriddenBy = &subject
	gate.OverrideComment = c.in.Comment
	gate.Status = models.RunGateOverridden

	updatedGate, err := c.dbClient.RunGates.UpdateRunGate(ctx, gate)
	if err != nil {
		return errors.Wrap(err, "failed to update run gate status")
	}
	c.UpdatedGate = updatedGate

	changes, err := statemachine.SetPolicyCheckStatus(run, check.GetPath(), models.PolicyCheckOverridden)
	if err != nil {
		return errors.Wrap(err, "failed to transition policy check to overridden")
	}
	if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
		return err
	}
	c.Updated = run
	return nil
}

// splitGateAllowedSubjects partitions a gate's allowed subjects into raw user, service-account,
// and team ID slices for eligibility matching. The snapshot stores raw resource IDs.
func splitGateAllowedSubjects(subjects []*models.RunGateAllowedSubject) (userIDs, serviceAccountIDs, teamIDs []string) {
	for _, subject := range subjects {
		switch subject.Type {
		case models.RunGateSubjectUser:
			userIDs = append(userIDs, subject.ID)
		case models.RunGateSubjectServiceAccount:
			serviceAccountIDs = append(serviceAccountIDs, subject.ID)
		case models.RunGateSubjectTeam:
			teamIDs = append(teamIDs, subject.ID)
		}
	}
	return
}
