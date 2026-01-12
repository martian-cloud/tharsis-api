package prompts

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWorkflowPrompt(t *testing.T) {
	wp := NewWorkflowPrompt("test_prompt", "Test prompt description")

	assert.NotNil(t, wp)
	assert.Equal(t, "test_prompt", wp.name)
	assert.Equal(t, "Test prompt description", wp.description)
	assert.Empty(t, wp.arguments)
	assert.Empty(t, wp.steps)
}

func TestWorkflowPromptAddArgument(t *testing.T) {
	wp := NewWorkflowPrompt("test", "Test").
		AddRequiredArgument("arg1", "First argument").
		AddRequiredArgument("arg2", "Second argument")

	assert.Len(t, wp.arguments, 2)
	assert.Equal(t, "arg1", wp.arguments[0].Name)
	assert.Equal(t, "First argument", wp.arguments[0].Description)
	assert.True(t, wp.arguments[0].Required)
	assert.Equal(t, "arg2", wp.arguments[1].Name)
}

func TestWorkflowPromptAddStep(t *testing.T) {
	wp := NewWorkflowPrompt("test", "Test").
		AddStep("get_run", "retrieve run details").
		AddStep("", "analyze the results")

	assert.Len(t, wp.steps, 2)
	assert.Equal(t, "get_run", wp.steps[0].Tool)
	assert.Equal(t, "retrieve run details", wp.steps[0].Description)
	assert.Equal(t, "", wp.steps[1].Tool)
	assert.Equal(t, "analyze the results", wp.steps[1].Description)
}

func TestWorkflowPromptBuild(t *testing.T) {
	type testCase struct {
		name       string
		builder    *WorkflowPrompt
		args       map[string]string
		expectText []string
		expectDesc string
	}

	tests := []testCase{
		{
			name: "simple workflow",
			builder: NewWorkflowPrompt("test", "Test workflow").
				AddStep("tool1", "do something"),
			args:       map[string]string{},
			expectText: []string{"Test workflow:", "1. Use tool1 to do something"},
			expectDesc: "Test workflow",
		},
		{
			name: "workflow with arguments",
			builder: NewWorkflowPrompt("test", "Process {item_id}").
				AddRequiredArgument("item_id", "The item ID").
				AddStep("get_item", "retrieve {item_id}"),
			args:       map[string]string{"item_id": "123"},
			expectText: []string{"Process 123:", "1. Use get_item to retrieve 123"},
			expectDesc: "Process {item_id}",
		},
		{
			name: "workflow with tool and non-tool steps",
			builder: NewWorkflowPrompt("test", "Multi-step process").
				AddStep("tool1", "first step").
				AddStep("", "analyze results").
				AddStep("tool2", "final step"),
			args: map[string]string{},
			expectText: []string{
				"Multi-step process:",
				"1. Use tool1 to first step",
				"2. analyze results",
				"3. Use tool2 to final step",
			},
			expectDesc: "Multi-step process",
		},
		{
			name: "multiple argument substitution",
			builder: NewWorkflowPrompt("test", "Process {id} in {env}").
				AddRequiredArgument("id", "ID").
				AddRequiredArgument("env", "Environment").
				AddStep("get_resource", "retrieve {id} from {env}"),
			args:       map[string]string{"id": "abc", "env": "prod"},
			expectText: []string{"Process abc in prod:", "1. Use get_resource to retrieve abc from prod"},
			expectDesc: "Process {id} in {env}",
		},
		{
			name:       "empty workflow",
			builder:    NewWorkflowPrompt("test", "Empty"),
			args:       map[string]string{},
			expectText: []string{"Empty:"},
			expectDesc: "Empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, handler := tt.builder.Build()

			assert.Equal(t, tt.builder.name, prompt.Name)
			assert.Equal(t, tt.builder.description, prompt.Description)
			assert.Equal(t, len(tt.builder.arguments), len(prompt.Arguments))

			result, err := handler(context.Background(), &mcp.GetPromptRequest{
				Params: &mcp.GetPromptParams{Arguments: tt.args},
			})

			require.NoError(t, err)
			assert.Equal(t, tt.expectDesc, result.Description)
			require.Len(t, result.Messages, 1)
			assert.Equal(t, mcp.Role("user"), result.Messages[0].Role)

			textContent, ok := result.Messages[0].Content.(*mcp.TextContent)
			require.True(t, ok)

			for _, expected := range tt.expectText {
				assert.Contains(t, textContent.Text, expected)
			}
		})
	}
}

func TestWorkflowPromptChaining(t *testing.T) {
	wp := NewWorkflowPrompt("test", "Test").
		AddRequiredArgument("arg1", "Arg 1").
		AddStep("tool1", "step 1").
		AddRequiredArgument("arg2", "Arg 2").
		AddStep("tool2", "step 2")

	assert.Len(t, wp.arguments, 2)
	assert.Len(t, wp.steps, 2)
}
