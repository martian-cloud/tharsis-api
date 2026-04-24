package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsTool_Spec(t *testing.T) {
	tool := newSkillsTool([]skill{
		{Name: "troubleshoot", Description: "Help troubleshoot runs"},
	})

	spec := tool.Spec()
	assert.Equal(t, "load_skill", spec.Name)
	assert.Contains(t, spec.Description, "troubleshoot")
	assert.Contains(t, spec.Parameters, "name")
	assert.True(t, spec.Parameters["name"].Required)
}

func TestSkillsTool_RunFound(t *testing.T) {
	tool := newSkillsTool([]skill{
		{Name: "troubleshoot", Description: "Help troubleshoot", Data: "step 1: check logs"},
	})

	result, err := tool.Run(context.Background(), map[string]any{"name": "troubleshoot"})
	require.Nil(t, err)
	assert.Equal(t, "step 1: check logs", result["instructions"])
}

func TestSkillsTool_RunCaseInsensitive(t *testing.T) {
	tool := newSkillsTool([]skill{
		{Name: "Troubleshoot", Data: "instructions"},
	})

	result, err := tool.Run(context.Background(), map[string]any{"name": "troubleshoot"})
	require.Nil(t, err)
	assert.Equal(t, "instructions", result["instructions"])
}

func TestSkillsTool_RunNotFound(t *testing.T) {
	tool := newSkillsTool([]skill{})

	_, err := tool.Run(context.Background(), map[string]any{"name": "nonexistent"})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}
