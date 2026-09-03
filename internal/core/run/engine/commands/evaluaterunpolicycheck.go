package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/statemachine"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// EvaluateRunPolicyCheck evaluates a policy check that has no job (module attestation today) and
// reports its outcome. It embeds ReportRunPolicyOutcomes: Prepare resolves the module and verifies
// each pinned policy to build the outcomes, then delegates to the embedded Prepare to upload
// diagnostics; Execute transitions the check to running (required by the transition table: only
// running may reach a verdict) and delegates to the embedded Execute to stamp the verdict. One home
// for the verdict rules, shared with the OPA path.
type EvaluateRunPolicyCheck struct {
	ReportRunPolicyOutcomes

	dbClient       *db.Client
	moduleResolver registry.ModuleResolver
	RunID          string

	// skip is set by Prepare when the check is no longer queued or running by the time this work
	// item is delivered — e.g. redelivered after its lease expired but the first attempt already
	// completed, or the check was independently retried or discarded. Both Prepare and Execute
	// become no-ops rather than acting on stale intent.
	skip bool
}

// Prepare loads the run and its workspace, no-ops (sets skip) if the check is no longer
// queued/running, resolves the module source, verifies each of the check's pinned policies via
// rules.VerifyModuleAttestation, and delegates to the embedded Prepare to upload the resulting
// diagnostics. It runs before the transaction is opened: module resolution is registry I/O.
//
// A verification error (the registry couldn't be reached, a malformed policy) is returned as-is so
// the work item is redelivered rather than recording a verdict on incomplete evidence; an
// unsatisfied attestation is instead a failed RunPolicyOutcome, which is a result, not an error.
func (c *EvaluateRunPolicyCheck) Prepare(ctx context.Context) error {
	run, err := c.dbClient.Runs.GetRunByID(ctx, c.RunID)
	if err != nil {
		return errors.Wrap(err, "failed to get run")
	}
	if run == nil {
		return errors.New("run with ID %s not found", c.RunID, errors.WithErrorCode(errors.ENotFound))
	}

	check := run.PolicyCheckByID(c.PolicyCheckID)
	if check == nil {
		return errors.New("run %s has no matching policy check node %s", c.RunID, c.PolicyCheckID, errors.WithErrorCode(errors.ENotFound))
	}

	if check.Status != models.PolicyCheckQueued && check.Status != models.PolicyCheckRunning {
		c.skip = true
		return nil
	}

	ws, err := c.dbClient.Workspaces.GetWorkspaceByID(ctx, run.WorkspaceID)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace")
	}
	if ws == nil {
		return errors.New("workspace %s not found", run.WorkspaceID, errors.WithErrorCode(errors.ENotFound))
	}

	var moduleSource registry.ModuleRegistrySource
	if run.ModuleSource != nil && run.ModuleDigest != nil {
		moduleSource, err = c.moduleResolver.ParseModuleRegistrySource(
			ctx, *run.ModuleSource, runvariables.ModuleRegistryToken(nil), corerun.GetFederatedRegistry(c.dbClient, ws))
		if err != nil {
			return errors.Wrap(err, "failed to parse module registry source")
		}
	}

	// This current state version will not change out from underneath the run since only a single apply run can be in-progress for
	// a given workspace
	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}

	outcomes := make([]RunPolicyOutcome, 0, len(check.Policies))
	for _, policy := range check.Policies {
		if policy.ModuleAttestationData == nil {
			return errors.New(
				"policy check policy %s on check %s has no module attestation data; refusing to evaluate a dropped policy gate",
				policy.ID, c.PolicyCheckID, errors.WithErrorCode(errors.EInternal))
		}

		diag, err := rules.VerifyModuleAttestation(ctx, c.dbClient, &rules.VerifyModuleAttestationInput{
			ModuleSource:          moduleSource,
			ModuleDigest:          run.ModuleDigest,
			ModuleSemanticVersion: run.ModuleVersion,
			PublicKey:             policy.ModuleAttestationData.PublicKey,
			PredicateType:         policy.ModuleAttestationData.PredicateType,
			VerifyStateLineage:    policy.ModuleAttestationData.VerifyStateLineage,
			CurrentStateVersionID: currentStateVersionID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to verify module attestation policy %s", policy.ID)
		}

		outcome := RunPolicyOutcome{PolicyID: policy.ID, Passed: diag == ""}
		if diag != "" {
			outcome.Messages = []string{diag}
		}
		outcomes = append(outcomes, outcome)
	}
	c.Outcomes = outcomes

	return c.ReportRunPolicyOutcomes.Prepare(ctx)
}

// Execute transitions the check from queued to running — the only status ReportRunPolicyOutcomes may
// set a verdict from — and delegates to the embedded Execute to stamp the outcomes and verdict.
func (c *EvaluateRunPolicyCheck) Execute(ctx context.Context, input *types.ExecuteInput) error {
	if c.skip {
		return nil
	}

	run, err := input.RunStore.GetRunByID(ctx, c.RunID)
	if err != nil {
		return err
	}

	check := run.PolicyCheckByID(c.PolicyCheckID)
	if check == nil {
		return errors.New("run %s has no matching policy check node %s", c.RunID, c.PolicyCheckID, errors.WithErrorCode(errors.ENotFound))
	}

	if check.Status == models.PolicyCheckQueued {
		changes, err := statemachine.SetPolicyCheckStatus(run, check.GetPath(), models.PolicyCheckRunning)
		if err != nil {
			return errors.Wrap(err, "failed to set policy check running")
		}
		if err := input.RunStore.AddRunChanges(run, changes...); err != nil {
			return err
		}
	}

	return c.ReportRunPolicyOutcomes.Execute(ctx, input)
}
