// Package run contains core run-domain logic, including the pure function that
// creates (assembles and persists) a run.
package run

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
	corepolicy "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/policy"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/registry"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/rules"
	runvariables "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/run/variables"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/terraform"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/limits"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// CreateRunInput is the configuration for creating a run. The caller (the command layer) resolves
// the module and variables beforehand: ModuleVersion, ModuleDigest, and ModuleRegistrySource come
// from ResolveModule, and Variables are the FINAL, already-resolved run variables.
type CreateRunInput struct {
	Subject                string
	WorkspaceID            string
	TerraformVersion       string
	ConfigurationVersionID *string
	ModuleSource           *string
	ModuleVersion          *string
	ModuleDigest           []byte
	ModuleRegistrySource   registry.ModuleRegistrySource
	Comment                *string
	Speculative            *bool
	AutoApply              bool
	TargetAddresses        []string
	IsDestroy              bool
	Refresh                bool
	RefreshOnly            bool
	IsAssessmentRun        bool

	// SkipActivityEvent suppresses the run-creation activity event. It is set for
	// system-initiated runs (e.g. scheduler-triggered assessments) whose high frequency
	// would otherwise flood the activity feed with noise.
	SkipActivityEvent bool

	// VariablesObjectStoreKey is the key of the run variables the caller already uploaded (in Prepare,
	// before the transaction), stored on the run row at insert.
	VariablesObjectStoreKey string
}

// Create assembles the run model from the (already module-resolved) input and creates it within
// the caller's transaction: it validates the Terraform version, enforces managed-identity rules,
// inserts the run, enforces the per-workspace run limit, and records the creation activity event.
// It returns the persisted run.
//
// Module resolution (ResolveModule) and the run-variables upload (UploadRunVariables) are done by
// the caller before the transaction; Create takes their results as input. Create does not register
// or queue the run on the run graph — that is engine-level orchestration the caller performs after.
func Create(
	ctx context.Context,
	dbClient *db.Client,
	terraformCLIVersionConstraint string,
	ruleEnforcer rules.RuleEnforcer,
	limitChecker limits.LimitChecker,
	input *CreateRunInput,
) (*models.Run, error) {
	ws, err := dbClient.Workspaces.GetWorkspaceByID(ctx, input.WorkspaceID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace (ID %s) associated with run", input.WorkspaceID)
	}
	if ws == nil {
		return nil, errors.New("failed to get workspace associated with run", errors.WithErrorCode(errors.ENotFound))
	}

	// Check if the Terraform version is supported; default to the workspace's value.
	terraformVersion := ws.TerraformVersion
	if input.TerraformVersion != "" {
		versions, vErr := terraform.GetCLIVersions(ctx, terraformCLIVersionConstraint)
		if vErr != nil {
			return nil, vErr
		}
		if err := versions.Supported(input.TerraformVersion); err != nil {
			return nil, err
		}
		terraformVersion = input.TerraformVersion
	}

	// Enforce the workspace's option to prevent a destroy run.
	if input.IsDestroy && ws.PreventDestroyPlan {
		return nil, errors.New("Workspace does not allow destroy plan", errors.WithErrorCode(errors.EForbidden))
	}

	// Verify the subject may create a plan for all managed identities on the workspace.
	managedIdentities, err := dbClient.ManagedIdentities.GetManagedIdentitiesForWorkspace(ctx, input.WorkspaceID)
	if err != nil {
		return nil, err
	}
	var currentStateVersionID *string
	if ws.CurrentStateVersionID != "" {
		currentStateVersionID = &ws.CurrentStateVersionID
	}
	runDetails := &rules.RunDetails{
		RunStage:              models.JobPlanType,
		ModuleDigest:          input.ModuleDigest,
		CurrentStateVersionID: currentStateVersionID,
		ModuleSource:          input.ModuleRegistrySource,
		ModuleSemanticVersion: input.ModuleVersion,
	}
	for _, mi := range managedIdentities {
		miCopy := mi
		if err := ruleEnforcer.EnforceRules(ctx, &miCopy, runDetails); err != nil {
			return nil, err
		}
	}

	// Determine if the run is speculative.
	isSpeculative := false
	if input.ModuleSource != nil && input.Speculative != nil {
		isSpeculative = *input.Speculative
	}
	if input.ConfigurationVersionID != nil {
		configVersion, cvErr := dbClient.ConfigurationVersions.GetConfigurationVersionByID(ctx, *input.ConfigurationVersionID)
		if cvErr != nil {
			return nil, errors.Wrap(cvErr, "Failed to get configuration version associated with run")
		}
		if configVersion.Speculative && input.Speculative != nil && !*input.Speculative {
			return nil, errors.New("Speculative configuration version does not allow non-speculative runs", errors.WithErrorCode(errors.EInvalid))
		}
		isSpeculative = configVersion.Speculative
		if input.Speculative != nil {
			isSpeculative = *input.Speculative
		}
	}

	// Resolve the policies that apply to this run BEFORE creating it, so the task stage nodes — and
	// the policy check nodes they own, with their policy snapshots
	stageNames := []models.RunTaskStageName{
		models.RunTaskStageNamePrePlan,
		models.RunTaskStageNamePostPlan,
	}
	if !isSpeculative {
		stageNames = append(stageNames,
			models.RunTaskStageNamePreApply,
			models.RunTaskStageNamePostApply,
		)
	}

	var taskStages []*models.RunTaskStage
	for _, stageName := range stageNames {
		stagePolicies, err := resolveRunPolicies(ctx, dbClient, ws, managedIdentities, stageName, isSpeculative)
		if err != nil {
			return nil, errors.Wrap(err, "failed to resolve run policies")
		}
		if len(stagePolicies) > 0 {
			taskStages = append(taskStages, &models.RunTaskStage{
				StageName: stageName,
				Status:    models.RunTaskStageCreated,
				PolicyChecks: []*models.PolicyCheck{{
					StageName: stageName,
					CheckType: models.PolicyKindOPA,
					Status:    models.PolicyCheckCreated,
					Policies:  stagePolicies,
				}},
			})
		}
	}

	runModel := &models.Run{
		WorkspaceID:             input.WorkspaceID,
		ConfigurationVersionID:  input.ConfigurationVersionID,
		IsDestroy:               input.IsDestroy,
		Status:                  models.RunPending,
		CreatedBy:               input.Subject,
		AutoApply:               input.AutoApply,
		ModuleSource:            input.ModuleSource,
		ModuleVersion:           input.ModuleVersion,
		ModuleDigest:            input.ModuleDigest,
		TerraformVersion:        terraformVersion,
		TargetAddresses:         input.TargetAddresses,
		Refresh:                 input.Refresh,
		RefreshOnly:             input.RefreshOnly,
		IsAssessmentRun:         input.IsAssessmentRun,
		VariablesObjectStoreKey: &input.VariablesObjectStoreKey,
		Plan:                    models.Plan{Status: models.PlanCreated},
	}
	if input.Comment != nil {
		runModel.Comment = *input.Comment
	}
	if !isSpeculative {
		runModel.Apply = &models.Apply{Status: models.ApplyCreated}
	}
	runModel.TaskStages = taskStages

	created, err := dbClient.Runs.CreateRun(ctx, runModel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create run")
	}

	// Enforce the per-workspace run limit. The count includes the run just created, so a violation
	// rolls back the surrounding transaction.
	recentRuns, err := dbClient.Runs.GetRuns(ctx, &db.GetRunsInput{
		Filter: &db.RunFilter{
			TimeRangeStart: ptr.Time(created.Metadata.CreationTimestamp.Add(-limits.ResourceLimitTimePeriod)),
			WorkspaceID:    &created.WorkspaceID,
		},
		PaginationOptions: &pagination.Options{First: ptr.Int32(0)},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workspace's runs")
	}
	if err := limitChecker.CheckLimit(ctx,
		limits.ResourceLimitRunsPerWorkspacePerTimePeriod, recentRuns.PageInfo.TotalCount); err != nil {
		return nil, err
	}

	if !input.SkipActivityEvent {
		if _, err := activity.CreateActivityEvent(ctx, dbClient, &activity.CreateActivityEventInput{
			NamespacePath: &ws.FullPath,
			Action:        models.ActionCreate,
			TargetType:    models.TargetRun,
			TargetID:      created.Metadata.ID,
		}); err != nil {
			return nil, errors.Wrap(err, "failed to create activity event")
		}
	}

	return created, nil
}

// UploadRunVariables strips sensitive values, marshals the run variables, uploads them, and returns
// the object key and a link func. Commands call it from Prepare so the upload stays out of the transaction.
func UploadRunVariables(
	ctx context.Context,
	artifactStore workspace.ArtifactStore,
	workspaceID string,
	variables []runvariables.Variable,
) (db.RetainObjectRefFunc, string, error) {
	// Don't persist sensitive values to object storage.
	for i := range variables {
		if variables[i].Sensitive {
			variables[i].Value = nil
		}
	}

	variablesData, err := json.Marshal(variables)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to marshal run variables")
	}

	run := &models.Run{WorkspaceID: workspaceID}
	retainFn, varKey, err := artifactStore.UploadRunVariables(ctx, run, bytes.NewReader(variablesData))
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to upload run variables")
	}

	return retainFn, varKey, nil
}

// resolveRunPolicies collects every policy at the given stage that applies to the run into a slice
// of PolicyCheckPolicy snapshots, one entry per policy (no de-duplication). Policies are fetched
// from the run's ancestor groups and filtered in application code using their JSONB scope rules.
// Each entry carries the owning group as its Source, the policy's version constraint left
// unresolved, the optional digest lock, and the policy's approver spec. The constraint is resolved
// to a concrete package version by the policy-eval job when the check runs, so a policy tracking a
// range picks up newly published versions without a new run; a constraint that matches no uploaded
// version is that job's failed outcome, not a creation-time skip. The caller stores the result on
// the stage's policy check node's Policies field; nil when nothing applies.
// A run with no apply — a speculative plan, or an assessment run, which is created speculative — is
// enforced at each policy's speculative level instead of its declared one, so what a failure does to a
// plan nobody can apply is stated by the policy rather than decided here. That level can only be
// advisory or hard_mandatory (models.ValidSpeculativeRunEnforcementLevel), so no run without an apply
// can end up parked at a policy gate waiting for an approval that would unblock nothing.
func resolveRunPolicies(
	ctx context.Context,
	dbClient *db.Client,
	ws *models.Workspace,
	managedIdentities []models.ManagedIdentity,
	stage models.RunTaskStageName,
	isSpeculative bool,
) ([]*models.PolicyCheckPolicy, error) {
	policies, err := corepolicy.GetWorkspaceAssignedPolicies(ctx, dbClient, ws, managedIdentities, &stage)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}

	// One PolicyCheckPolicy per policy: snapshot the version constraint unresolved (empty ⇒ latest)
	// along with the owner group, enforcement, digest, and approver spec.
	checkPolicies := make([]*models.PolicyCheckPolicy, 0, len(policies))
	for i := range policies {
		a := &policies[i]

		// A policy whose kind isn't OPA is legitimately skipped: only OPA is supported today, and a
		// future policy type adds its own sibling check rather than being evaluated here.
		if a.Kind != models.PolicyKindOPA {
			continue
		}

		// An OPA-kind policy with no OPAData is a data-integrity violation, not something to skip. The
		// policy exists and declares a mandatory gate, but the data needed to evaluate it failed to
		// hydrate (a deserialization bug, a partial migration, a future regression). Skipping it would
		// drop the gate silently and let the run proceed as if the policy never existed. Fail run
		// creation instead so a dropped mandatory gate can't go unnoticed.
		if a.OPAData == nil {
			return nil, errors.New(
				"policy %s (%s) is OPA kind but has no OPA data; refusing to create run with a dropped policy gate",
				a.Name, a.Metadata.TRN, errors.WithErrorCode(errors.EInternal))
		}

		versionConstraint := ""
		if a.OPAData.PackageVersionConstraint != nil {
			versionConstraint = *a.OPAData.PackageVersionConstraint
		}

		provenance := models.PolicyCheckPolicyProvenance{
			GroupID:   a.GroupID,
			PolicyTRN: a.Metadata.TRN,
		}

		enforcementLevel := a.OPAData.EnforcementLevel
		if isSpeculative {
			enforcementLevel = a.OPAData.SpeculativeRunEnforcementLevel
		}

		description := ""
		if a.Description != nil {
			description = *a.Description
		}

		checkPolicies = append(checkPolicies, &models.PolicyCheckPolicy{
			ID:                       a.Metadata.ID,
			Name:                     a.Name,
			Description:              description,
			PackageSource:            a.OPAData.PackageSource,
			PackageVersionConstraint: versionConstraint,
			EnforcementLevel:         enforcementLevel,
			Provenance:               provenance,
			PackageDigest:            a.OPAData.PackageDigest,
			RequiredApprovals:        a.RequiredApprovals,
			AllowedUserIDs:           a.AllowedUserIDs,
			AllowedServiceAccountIDs: a.AllowedServiceAccountIDs,
			AllowedTeamIDs:           a.AllowedTeamIDs,
			Status:                   models.PolicyCheckPolicyPending,
		})
	}

	if len(checkPolicies) == 0 {
		return nil, nil
	}

	return checkPolicies, nil
}

// GetFederatedRegistry returns a getter that searches the workspace's parent group paths for a
// federated registry matching a host.
func GetFederatedRegistry(dbClient *db.Client, ws *models.Workspace) registry.FederatedRegistryGetterFunc {
	return func(ctx context.Context, hostname string) (*models.FederatedRegistry, error) {
		federatedRegistries, err := registry.GetFederatedRegistries(ctx, &registry.GetFederatedRegistriesInput{
			DBClient:  dbClient,
			GroupPath: ws.GetGroupPath(),
			Hostname:  &hostname,
		})
		if err != nil {
			return nil, errors.Wrap(err, "failed to get federated registries")
		}
		if len(federatedRegistries) > 0 {
			return federatedRegistries[0], nil
		}
		return nil, nil
	}
}
