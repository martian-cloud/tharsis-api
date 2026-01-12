package mcp

// DefaultInstructions returns the default MCP server instructions for Tharsis.
func DefaultInstructions() string {
	return `Tharsis is a remote Terraform backend for infrastructure management.

Resource IDs:
- IDs are returned in tool responses (e.g. run.run_id, workspace.workspace_id)
- TRNs format: trn:TYPE:PATH (e.g. trn:workspace:group/workspace-name)
- Use TRNs when you know the resource path but don't have the ID yet

Documentation:
- Search Tharsis docs for features, commands, or concepts

Safety:
- MUST get explicit user consent before write operations
- When polling status, use reasonable intervals (e.g. 5-10 seconds) to avoid server load
- Confirm before deleting (irreversible)`
}
