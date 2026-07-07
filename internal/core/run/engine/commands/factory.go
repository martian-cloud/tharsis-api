package commands

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	corerun "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/engine/admission"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// createRunFunc creates and persists a run from the already-resolved input. It has
// the signature of corerun.Create with the shared collaborators (db client, rule
// enforcer, limit checker, artifact store, Terraform CLI version constraint) already
// bound by the factory, so a command depends only on the input. Holding the creation
// behind this field also lets tests substitute a stub instead of wiring every
// collaborator corerun.Create touches.
type createRunFunc func(ctx context.Context, input *corerun.CreateRunInput) (*models.Run, error)

// Factory builds run commands with their shared dependencies pre-wired, so callers
// (the run service, job service, and work item consumer) construct commands by passing only
// request data.
type Factory struct {
	logger                        logger.Logger
	dbClient                      *db.Client
	admitter                      *admission.Admitter
	variablesBuilder              *runvariables.Builder
	moduleResolver                registry.ModuleResolver
	terraformCLIVersionConstraint string
	ruleEnforcer                  rules.RuleEnforcer
	limitChecker                  limits.LimitChecker
	artifactStore                 workspace.ArtifactStore
	planParser                    plan.Parser
}

// NewFactory creates a run command Factory. The rule enforcer and plan parser are
// trivial and constructed here; the remaining dependencies are injected.
func NewFactory(
	logger logger.Logger,
	dbClient *db.Client,
	admitter *admission.Admitter,
	variablesBuilder *runvariables.Builder,
	moduleResolver registry.ModuleResolver,
	terraformCLIVersionConstraint string,
	limitChecker limits.LimitChecker,
	artifactStore workspace.ArtifactStore,
) *Factory {
	return &Factory{
		logger:                        logger,
		dbClient:                      dbClient,
		admitter:                      admitter,
		variablesBuilder:              variablesBuilder,
		moduleResolver:                moduleResolver,
		terraformCLIVersionConstraint: terraformCLIVersionConstraint,
		ruleEnforcer:                  rules.NewRuleEnforcer(dbClient),
		limitChecker:                  limitChecker,
		artifactStore:                 artifactStore,
		planParser:                    plan.NewParser(),
	}
}

// createRun returns a createRunFunc bound to the factory's shared collaborators, so a
// run-creation command depends only on the input it passes (and tests can substitute a stub).
func (f *Factory) createRun() createRunFunc {
	return func(ctx context.Context, input *corerun.CreateRunInput) (*models.Run, error) {
		return corerun.Create(ctx, f.dbClient, f.terraformCLIVersionConstraint,
			f.ruleEnforcer, f.limitChecker, f.artifactStore, input)
	}
}

// NewRun creates a user-initiated CreateRun command.
func (f *Factory) NewRun(in *NewRunInput) *CreateRun {
	return &CreateRun{
		dbClient:         f.dbClient,
		variablesBuilder: f.variablesBuilder,
		moduleResolver:   f.moduleResolver,
		createRun:        f.createRun(),
		in:               in,
	}
}

// NewCreateDestroyRun creates a CreateDestroyRun command.
func (f *Factory) NewCreateDestroyRun(in *CreateDestroyRunInput) *CreateDestroyRun {
	return &CreateDestroyRun{
		dbClient:         f.dbClient,
		variablesBuilder: f.variablesBuilder,
		moduleResolver:   f.moduleResolver,
		createRun:        f.createRun(),
		in:               in,
	}
}

// NewCreateReconcileRun creates a CreateReconcileRun command.
func (f *Factory) NewCreateReconcileRun(in *CreateReconcileRunInput) *CreateReconcileRun {
	return &CreateReconcileRun{
		dbClient:         f.dbClient,
		variablesBuilder: f.variablesBuilder,
		moduleResolver:   f.moduleResolver,
		createRun:        f.createRun(),
		in:               in,
	}
}

// NewCreateAssessmentRun creates a CreateAssessmentRun command.
func (f *Factory) NewCreateAssessmentRun(in *CreateAssessmentRunInput) *CreateAssessmentRun {
	return &CreateAssessmentRun{
		dbClient:         f.dbClient,
		variablesBuilder: f.variablesBuilder,
		moduleResolver:   f.moduleResolver,
		createRun:        f.createRun(),
		in:               in,
	}
}

// NewStartApply creates a StartApply command.
func (f *Factory) NewStartApply(in *StartApplyInput) *StartApply {
	return &StartApply{dbClient: f.dbClient, moduleResolver: f.moduleResolver, ruleEnforcer: f.ruleEnforcer, in: in}
}

// NewCancelRun creates a CancelRun command.
func (f *Factory) NewCancelRun(in *CancelRunInput) *CancelRun {
	return &CancelRun{dbClient: f.dbClient, in: in}
}

// NewSetRunAutoApply creates a SetRunAutoApply command.
func (f *Factory) NewSetRunAutoApply(in *SetRunAutoApplyInput) *SetRunAutoApply {
	return &SetRunAutoApply{dbClient: f.dbClient, in: in}
}

// NewRetryRunNode creates a RetryRunNode command.
func (f *Factory) NewRetryRunNode(in *RetryRunNodeInput) *RetryRunNode {
	return &RetryRunNode{dbClient: f.dbClient, in: in}
}

// NewDiscardRun creates a DiscardRun command.
func (f *Factory) NewDiscardRun(in *DiscardRunInput) *DiscardRun {
	return &DiscardRun{dbClient: f.dbClient, in: in}
}

// NewUndiscardRun creates an UndiscardRun command.
func (f *Factory) NewUndiscardRun(in *UndiscardRunInput) *UndiscardRun {
	return &UndiscardRun{dbClient: f.dbClient, in: in}
}

// NewUpdatePlanSummary creates an UpdatePlanSummary command.
func (f *Factory) NewUpdatePlanSummary(in *UpdatePlanSummaryInput) *UpdatePlanSummary {
	return &UpdatePlanSummary{dbClient: f.dbClient, planParser: f.planParser, artifactStore: f.artifactStore, in: in}
}

// NewUpdatePlan creates an UpdatePlan command.
func (f *Factory) NewUpdatePlan(planID string, hasChanges bool, errorMessage *string) *UpdatePlan {
	return &UpdatePlan{dbClient: f.dbClient, PlanID: planID, HasChanges: hasChanges, ErrorMessage: errorMessage}
}

// NewUpdateApply creates an UpdateApply command.
func (f *Factory) NewUpdateApply(applyID string, errorMessage *string) *UpdateApply {
	return &UpdateApply{dbClient: f.dbClient, ApplyID: applyID, ErrorMessage: errorMessage}
}

// NewQueueRun creates a QueueRun command.
func (f *Factory) NewQueueRun(runID string) *QueueRun {
	return &QueueRun{admitter: f.admitter, RunID: runID}
}

// NewSyncJobStatus creates a SyncJobStatus command. jobID identifies the reporting
// job so the command skips the node projection for a superseded job (e.g. after a
// retry). persistJob, if non-nil, runs inside the command transaction before the node
// sync (see SyncJobStatus.PersistJob).
func (f *Factory) NewSyncJobStatus(runID string, jobType models.JobType, jobID string, newStatus models.JobStatus, persistJob func(ctx context.Context) error) *SyncJobStatus {
	return &SyncJobStatus{logger: f.logger, RunID: runID, JobType: jobType, JobID: jobID, NewStatus: newStatus, PersistJob: persistJob}
}
