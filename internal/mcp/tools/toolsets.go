package tools

import (
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp/tools"
)

// Metadata on all available toolsets
var (
	ToolsetMetadataApplies = tools.ToolsetMetadata{
		Name:        "apply",
		Description: "Tools for retrieving apply status and execution details.",
	}
	ToolsetMetadataDocumentation = tools.ToolsetMetadata{
		Name:        "documentation",
		Description: "Tools for searching and retrieving Tharsis documentation.",
	}
	ToolsetMetadataJobs = tools.ToolsetMetadata{
		Name:        "job",
		Description: "Tools for retrieving job status and logs.",
	}
	ToolsetMetadataPlans = tools.ToolsetMetadata{
		Name:        "plan",
		Description: "Tools for retrieving plan status and resource changes.",
	}
	ToolsetMetadataRuns = tools.ToolsetMetadata{
		Name:        "run",
		Description: "Tools for retrieving run status and configuration details.",
	}
	ToolsetMetadataWorkspaces = tools.ToolsetMetadata{
		Name:        "workspace",
		Description: "Tools for retrieving workspace configuration.",
	}
)

// AllToolsets returns the list of all available toolset names.
// This is used as the default when no specific toolsets are configured.
func AllToolsets() []string {
	return []string{
		ToolsetMetadataApplies.Name,
		ToolsetMetadataDocumentation.Name,
		ToolsetMetadataJobs.Name,
		ToolsetMetadataPlans.Name,
		ToolsetMetadataRuns.Name,
		ToolsetMetadataWorkspaces.Name,
	}
}

// BuildToolsetGroup creates and configures all toolsets for the API MCP server.
func BuildToolsetGroup(readOnly bool, tc *ToolContext) (*tools.ToolsetGroup, error) {
	group := tools.NewToolsetGroup(readOnly)

	// Apply tools
	applies := tools.NewToolset(ToolsetMetadataApplies).
		AddReadTools(
			tools.NewServerTool(GetApply(tc)),
		)

	// Documentation tools
	docService, err := tools.NewDocumentSearchService(tc.httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create documentation search service: %v", err)
	}

	documentation := tools.NewToolset(ToolsetMetadataDocumentation).
		AddReadTools(
			tools.NewServerTool(tools.SearchDocumentation(docService)),
			tools.NewServerTool(tools.GetDocumentationPage(docService)),
		)

	// Job tools
	jobs := tools.NewToolset(ToolsetMetadataJobs).
		AddReadTools(
			tools.NewServerTool(GetJob(tc)),
			tools.NewServerTool(GetLatestJob(tc)),
			tools.NewServerTool(GetJobLogs(tc)),
		)

	// Plan tools
	plans := tools.NewToolset(ToolsetMetadataPlans).
		AddReadTools(
			tools.NewServerTool(GetPlan(tc)),
		)

	// Run tools
	runs := tools.NewToolset(ToolsetMetadataRuns).
		AddReadTools(
			tools.NewServerTool(GetRun(tc)),
		)

	// Workspace tools
	workspaces := tools.NewToolset(ToolsetMetadataWorkspaces).
		AddReadTools(
			tools.NewServerTool(GetWorkspace(tc)),
		)

	group.AddToolset(applies)
	group.AddToolset(documentation)
	group.AddToolset(jobs)
	group.AddToolset(plans)
	group.AddToolset(runs)
	group.AddToolset(workspaces)

	return group, nil
}
