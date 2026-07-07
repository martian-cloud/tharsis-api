package tools

import (
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/mcp/tools"
)

// Metadata on all available toolsets
var (
	ToolsetMetadataDocumentation = tools.ToolsetMetadata{
		Name:        "documentation",
		Description: "Tools for searching and retrieving Tharsis documentation.",
	}
	ToolsetMetadataGroups = tools.ToolsetMetadata{
		Name:        "group",
		Description: "Tools for retrieving group configuration.",
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
		ToolsetMetadataDocumentation.Name,
		ToolsetMetadataGroups.Name,
		ToolsetMetadataJobs.Name,
		ToolsetMetadataPlans.Name,
		ToolsetMetadataRuns.Name,
		ToolsetMetadataWorkspaces.Name,
	}
}

// BuildToolsetGroup creates and configures all toolsets for the API MCP server.
func BuildToolsetGroup(readOnly bool, tc *ToolContext) *tools.ToolsetGroup {
	group := tools.NewToolsetGroup(readOnly)

	// Documentation tools
	docService := tools.NewDocumentSearchService(tc.httpClient)

	documentation := tools.NewToolset(ToolsetMetadataDocumentation).
		AddReadTools(
			tools.NewServerTool(tools.SearchDocumentation(docService)),
			tools.NewServerTool(tools.GetDocumentationPage(docService)),
		)

	// Group tools
	groups := tools.NewToolset(ToolsetMetadataGroups).
		AddReadTools(
			tools.NewServerTool(GetGroup(tc)),
			tools.NewServerTool(GetManagedIdentity(tc)),
			tools.NewServerTool(GetServiceAccount(tc)),
		)

	// Job tools
	jobs := tools.NewToolset(ToolsetMetadataJobs).
		AddReadTools(
			tools.NewServerTool(GetJob(tc)),
			tools.NewServerTool(GetJobLogs(tc)),
		)

	// Plan tools
	plans := tools.NewToolset(ToolsetMetadataPlans).
		AddReadTools(
			tools.NewServerTool(GetPlanDiff(tc)),
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
			tools.NewServerTool(GetWorkspaces(tc)),
		)

	group.AddToolset(documentation)
	group.AddToolset(groups)
	group.AddToolset(jobs)
	group.AddToolset(plans)
	group.AddToolset(runs)
	group.AddToolset(workspaces)

	return group
}
