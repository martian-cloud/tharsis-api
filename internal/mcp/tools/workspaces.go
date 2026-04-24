package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/gid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	workspacesvc "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/workspace"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/pagination"
)

// workspace represents a Terraform workspace configuration.
type workspace struct {
	WorkspaceID           string            `json:"workspace_id" jsonschema:"Unique identifier for this workspace"`
	TRN                   string            `json:"trn" jsonschema:"Tharsis Resource Name (e.g. trn:workspace:group/workspace-name)"`
	Name                  string            `json:"name" jsonschema:"Workspace name (last segment of full path)"`
	FullPath              string            `json:"full_path" jsonschema:"Complete path including parent groups (e.g. org/team/workspace-name)"`
	Description           string            `json:"description" jsonschema:"Human-readable description of this workspace's purpose"`
	GroupID               string            `json:"group_id" jsonschema:"ID of the parent group containing this workspace"`
	TerraformVersion      string            `json:"terraform_version" jsonschema:"Terraform CLI version used for runs (e.g. 1.5.0)"`
	MaxJobDuration        *int32            `json:"max_job_duration,omitempty" jsonschema:"Maximum minutes a job can run before timing out (null means no limit)"`
	CurrentJobID          string            `json:"current_job_id" jsonschema:"ID of the currently executing job (empty if no job running)"`
	CurrentStateVersionID string            `json:"current_state_version_id" jsonschema:"ID of the current Terraform state"`
	CreatedBy             string            `json:"created_by" jsonschema:"Email address of the user who created this workspace"`
	DirtyState            bool              `json:"dirty_state" jsonschema:"True if state is out of sync with configuration (drift detected)"`
	Locked                bool              `json:"locked" jsonschema:"True if workspace is locked (prevents new runs from starting)"`
	PreventDestroyPlan    bool              `json:"prevent_destroy_plan" jsonschema:"True if destroy plans are blocked (safety feature)"`
	EnableDriftDetection  *bool             `json:"enable_drift_detection,omitempty" jsonschema:"True if automatic drift detection is enabled"`
	RunnerTags            []string          `json:"runner_tags,omitempty" jsonschema:"Tags used to select which runner agents can execute jobs"`
	Labels                map[string]string `json:"labels,omitempty" jsonschema:"Key-value labels for organizing and filtering workspaces"`
}

// getWorkspaceInput defines the parameters for retrieving a workspace.
type getWorkspaceInput struct {
	ID string `json:"id" jsonschema:"required,Workspace ID or TRN (e.g. Ul8yZ... or trn:workspace:group-path/workspace-name)"`
}

// getWorkspaceOutput wraps the workspace response.
type getWorkspaceOutput struct {
	Workspace workspace `json:"workspace" jsonschema:"The workspace configuration and state"`
}

// GetWorkspace returns an MCP tool for retrieving workspace configuration.
// Use this to understand workspace settings when troubleshooting runs.
func GetWorkspace(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getWorkspaceInput, getWorkspaceOutput]) {
	tool := mcp.Tool{
		Name:        "get_workspace",
		Description: "Retrieve workspace configuration and settings. A workspace contains Terraform state and run configuration. Check this to see Terraform version, locked status, and current job.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Workspace",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getWorkspaceInput) (*mcp.CallToolResult, getWorkspaceOutput, error) {
		fetchedModel, err := tc.servicesCatalog.FetchModel(ctx, input.ID)
		if err != nil {
			return nil, getWorkspaceOutput{}, WrapMCPToolError(err, "failed to resolve workspace %q", input.ID)
		}

		ws, ok := fetchedModel.(*models.Workspace)
		if !ok {
			return nil, getWorkspaceOutput{}, NewMCPToolError("workspace with id %s not found", input.ID)
		}

		return nil, getWorkspaceOutput{
			Workspace: workspace{
				WorkspaceID:           ws.GetGlobalID(),
				TRN:                   ws.Metadata.TRN,
				Name:                  ws.Name,
				FullPath:              ws.FullPath,
				Description:           ws.Description,
				GroupID:               gid.ToGlobalID(types.GroupModelType, ws.GroupID),
				TerraformVersion:      ws.TerraformVersion,
				MaxJobDuration:        ws.MaxJobDuration,
				CurrentJobID:          gid.ToGlobalID(types.JobModelType, ws.CurrentJobID),
				CurrentStateVersionID: gid.ToGlobalID(types.StateVersionModelType, ws.CurrentStateVersionID),
				CreatedBy:             ws.CreatedBy,
				DirtyState:            ws.DirtyState,
				Locked:                ws.Locked,
				PreventDestroyPlan:    ws.PreventDestroyPlan,
				EnableDriftDetection:  ws.EnableDriftDetection,
				RunnerTags:            ws.RunnerTags,
				Labels:                ws.Labels,
			},
		}, nil
	}

	return tool, handler
}

// getWorkspacesInput defines the parameters for listing workspaces.
type getWorkspacesInput struct {
	Search *string `json:"search,omitempty" jsonschema:"Search by workspace name or path"`
	After  *string `json:"after,omitempty" jsonschema:"Cursor for forward pagination (from previous response)"`
	First  *int32  `json:"first,omitempty" jsonschema:"Number of items to return from the beginning (defaults to 100)"`
}

// getWorkspacesOutput wraps the workspaces list response.
type getWorkspacesOutput struct {
	Workspaces []workspace `json:"workspaces" jsonschema:"List of workspaces"`
	PageInfo   pageInfo    `json:"page_info" jsonschema:"Pagination information"`
}

// GetWorkspaces returns an MCP tool for listing workspaces with search and cursor-based pagination.
func GetWorkspaces(tc *ToolContext) (mcp.Tool, mcp.ToolHandlerFor[getWorkspacesInput, getWorkspacesOutput]) {
	tool := mcp.Tool{
		Name:        "get_workspaces",
		Description: "List workspaces with optional search and cursor-based pagination. Use 'search' to filter by name or path. Use 'first'/'after' for forward pagination or 'last'/'before' for backward pagination.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Get Workspaces",
			ReadOnlyHint: true,
		},
	}

	handler := func(ctx context.Context, _ *mcp.CallToolRequest, input getWorkspacesInput) (*mcp.CallToolResult, getWorkspacesOutput, error) {
		var first int32
		if input.First != nil {
			first = *input.First
		} else {
			first = 100
		}

		svcInput := &workspacesvc.GetWorkspacesInput{
			Search: input.Search,
			PaginationOptions: &pagination.Options{
				First: &first,
				After: input.After,
			},
		}

		result, err := tc.servicesCatalog.WorkspaceService.GetWorkspaces(ctx, svcInput)
		if err != nil {
			return nil, getWorkspacesOutput{}, WrapMCPToolError(err, "failed to list workspaces")
		}

		workspaces := make([]workspace, len(result.Workspaces))
		for i, ws := range result.Workspaces {
			workspaces[i] = workspace{
				WorkspaceID:           ws.GetGlobalID(),
				TRN:                   ws.Metadata.TRN,
				Name:                  ws.Name,
				FullPath:              ws.FullPath,
				Description:           ws.Description,
				GroupID:               gid.ToGlobalID(types.GroupModelType, ws.GroupID),
				TerraformVersion:      ws.TerraformVersion,
				MaxJobDuration:        ws.MaxJobDuration,
				CurrentJobID:          gid.ToGlobalID(types.JobModelType, ws.CurrentJobID),
				CurrentStateVersionID: gid.ToGlobalID(types.StateVersionModelType, ws.CurrentStateVersionID),
				CreatedBy:             ws.CreatedBy,
				DirtyState:            ws.DirtyState,
				Locked:                ws.Locked,
				PreventDestroyPlan:    ws.PreventDestroyPlan,
				EnableDriftDetection:  ws.EnableDriftDetection,
				RunnerTags:            ws.RunnerTags,
				Labels:                ws.Labels,
			}
		}

		pi := pageInfo{
			HasNextPage:     result.PageInfo.HasNextPage,
			HasPreviousPage: result.PageInfo.HasPreviousPage,
			TotalCount:      result.PageInfo.TotalCount,
		}

		if len(result.Workspaces) > 0 {
			startCursor, _ := result.PageInfo.Cursor(&result.Workspaces[0])
			endCursor, _ := result.PageInfo.Cursor(&result.Workspaces[len(result.Workspaces)-1])
			pi.StartCursor = startCursor
			pi.EndCursor = endCursor
		}

		return nil, getWorkspacesOutput{
			Workspaces: workspaces,
			PageInfo:   pi,
		}, nil
	}

	return tool, handler
}
