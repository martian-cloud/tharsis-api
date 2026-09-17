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
	Annotations            []*models.RunAnnotation
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

	checksByStage, err := resolveRunPolicies(ctx, dbClient, ws, managedIdentities, stageNames, isSpeculative)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve run policies")
	}

	var taskStages []*models.RunTaskStage
	for _, stageName := range stageNames {
		checksByKind := checksByStage[stageName]
		if len(checksByKind) == 0 {
			continue
		}

		// Ordered rather than a map range so a stage with both kinds always lists OPA first.
		var policyChecks []*models.PolicyCheck
		for _, kind := range []models.PolicyKind{models.PolicyKindOPA, models.PolicyKindModuleAttestation} {
			stagePolicies, ok := checksByKind[kind]
			if !ok {
				continue
			}
			policyChecks = append(policyChecks, &models.PolicyCheck{
				StageName: stageName,
				CheckType: kind,
				Status:    models.PolicyCheckCreated,
				Policies:  stagePolicies,
			})
		}

		taskStages = append(taskStages, &models.RunTaskStage{
			StageName:    stageName,
			Status:       models.RunTaskStageCreated,
			PolicyChecks: policyChecks,
		})
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
		Annotations:             input.Annotations,
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

	if err := runModel.Validate(); err != nil {
		return nil, err
	}
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

// resolveRunPolicies collects every policy across all of the given stages that applies to the run
// into PolicyCheckPolicy snapshots, grouped by stage and then by kind, one entry per policy (no
// de-duplication). Policies are fetched from the run's ancestor groups in a single query spanning
// every stage — rather than one query per stage — and filtered in application code using their
// JSONB scope rules and their own declared stage.
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
	stages []models.RunTaskStageName,
	isSpeculative bool,
) (map[models.RunTaskStageName]map[models.PolicyKind][]*models.PolicyCheckPolicy, error) {
	// No stage filter fetches policies at every stage in one round trip; each one is then bucketed
	// under its own declared stage below, dropping any stage the caller didn't ask for (a speculative
	// run's stages exclude pre_apply/post_apply).
	policies, err := corepolicy.GetWorkspaceAssignedPolicies(ctx, dbClient, ws, managedIdentities, nil)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}

	wantedStages := make(map[models.RunTaskStageName]bool, len(stages))
	for _, s := range stages {
		wantedStages[s] = true
	}

	checksByStage := make(map[models.RunTaskStageName]map[models.PolicyKind][]*models.PolicyCheckPolicy)
	for i := range policies {
		a := &policies[i]

		stage := a.Stage()

		// An unrecognized kind is forward-compatibility noise so we will skip it
		if stage != "" && !wantedStages[stage] && a.Kind != models.PolicyKindOPA && a.Kind != models.PolicyKindModuleAttestation {
			continue
		}

		description := ""
		if a.Description != nil {
			description = *a.Description
		}

		enforcementLevel := a.EnforcementLevel()
		if isSpeculative {
			enforcementLevel = a.SpeculativeRunEnforcementLevel()
		}

		checkPolicy := &models.PolicyCheckPolicy{
			ID:               a.Metadata.ID,
			Name:             a.Name,
			Description:      description,
			EnforcementLevel: enforcementLevel,
			Provenance: models.PolicyCheckPolicyProvenance{
				GroupID:   a.GroupID,
				PolicyTRN: a.Metadata.TRN,
			},
			RequiredApprovals:        a.RequiredApprovals,
			AllowedUserIDs:           a.AllowedUserIDs,
			AllowedServiceAccountIDs: a.AllowedServiceAccountIDs,
			AllowedTeamIDs:           a.AllowedTeamIDs,
			Status:                   models.PolicyCheckPolicyPending,
		}

		switch a.Kind {
		case models.PolicyKindOPA:
			// A policy whose kind data failed to hydrate is a data-integrity violation, not
			// something to skip. The policy exists and declares a mandatory gate, but the data
			// needed to evaluate it is missing (a deserialization bug, a partial migration, a
			// future regression). Skipping it would drop the gate silently and let the run
			// proceed as if the policy never existed. Fail run creation instead so a dropped
			// mandatory gate can't go unnoticed.
			if a.OPAData == nil {
				return nil, errors.New(
					"policy %s (%s) is OPA kind but has no OPA data; refusing to create run with a dropped policy gate",
					a.Name, a.Metadata.TRN, errors.WithErrorCode(errors.EInternal))
			}
			versionConstraint := ""
			if a.OPAData.PackageVersionConstraint != nil {
				versionConstraint = *a.OPAData.PackageVersionConstraint
			}
			checkPolicy.OPAData = &models.PolicyCheckOPAData{
				PackageSource:            a.OPAData.PackageSource,
				PackageVersionConstraint: versionConstraint,
				PackageDigest:            a.OPAData.PackageDigest,
			}
		case models.PolicyKindModuleAttestation:
			if a.ModuleAttestationData == nil {
				return nil, errors.New(
					"policy %s (%s) is module_attestation kind but has no module attestation data; refusing to create run with a dropped policy gate",
					a.Name, a.Metadata.TRN, errors.WithErrorCode(errors.EInternal))
			}
			checkPolicy.ModuleAttestationData = &models.PolicyCheckModuleAttestationData{
				PublicKey:          a.ModuleAttestationData.PublicKey,
				PredicateType:      a.ModuleAttestationData.PredicateType,
				VerifyStateLineage: a.ModuleAttestationData.VerifyStateLineage,
			}
		default:
			return nil, errors.New("policy %s (%s) has unsupported kind %s; refusing to create run with a dropped policy gate",
				a.Name, a.Metadata.TRN, a.Kind, errors.WithErrorCode(errors.EInternal))
		}

		if !wantedStages[stage] {
			continue
		}
		if checksByStage[stage] == nil {
			checksByStage[stage] = make(map[models.PolicyKind][]*models.PolicyCheckPolicy)
		}
		checksByStage[stage][a.Kind] = append(checksByStage[stage][a.Kind], checkPolicy)
	}

	return checksByStage, nil
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
