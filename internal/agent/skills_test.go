package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSkillFile(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectName  string
		expectDesc  string
		expectBody  string
		expectError bool
	}{
		{
			name: "valid skill file",
			input: `---
name: test-skill
description: A test skill
---
Skill body content here.`,
			expectName: "test-skill",
			expectDesc: "A test skill",
			expectBody: "Skill body content here.",
		},
		{
			name:        "missing frontmatter",
			input:       "no frontmatter here",
			expectError: true,
		},
		{
			name:        "missing closing delimiter",
			input:       "---\nname: test\n",
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			s, err := parseSkillFile([]byte(test.input))

			if test.expectError {
				assert.NotNil(t, err)
				return
			}

			require.Nil(t, err)
			assert.Equal(t, test.expectName, s.Name)
			assert.Equal(t, test.expectDesc, s.Description)
			assert.Equal(t, test.expectBody, s.Data)
		})
	}
}

func TestLoadSkills(t *testing.T) {
	skills, err := loadSkills()
	require.Nil(t, err)
	assert.NotEmpty(t, skills)

	// Verify each skill has required fields
	for _, s := range skills {
		assert.NotEmpty(t, s.Name, "skill should have a name")
		assert.NotEmpty(t, s.Description, "skill should have a description")
		assert.NotEmpty(t, s.Data, "skill should have body content")
	}
}

func TestParseSkillFile_EmptyBody(t *testing.T) {
	input := "---\nname: empty\ndescription: empty body\n---\n"
	s, err := parseSkillFile([]byte(input))
	require.Nil(t, err)
	assert.Equal(t, "empty", s.Name)
	assert.Equal(t, "", s.Data)
}
