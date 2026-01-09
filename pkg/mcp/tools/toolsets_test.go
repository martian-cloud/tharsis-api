package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseToolsets(t *testing.T) {
	type testCase struct {
		name            string
		input           string
		expectedValid   []string
		expectedInvalid []string
	}

	tests := []testCase{
		{
			name:            "empty string",
			input:           "",
			expectedValid:   nil,
			expectedInvalid: nil,
		},
		{
			name:            "single valid toolset",
			input:           "test",
			expectedValid:   []string{"test"},
			expectedInvalid: []string{},
		},
		{
			name:            "multiple valid toolsets",
			input:           "test,another,third_one",
			expectedValid:   []string{"another", "test", "third_one"},
			expectedInvalid: []string{},
		},
		{
			name:            "with whitespace",
			input:           " test , another , third ",
			expectedValid:   []string{"another", "test", "third"},
			expectedInvalid: []string{},
		},
		{
			name:            "with invalid names",
			input:           "test,Invalid,_bad,good_one",
			expectedValid:   []string{"good_one", "test"},
			expectedInvalid: []string{"Invalid", "_bad"},
		},
		{
			name:            "all invalid",
			input:           "AA,BB,CC",
			expectedValid:   []string{},
			expectedInvalid: []string{"AA", "BB", "CC"},
		},
		{
			name:            "with empty entries",
			input:           "test,,another",
			expectedValid:   []string{"another", "test"},
			expectedInvalid: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, invalid := ParseToolsets(tt.input)
			assert.Equal(t, tt.expectedValid, valid)
			assert.Equal(t, tt.expectedInvalid, invalid)
		})
	}
}

func TestParseTools(t *testing.T) {
	type testCase struct {
		name     string
		input    string
		expected []string
	}

	tests := []testCase{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single tool",
			input:    "get_run",
			expected: []string{"get_run"},
		},
		{
			name:     "multiple tools",
			input:    "get_run,get_job,get_workspace",
			expected: []string{"get_job", "get_run", "get_workspace"},
		},
		{
			name:     "with whitespace",
			input:    " get_run , get_job , get_workspace ",
			expected: []string{"get_job", "get_run", "get_workspace"},
		},
		{
			name:     "with empty entries",
			input:    "get_run,,get_job",
			expected: []string{"get_job", "get_run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTools(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
