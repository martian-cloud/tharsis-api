package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// plan represents a Terraform plan showing proposed infrastructure changes.
type plan struct {
	PlanID               string            `json:"plan_id" jsonschema:"Unique identifier for this plan"`
	TRN                  string            `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:plan:group/workspace/plan-id)"`
	Status               models.PlanStatus `json:"status" jsonschema:"Current status: queued, running, finished, errored, or canceled"`
	WorkspaceID          string            `json:"workspace_id" jsonschema:"ID of the workspace where this plan was created"`
	ErrorMessage         *string           `json:"error_message,omitempty" jsonschema:"Error details if the plan failed (null if successful)"`
	ResourceAdditions    int32             `json:"resource_additions" jsonschema:"Number of new resources that will be created"`
	ResourceChanges      int32             `json:"resource_changes" jsonschema:"Number of existing resources that will be modified"`
	ResourceDestructions int32             `json:"resource_destructions" jsonschema:"Number of resources that will be deleted"`
	HasChanges           bool              `json:"has_changes" jsonschema:"True if any resources will be added, changed, or destroyed"`
}

// getPlanInput defines the parameters for retrieving a plan.
type getPlanInput struct {
	ID string `json:"id" jsonschema:"required,Plan ID or TRN (e.g. Ul8yZ... or trn:plan:workspace-path/plan-id)"`
}

// getPlanOutput wraps the plan response.
type getPlanOutput struct {
	Plan plan `json:"plan" jsonschema:"The plan showing proposed infrastructure changes"`
}

// GetPlan returns an MCP tool for retrieving plan information.
// Use this to see what changes Terraform will make before applying them.
func GetPlan(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getPlanInput, getPlanOutput]) {
	tool := mcp.Tool{
		Name:        "get_plan",
		Description: "Retrieve plan status and resource change summary. A plan shows what infrastructure changes Terraform will make. Check resource_additions, resource_changes, and resource_destructions to understand the impact before applying.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Plan",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getPlanInput) (*mcp.CallToolResult, getPlanOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getPlanOutput{}, WrapMCPToolError(err, "failed to get plan %q", input.ID)
		}

		p, ok := fetchedModel.(*models.Plan)
		if !ok {
			return nil, getPlanOutput{}, NewMCPToolError("plan with id %s not found", input.ID)
		}

		output := getPlanOutput{
			Plan: plan{
				PlanID:               p.GetGlobalID(),
				TRN:                  p.Metadata.TRN,
				Status:               p.Status,
				WorkspaceID:          gid.ToGlobalID(types.WorkspaceModelType, p.WorkspaceID),
				ErrorMessage:         p.ErrorMessage,
				HasChanges:           p.HasChanges,
				ResourceAdditions:    p.Summary.ResourceAdditions,
				ResourceChanges:      p.Summary.ResourceChanges,
				ResourceDestructions: p.Summary.ResourceDestructions,
			},
		}

		return nil, output, nil
	}

	return tool, handler
}
