// Package prompts provides infrastructure for building MCP prompts.
//
// Prompts are reusable templates that guide LLMs through multi-step workflows.
// They can include arguments for customization and reference specific tools.
//
// Example usage:
//
//	func TroubleshootRunPrompt() (mcp.Prompt, mcp.PromptHandler) {
//	    return prompts.NewWorkflowPrompt(
//	        "troubleshoot_run",
//	        "Troubleshoot and fix the failed run {run_id}",
//	    ).
//	    AddRequiredArgument("run_id", "The ID of the failed run to troubleshoot").
//	    AddStep("get_run", "retrieve run details and error messages for {run_id}").
//	    AddStep("get_job_logs", "retrieve the full logs to see detailed error context").
//	    AddStep("", "analyze the error and explain it to the user").
//	    Build()
//	}
//
//	// Add to toolset
//	toolset := tools.NewToolset(metadata).
//	    AddPrompts(tools.NewServerPrompt(TroubleshootRunPrompt()))
package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// PromptStep represents a single step in a workflow prompt.
type PromptStep struct {
	Tool        string
	Description string
}

// WorkflowPrompt builds a workflow-style prompt with numbered steps.
type WorkflowPrompt struct {
	name        string
	description string
	arguments   []*mcp.PromptArgument
	steps       []PromptStep
}

// NewWorkflowPrompt creates a new workflow prompt builder.
func NewWorkflowPrompt(name, description string) *WorkflowPrompt {
	return &WorkflowPrompt{
		name:        name,
		description: description,
		arguments:   []*mcp.PromptArgument{},
		steps:       []PromptStep{},
	}
}

// AddRequiredArgument adds a required argument to the prompt.
func (w *WorkflowPrompt) AddRequiredArgument(name, description string) *WorkflowPrompt {
	w.arguments = append(w.arguments, &mcp.PromptArgument{
		Name:        name,
		Description: description,
		Required:    true,
	})
	return w
}

// AddOptionalArgument adds a required argument to the prompt.
func (w *WorkflowPrompt) AddOptionalArgument(name, description string) *WorkflowPrompt {
	w.arguments = append(w.arguments, &mcp.PromptArgument{
		Name:        name,
		Description: description,
	})
	return w
}

// AddStep adds a step to the workflow.
func (w *WorkflowPrompt) AddStep(tool, description string) *WorkflowPrompt {
	w.steps = append(w.steps, PromptStep{
		Tool:        tool,
		Description: description,
	})
	return w
}

// Build creates the MCP prompt and handler.
func (w *WorkflowPrompt) Build() (mcp.Prompt, mcp.PromptHandler) {
	prompt := mcp.Prompt{
		Name:        w.name,
		Description: w.description,
		Arguments:   w.arguments,
	}

	handler := func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Build the workflow text with argument substitution
		var sb strings.Builder
		sb.WriteString(w.description)
		sb.WriteString(":\n")

		for i, step := range w.steps {
			if step.Tool != "" {
				sb.WriteString(fmt.Sprintf("%d. Use %s to %s\n", i+1, step.Tool, step.Description))
			} else {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step.Description))
			}
		}

		// Substitute argument placeholders
		text := sb.String()
		for key, value := range req.Params.Arguments {
			text = strings.ReplaceAll(text, "{"+key+"}", value)
		}

		return &mcp.GetPromptResult{
			Description: w.description,
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: text},
				},
			},
		}, nil
	}

	return prompt, handler
}
