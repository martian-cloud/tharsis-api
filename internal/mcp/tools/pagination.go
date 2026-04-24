package tools

// pageInfo contains pagination metadata.
type pageInfo struct {
	HasNextPage     bool    `json:"has_next_page" jsonschema:"True if more items exist after this page"`
	HasPreviousPage bool    `json:"has_previous_page" jsonschema:"True if more items exist before this page"`
	StartCursor     *string `json:"start_cursor,omitempty" jsonschema:"Cursor of the first item in this page"`
	EndCursor       *string `json:"end_cursor,omitempty" jsonschema:"Cursor of the last item in this page"`
	TotalCount      int32   `json:"total_count" jsonschema:"Total number of matching workspaces"`
}
