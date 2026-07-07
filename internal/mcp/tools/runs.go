package tools

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// run represents a complete Terraform workflow execution (plan + optional apply).
type run struct {
	RunID                  string           `json:"run_id" jsonschema:"Unique identifier for this run"`
	TRN                    string           `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:run:group/workspace/run-id)"`
	Status                 models.RunStatus `json:"status" jsonschema:"Overall run status: pending, queuing, plan_queued, planning, planned, queuing_apply, apply_queued, applying, applied, planned_and_finished, errored, canceled, or discarded"`
	CreatedBy              string           `json:"created_by" jsonschema:"Email address of the user or service account that created this run"`
	TerraformVersion       string           `json:"terraform_version" jsonschema:"Terraform CLI version used (e.g. 1.5.0)"`
	Plan                   *plan            `json:"plan,omitempty" jsonschema:"Planning phase: the proposed changes Terraform computed and their status"`
	Apply                  *apply           `json:"apply,omitempty" jsonschema:"Apply phase: execution of the planned changes (null if not yet applied)"`
	WorkspaceID            string           `json:"workspace_id,omitempty" jsonschema:"ID of the workspace where this run is executing"`
	ModuleSource           *string          `json:"module_source,omitempty" jsonschema:"Source location of the Terraform module (e.g. registry.terraform.io/hashicorp/aws)"`
	ModuleVersion          *string          `json:"module_version,omitempty" jsonschema:"Version of the module being used (e.g. 5.0.0)"`
	ForceCanceledBy        *string          `json:"force_canceled_by,omitempty" jsonschema:"Email of user who force-canceled this run (null if not force-canceled)"`
	ConfigurationVersionID *string          `json:"configuration_version_id,omitempty" jsonschema:"ID of the uploaded Terraform code being used"`
	ForceCancelAvailableAt *time.Time       `json:"force_cancel_available_at,omitempty" jsonschema:"When force-cancel becomes available (after graceful cancel timeout)"`
	TargetAddresses        []string         `json:"target_addresses,omitempty" jsonschema:"Specific resources being targeted (empty means all resources)"`
	ModuleDigest           string           `json:"module_digest,omitempty" jsonschema:"SHA256 hash of the module as hex string (for Tharsis registry modules only)"`
	IsDestroy              bool             `json:"is_destroy" jsonschema:"True if this run will destroy resources instead of creating them"`
	ForceCanceled          bool             `json:"force_canceled,omitempty" jsonschema:"True if this run was force-canceled (immediate stop)"`
	HasChanges             bool             `json:"has_changes,omitempty" jsonschema:"True if the plan detected any resource changes"`
	IsAssessmentRun        bool             `json:"is_assessment_run,omitempty" jsonschema:"True if this is an automated drift detection run"`
	AutoApply              bool             `json:"auto_apply,omitempty" jsonschema:"True if changes will be automatically applied after planning"`
	Refresh                bool             `json:"refresh,omitempty" jsonschema:"True if state will be refreshed before planning"`
	RefreshOnly            bool             `json:"refresh_only,omitempty" jsonschema:"True if this run only refreshes state without planning changes"`
}

// plan describes the planning phase of a run: the changes Terraform proposes to make.
type plan struct {
	PlanID       string            `json:"plan_id" jsonschema:"ID of the plan showing proposed changes"`
	Status       models.PlanStatus `json:"status" jsonschema:"Plan status: created, pending, queued, running, finished, errored, or canceled"`
	ErrorMessage *string           `json:"error_message,omitempty" jsonschema:"Error message if the plan failed (null otherwise)"`
	Summary      planSummary       `json:"summary" jsonschema:"Counts of the resource and output changes the plan computed"`
	DiffSize     int               `json:"diff_size,omitempty" jsonschema:"Size in bytes of the stored plan diff"`
	HasChanges   bool              `json:"has_changes" jsonschema:"True if the plan detected any resource or output changes"`
}

// planSummary breaks down the resource and output changes a plan computed.
type planSummary struct {
	ResourceAdditions    int32 `json:"resource_additions" jsonschema:"Number of resources to be created"`
	ResourceChanges      int32 `json:"resource_changes" jsonschema:"Number of resources to be updated in place"`
	ResourceDestructions int32 `json:"resource_destructions" jsonschema:"Number of resources to be destroyed"`
	ResourceImports      int32 `json:"resource_imports" jsonschema:"Number of resources to be imported into state"`
	ResourceDrift        int32 `json:"resource_drift" jsonschema:"Number of resources that drifted from the recorded state"`
	OutputAdditions      int32 `json:"output_additions" jsonschema:"Number of outputs to be added"`
	OutputChanges        int32 `json:"output_changes" jsonschema:"Number of outputs to be changed"`
	OutputDestructions   int32 `json:"output_destructions" jsonschema:"Number of outputs to be removed"`
}

// apply describes the apply phase of a run: executing the changes the plan proposed.
type apply struct {
	ApplyID      string             `json:"apply_id" jsonschema:"ID of the apply executing changes"`
	Status       models.ApplyStatus `json:"status" jsonschema:"Apply status: created, pending, queued, running, finished, errored, canceled, or skipped"`
	TriggeredBy  string             `json:"triggered_by,omitempty" jsonschema:"Email of the user or service account that triggered the apply"`
	Comment      string             `json:"comment,omitempty" jsonschema:"Optional comment provided when the apply was triggered"`
	ErrorMessage *string            `json:"error_message,omitempty" jsonschema:"Error message if the apply failed (null otherwise)"`
}

// getRunInput defines the parameters for retrieving a run.
type getRunInput struct {
	ID string `json:"id" jsonschema:"required, Run ID or TRN"`
}

// getRunOutput wraps the run response.
type getRunOutput struct {
	Run run `json:"run" jsonschema:"The complete run workflow details"`
}

// GetRun returns an MCP tool for retrieving run information.
// A run is the complete workflow: plan the changes, optionally apply them.
func GetRun(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getRunInput, getRunOutput]) {
	tool := mcp.Tool{
		Name:        "get_run",
		Description: "Retrieve run status and configuration. A run is the complete Terraform workflow (plan + optional apply). Check status to see progress, and the plan and apply objects for the proposed changes, change summary, and execution results.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Run",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getRunInput) (*mcp.CallToolResult, getRunOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getRunOutput{}, WrapMCPToolError(err, "failed to get run %q", input.ID)
		}

		r, ok := fetchedModel.(*models.Run)
		if !ok {
			return nil, getRunOutput{}, NewMCPToolError("run with id %s not found", input.ID)
		}

		output := getRunOutput{
			Run: run{
				RunID:                  r.GetGlobalID(),
				TRN:                    r.Metadata.TRN,
				Status:                 r.Status,
				CreatedBy:              r.CreatedBy,
				TerraformVersion:       r.TerraformVersion,
				WorkspaceID:            gid.ToGlobalID(types.WorkspaceModelType, r.WorkspaceID),
				ModuleSource:           r.ModuleSource,
				ModuleVersion:          r.ModuleVersion,
				ForceCanceledBy:        r.ForceCanceledBy,
				ConfigurationVersionID: r.ConfigurationVersionID,
				ForceCancelAvailableAt: r.ForceCancelAvailableAt,
				TargetAddresses:        r.TargetAddresses,
				ModuleDigest:           hex.EncodeToString(r.ModuleDigest),
				IsDestroy:              r.IsDestroy,
				ForceCanceled:          r.ForceCanceled,
				HasChanges:             r.HasChanges(),
				IsAssessmentRun:        r.IsAssessmentRun,
				AutoApply:              r.AutoApply,
				Refresh:                r.Refresh,
				RefreshOnly:            r.RefreshOnly,
			},
		}

		output.Run.Plan = &plan{
			PlanID:       r.Plan.GetGlobalID(),
			Status:       r.Plan.Status,
			ErrorMessage: r.Plan.ErrorMessage,
			Summary: planSummary{
				ResourceAdditions:    r.Plan.Summary.ResourceAdditions,
				ResourceChanges:      r.Plan.Summary.ResourceChanges,
				ResourceDestructions: r.Plan.Summary.ResourceDestructions,
				ResourceImports:      r.Plan.Summary.ResourceImports,
				ResourceDrift:        r.Plan.Summary.ResourceDrift,
				OutputAdditions:      r.Plan.Summary.OutputAdditions,
				OutputChanges:        r.Plan.Summary.OutputChanges,
				OutputDestructions:   r.Plan.Summary.OutputDestructions,
			},
			DiffSize:   r.Plan.DiffSize,
			HasChanges: r.Plan.HasChanges,
		}

		if applyNode := r.Apply; applyNode != nil {
			output.Run.Apply = &apply{
				ApplyID:      applyNode.GetGlobalID(),
				Status:       applyNode.Status,
				TriggeredBy:  applyNode.TriggeredBy,
				Comment:      applyNode.Comment,
				ErrorMessage: applyNode.ErrorMessage,
			}
		}

		return nil, output, nil
	}

	return tool, handler
}
