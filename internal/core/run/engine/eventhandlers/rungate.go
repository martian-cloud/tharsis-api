package eventhandlers

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// RunGateManager creates a run gate when a run's policy check reaches awaiting_override (a
// soft-mandatory policy failure). It is a stateful change handler so it runs inside the run command
// transaction: the gate and its query-index rows are committed atomically with the run change.
//
// Exactly one gate is created per blocked policy-check node, unconditionally: the gate is the block
// itself, and overriding it is the only way to clear the check. It carries one approval rule per
// soft-mandatory policy that failed and declares approvers, so a check whose soft failures declare no
// approvers gets a gate with no rules — nobody can approve it, and it is cleared by override alone.
type RunGateManager struct {
	dbClient *db.Client
	logger   logger.Logger
}

// NewRunGateManager creates a new RunGateManager.
func NewRunGateManager(logger logger.Logger, dbClient *db.Client) *RunGateManager {
	return &RunGateManager{dbClient: dbClient, logger: logger}
}

// HandleRunChanges implements RunChangeHandler. It creates gates when a run's policy check reaches
// awaiting_override, and closes out any still-pending gates when the check that was blocked is settled
// without a decision (retried or canceled) or the run reaches a terminal status (e.g. discarded,
// canceled, or errored) so an abandoned run never lingers in an approver's inbox.
func (h *RunGateManager) HandleRunChanges(ctx context.Context, changes []types.RunChange) error {
	for _, change := range changes {
		run := change.Run
		for _, sc := range change.NodeStatusChanges {
			switch c := sc.(type) {
			case statemachine.PolicyCheckStatusChange:
				switch c.NewStatus {
				case models.PolicyCheckSoftFailed:
					if err := h.createGatesForRun(ctx, run, c.CheckID); err != nil {
						return err
					}
				case models.PolicyCheckPending:
					// A check only reaches pending from errored, canceled or soft_failed, and one that
					// has never run goes created -> queued, so this transition is exactly a retry.
					if err := h.deleteGateForCheck(ctx, run, c.CheckID); err != nil {
						return err
					}
				case models.PolicyCheckCanceled:
					// The check was settled without its gate ever being decided — its stage errored or
					// the run was canceled/discarded under it — so the decision the gate was collecting
					// can no longer change anything. Cancel it here, at the check, as well as at the run
					// (closePendingGatesForRun): the run-level sweep only sees gates that already exist
					// when the run terminates, and a check can soft-fail and be canceled in the same
					// batch of changes. Canceling rather than deleting matches the run-level sweep and
					// keeps the gate and its approvals as history; a retry of the check deletes it,
					// because the verdict those approvals were given against is then re-evaluated.
					if err := h.cancelPendingGateForCheck(ctx, run, c.CheckID); err != nil {
						return err
					}
				}
			case statemachine.RunStatusChange:
				if !c.NewStatus.IsFinalStatus() {
					continue
				}
				if err := h.closePendingGatesForRun(ctx, run); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// closePendingGatesForRun sets any still-pending gate on the run to canceled. Approved gates are
// left untouched (they record a completed decision), so this is a no-op for a run that reached a
// terminal status through a normal override. It is idempotent: re-running only affects gates that
// are still pending.
func (h *RunGateManager) closePendingGatesForRun(ctx context.Context, run *models.Run) error {
	gatesResult, err := h.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{RunID: &run.Metadata.ID},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get run gates")
	}

	return h.cancelPendingGates(ctx, run, gatesResult.RunGates,
		"Canceled a pending run gate because its run reached a terminal status.")
}

// cancelPendingGateForCheck cancels the still-pending gate of a policy check that was canceled, so a
// check that will never be decided leaves no gate behind in its approvers' queue. It is scoped to the
// one check rather than the run because a run can hold a gate per stage, and a sibling's gate may still
// be awaiting a legitimate decision.
func (h *RunGateManager) cancelPendingGateForCheck(ctx context.Context, run *models.Run, checkID string) error {
	gatesResult, err := h.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{PolicyCheckIDs: []string{checkID}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get run gates")
	}

	return h.cancelPendingGates(ctx, run, gatesResult.RunGates,
		"Canceled a pending run gate because its policy check was canceled.")
}

// cancelPendingGates moves every pending gate in gates to canceled, leaving any gate that already
// carries a decision (approved, overridden, rejected) untouched — it records something that happened,
// which cancellation would erase. Only pending gates are touched, so it is idempotent.
func (h *RunGateManager) cancelPendingGates(ctx context.Context, run *models.Run, gates []models.RunGate, reason string) error {
	for i := range gates {
		gate := &gates[i]
		if gate.Status != models.RunGatePending {
			continue
		}
		gate.Status = models.RunGateCanceled
		if _, err := h.dbClient.RunGates.UpdateRunGate(ctx, gate); err != nil {
			return errors.Wrap(err, "failed to cancel pending run gate")
		}
		h.logger.WithContextFields(ctx).Infow(reason,
			"runID", run.Metadata.ID,
			"runGateID", gate.Metadata.ID,
			"policyCheckID", gate.PolicyCheckID,
			"runStatus", run.Status,
		)
	}

	return nil
}

// deleteGateForCheck discards the gate governing a retried policy check, so the re-evaluation
// creates a fresh one if its policies soft-fail again. Deleting rather than canceling is deliberate:
// the creation guard treats any existing gate for the check as "already handled", and a superseded
// gate's approval rules were snapshotted from the previous attempt. Any approvals already collected go
// with it — they approved a verdict that is being thrown away. Retrying an errored or canceled check is
// usually a no-op here (only a soft failure creates a gate); retrying a soft-failed one is the case
// this exists for.
func (h *RunGateManager) deleteGateForCheck(ctx context.Context, run *models.Run, checkID string) error {
	gatesResult, err := h.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{PolicyCheckIDs: []string{checkID}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get run gates")
	}

	gates := gatesResult.RunGates
	for i := range gates {
		gate := &gates[i]
		if err := h.dbClient.RunGates.DeleteRunGate(ctx, gate); err != nil {
			return errors.Wrap(err, "failed to delete run gate for retried policy check")
		}
		h.logger.WithContextFields(ctx).Infow("Deleted a run gate because its policy check was retried.",
			"runID", run.Metadata.ID,
			"runGateID", gate.Metadata.ID,
			"policyCheckID", checkID,
		)
	}

	return nil
}

func (h *RunGateManager) createGatesForRun(ctx context.Context, run *models.Run, checkID string) error {
	check := run.PolicyCheckByID(checkID)
	if check == nil {
		return nil
	}

	// A gate is only opened for a block that still stands. The soft failure that triggers this is a
	// change, not the current state: by the time it is handled the check may already have been settled
	// and the run ended — a check can soft-fail and be canceled in the same batch of changes when a
	// sibling check in its stage has failed a hard gate, and the run's terminal change is handled before
	// the check's (changes arrive in tree order, run first). Creating a gate then would leave it pending
	// in its approvers' queue forever, with no decision left for it to affect: the run-level sweep has
	// already passed, and its check is canceled.
	if check.Status != models.PolicyCheckSoftFailed {
		return nil
	}

	gateType, err := runGateTypeForCheck(check.CheckType)
	if err != nil {
		return err
	}

	// Idempotency: one gate per policy-check node. A gate already existing for this check means the
	// change is being reprocessed, so do nothing. This is scoped to the check rather than the run
	// because a run can soft-fail at more than one stage, and each check needs its own gate.
	existingResult, err := h.dbClient.RunGates.GetRunGates(ctx, &db.GetRunGatesInput{
		Filter: &db.RunGateFilter{PolicyCheckIDs: []string{check.ID}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to get existing run gates")
	}
	if len(existingResult.RunGates) > 0 {
		return nil
	}

	// Build one approval rule per soft-mandatory policy that failed and declares approvers. Advisory
	// never blocks and hard cannot be overridden, so neither produces a rule. A soft failure with no
	// approvers produces no rule either, leaving a rule-less gate that only an override can clear.
	rules := []*models.RunGateApprovalRule{}

	for i := range check.Policies {
		policy := check.Policies[i]

		if policy.Status != models.PolicyCheckPolicyFailed ||
			policy.EnforcementLevel != models.PolicyEnforcementSoftMandatory ||
			policy.RequiredApprovals < 1 {
			continue
		}

		subjects, err := h.resolveAllowedSubjects(ctx, policy)
		if err != nil {
			return err
		}

		rules = append(rules, &models.RunGateApprovalRule{
			Name:              policy.ID,
			RequiredApprovals: policy.RequiredApprovals,
			AllowedSubjects:   subjects,
		})
	}

	// The DB layer seeds the run_gate_allowed_* query-index tables from the gate's rules' allowed
	// subjects. A rule-less gate seeds nothing, so it never surfaces in an approver's inbox — the
	// eligibility filter matches on those tables.
	gate := &models.RunGate{
		RunID:         run.Metadata.ID,
		WorkspaceID:   run.WorkspaceID,
		PolicyCheckID: check.ID,
		Type:          gateType,
		ApprovalRules: rules,
		Status:        models.RunGatePending,
	}
	if err := gate.Validate(); err != nil {
		return errors.Wrap(err, "invalid run gate")
	}
	if _, err := h.dbClient.RunGates.CreateRunGate(ctx, gate); err != nil {
		return errors.Wrap(err, "failed to create run gate")
	}

	h.logger.WithContextFields(ctx).Infow("Created a run gate for a policy check with soft-failed policies.",
		"runID", run.Metadata.ID,
		"runGateType", gateType,
		"ruleCount", len(rules),
	)

	return nil
}

// runGateTypeForCheck maps a policy check's engine type to the corresponding run gate type.
func runGateTypeForCheck(checkType models.PolicyKind) (models.RunGateType, error) {
	switch checkType {
	case models.PolicyKindOPA:
		return models.RunGateTypeOPAPolicy, nil
	case models.PolicyKindModuleAttestation:
		return models.RunGateTypeModuleAttestation, nil
	default:
		return "", errors.New("unsupported policy check type %q for run gate", checkType)
	}
}

// resolveAllowedSubjects resolves a policy's raw approver principal IDs into durable allowed-subject
// snapshots (raw resource ID + TRN + type), skipping any principal that has since been deleted. Each
// principal type is resolved with a single multi-ID query rather than one query per ID.
func (h *RunGateManager) resolveAllowedSubjects(ctx context.Context, policy *models.PolicyCheckPolicy) ([]*models.RunGateAllowedSubject, error) {
	subjects := []*models.RunGateAllowedSubject{}

	if len(policy.AllowedUserIDs) > 0 {
		usersResult, err := h.dbClient.Users.GetUsers(ctx, &db.GetUsersInput{
			Filter: &db.UserFilter{UserIDs: policy.AllowedUserIDs},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get allowed users")
		}
		for i := range usersResult.Users {
			user := &usersResult.Users[i]
			subjects = append(subjects, &models.RunGateAllowedSubject{
				ID:   user.Metadata.ID,
				TRN:  user.Metadata.TRN,
				Type: models.RunGateSubjectUser,
			})
		}
	}

	if len(policy.AllowedServiceAccountIDs) > 0 {
		saResult, err := h.dbClient.ServiceAccounts.GetServiceAccounts(ctx, &db.GetServiceAccountsInput{
			Filter: &db.ServiceAccountFilter{ServiceAccountIDs: policy.AllowedServiceAccountIDs},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get allowed service accounts")
		}
		for i := range saResult.ServiceAccounts {
			sa := &saResult.ServiceAccounts[i]
			subjects = append(subjects, &models.RunGateAllowedSubject{
				ID:   sa.Metadata.ID,
				TRN:  sa.Metadata.TRN,
				Type: models.RunGateSubjectServiceAccount,
			})
		}
	}

	if len(policy.AllowedTeamIDs) > 0 {
		teamsResult, err := h.dbClient.Teams.GetTeams(ctx, &db.GetTeamsInput{
			Filter: &db.TeamFilter{TeamIDs: policy.AllowedTeamIDs},
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get allowed teams")
		}
		for i := range teamsResult.Teams {
			team := &teamsResult.Teams[i]
			subjects = append(subjects, &models.RunGateAllowedSubject{
				ID:   team.Metadata.ID,
				TRN:  team.Metadata.TRN,
				Type: models.RunGateSubjectTeam,
			})
		}
	}

	return subjects, nil
}
