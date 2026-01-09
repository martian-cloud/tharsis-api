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
	Status                 models.RunStatus `json:"status" jsonschema:"Overall run status: pending, plan_queued, planning, planned, apply_queued, applying, applied, errored, canceled, or plan_only"`
	CreatedBy              string           `json:"created_by" jsonschema:"Email address of the user or service account that created this run"`
	TerraformVersion       string           `json:"terraform_version" jsonschema:"Terraform CLI version used (e.g. 1.5.0)"`
	PlanID                 string           `json:"plan_id,omitempty" jsonschema:"ID of the plan showing proposed changes"`
	ApplyID                string           `json:"apply_id,omitempty" jsonschema:"ID of the apply executing changes (empty if not yet applied)"`
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

// getRunInput defines the parameters for retrieving a run.
type getRunInput struct {
	ID string `json:"id" jsonschema:"required,Run ID or TRN (e.g. Ul8yZ... or trn:run:workspace-path/run-id)"`
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
		Description: "Retrieve run status and configuration. A run is the complete Terraform workflow (plan + optional apply). Check status to see progress, has_changes to see if infrastructure will change, and plan_id/apply_id to get detailed results.",
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
				PlanID:                 gid.ToGlobalID(types.PlanModelType, r.PlanID),
				ApplyID:                gid.ToGlobalID(types.ApplyModelType, r.ApplyID),
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
				HasChanges:             r.HasChanges,
				IsAssessmentRun:        r.IsAssessmentRun,
				AutoApply:              r.AutoApply,
				Refresh:                r.Refresh,
				RefreshOnly:            r.RefreshOnly,
			},
		}

		return nil, output, nil
	}

	return tool, handler
}
