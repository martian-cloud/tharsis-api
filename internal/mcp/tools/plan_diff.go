package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	plandiff "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plan/action"
)

type getPlanDiffInput struct {
	PlanID string `json:"plan_id" jsonschema:"required,Plan ID or TRN (e.g. trn:plan:workspace-path/plan-id)"`
}

type getPlanDiffOutput struct {
	Resources []resourceDiff `json:"resources" jsonschema:"List of resource changes in the plan"`
	Outputs   []outputDiff   `json:"outputs" jsonschema:"List of output changes in the plan"`
}

type resourceDiff struct {
	Address      string        `json:"address" jsonschema:"Resource address (e.g. aws_instance.example)"`
	ResourceType string        `json:"resource_type" jsonschema:"Resource type (e.g. aws_instance)"`
	Action       action.Action `json:"action" jsonschema:"Action: CREATE, UPDATE, DELETE, DELETE_THEN_CREATE, CREATE_THEN_DELETE, NOOP, READ"`
	UnifiedDiff  string        `json:"unified_diff" jsonschema:"Unified diff showing attribute changes"`
	ProviderName string        `json:"provider_name" jsonschema:"Provider name"`
}

type outputDiff struct {
	OutputName  string        `json:"output_name" jsonschema:"Output name"`
	Action      action.Action `json:"action" jsonschema:"Action: CREATE, UPDATE, DELETE, NOOP"`
	UnifiedDiff string        `json:"unified_diff" jsonschema:"Unified diff showing value changes"`
}

// GetPlanDiff returns an MCP tool for retrieving the detailed diff of a plan.
func GetPlanDiff(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getPlanDiffInput, getPlanDiffOutput]) {
	tool := mcp.Tool{
		Name:        "get_plan_diff",
		Description: "Retrieve the detailed diff for a plan showing exactly what resource and output changes Terraform will make, including unified diffs of attribute changes.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Plan Diff",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getPlanDiffInput) (*mcp.CallToolResult, getPlanDiffOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.PlanID)
		if err != nil {
			return nil, getPlanDiffOutput{}, WrapMCPToolError(err, "failed to get plan %q", input.PlanID)
		}

		plan, ok := fetchedModel.(*models.Plan)
		if !ok {
			return nil, getPlanDiffOutput{}, NewMCPToolError("plan with id %s not found", input.PlanID)
		}

		diff, err := tc.servicesCatalog.RunService.GetPlanDiff(ctx, plan.Metadata.ID)
		if err != nil {
			return nil, getPlanDiffOutput{}, WrapMCPToolError(err, "failed to get plan diff for %q", input.PlanID)
		}

		return nil, toPlanDiffOutput(diff), nil
	}

	return tool, handler
}

func toPlanDiffOutput(d *plandiff.Diff) getPlanDiffOutput {
	out := getPlanDiffOutput{}

	for _, r := range d.Resources {
		out.Resources = append(out.Resources, resourceDiff{
			Address:      r.Address,
			ResourceType: r.ResourceType,
			Action:       r.Action,
			UnifiedDiff:  r.UnifiedDiff,
			ProviderName: r.ProviderName,
		})
	}

	for _, o := range d.Outputs {
		out.Outputs = append(out.Outputs, outputDiff{
			OutputName:  o.OutputName,
			Action:      o.Action,
			UnifiedDiff: o.UnifiedDiff,
		})
	}

	return out
}
