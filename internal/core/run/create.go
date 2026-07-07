// Package run contains core run-domain logic, including the pure function that
// creates (assembles and persists) a run.
package run

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/aws/smithy-go/ptr"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/core/activity"
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

	// Variables are the final, already-resolved run variables.
	Variables []runvariables.Variable
}

// Create assembles the run model from the (already module-resolved) input and creates it within
// the caller's transaction: it validates the Terraform version, enforces managed-identity rules,
// inserts the run, enforces the per-workspace run limit, records the creation activity event, and
// uploads the run variables. It returns the persisted run.
//
// Module version/digest resolution is done by the caller via ResolveModule (before the
// transaction); Create takes the resolved values as input. Create does not register or queue the
// run on the run graph — that is engine-level orchestration the caller performs afterward.
func Create(
	ctx context.Context,
	dbClient *db.Client,
	terraformCLIVersionConstraint string,
	ruleEnforcer rules.RuleEnforcer,
	limitChecker limits.LimitChecker,
	artifactStore workspace.ArtifactStore,
	input *CreateRunInput,
) (*models.Run, error) {
	// The variables are already resolved by the caller. Strip sensitive values so they aren't
	// persisted to object storage.
	variables := input.Variables
	for i := range variables {
		if variables[i].Sensitive {
			variables[i].Value = nil
		}
	}
	variablesData, err := json.Marshal(variables)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal run variables")
	}

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

	runModel := &models.Run{
		WorkspaceID:            input.WorkspaceID,
		ConfigurationVersionID: input.ConfigurationVersionID,
		IsDestroy:              input.IsDestroy,
		Status:                 models.RunPending,
		CreatedBy:              input.Subject,
		AutoApply:              input.AutoApply,
		ModuleSource:           input.ModuleSource,
		ModuleVersion:          input.ModuleVersion,
		ModuleDigest:           input.ModuleDigest,
		TerraformVersion:       terraformVersion,
		TargetAddresses:        input.TargetAddresses,
		Refresh:                input.Refresh,
		RefreshOnly:            input.RefreshOnly,
		IsAssessmentRun:        input.IsAssessmentRun,
		Plan:                   models.Plan{Status: models.PlanCreated},
	}
	if input.Comment != nil {
		runModel.Comment = *input.Comment
	}
	if !isSpeculative {
		runModel.Apply = &models.Apply{Status: models.ApplyCreated}
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

	if err := artifactStore.UploadRunVariables(ctx, created, bytes.NewReader(variablesData)); err != nil {
		return nil, errors.Wrap(err, "failed to upload run variables")
	}

	return created, nil
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
