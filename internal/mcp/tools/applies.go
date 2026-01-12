package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

// apply represents the details of a Terraform apply operation.
type apply struct {
	ApplyID      string             `json:"apply_id" jsonschema:"Unique identifier for this apply operation"`
	TRN          string             `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:apply:group/workspace/apply-id)"`
	Status       models.ApplyStatus `json:"status" jsonschema:"Current status: queued, running, finished, errored, or canceled"`
	WorkspaceID  string             `json:"workspace_id" jsonschema:"ID of the workspace where this apply is running"`
	TriggeredBy  string             `json:"triggered_by" jsonschema:"Email address of the user or service account that started this apply"`
	ErrorMessage *string            `json:"error_message,omitempty" jsonschema:"Error details if the apply failed (null if successful)"`
}

// getApplyInput defines the parameters for retrieving an apply.
type getApplyInput struct {
	ID string `json:"id" jsonschema:"required,Apply ID or TRN (e.g. Ul8yZ... or trn:apply:workspace-path/apply-id)"`
}

// getApplyOutput wraps the apply response.
type getApplyOutput struct {
	Apply apply `json:"apply" jsonschema:"The apply operation details"`
}

// GetApply returns an MCP tool for retrieving apply information.
// Use this to check the status of infrastructure changes being applied.
func GetApply(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getApplyInput, getApplyOutput]) {
	tool := mcp.Tool{
		Name:        "get_apply",
		Description: "Retrieve apply status and details. An apply executes the planned infrastructure changes. Check this to see if changes were successfully applied or if errors occurred.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Apply",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getApplyInput) (*mcp.CallToolResult, getApplyOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getApplyOutput{}, WrapMCPToolError(err, "failed to get apply %q", input.ID)
		}

		a, ok := fetchedModel.(*models.Apply)
		if !ok {
			return nil, getApplyOutput{}, NewMCPToolError("apply with id %s not found", input.ID)
		}

		output := getApplyOutput{
			Apply: apply{
				ApplyID:      a.GetGlobalID(),
				TRN:          a.Metadata.TRN,
				Status:       a.Status,
				WorkspaceID:  gid.ToGlobalID(types.WorkspaceModelType, a.WorkspaceID),
				ErrorMessage: a.ErrorMessage,
				TriggeredBy:  a.TriggeredBy,
			},
		}

		return nil, output, nil
	}

	return tool, handler
}
