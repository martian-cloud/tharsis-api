package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/m-mizutani/gollem"
)

const loadSkillToolName = "load_skill"

type skillsTool struct {
	skills []skill
}

func newSkillsTool(skills []skill) gollem.Tool {
	return &skillsTool{skills: skills}
}

func (t *skillsTool) Spec() gollem.ToolSpec {
	var lines []string
	for _, s := range t.skills {
		lines = append(lines, fmt.Sprintf("• %s — %s", s.Name, s.Description))
	}

	return gollem.ToolSpec{
		Name: loadSkillToolName,
		Description: fmt.Sprintf(
			"Load skill instructions to help accomplish the user's request. "+
				"Evaluate the user query against the available skills and call this tool with any matching skill name before responding.\n\n"+
				"IMPORTANT: If a matching skill is found, make sure you call this tool before calling any other tools.\n\n"+
				"Available skills:\n%s", strings.Join(lines, "\n")),
		Parameters: map[string]*gollem.Parameter{
			"name": {
				Type:        gollem.TypeString,
				Description: "The name of the skill to load",
				Required:    true,
			},
		},
	}
}

func (t *skillsTool) Run(_ context.Context, args map[string]any) (map[string]any, error) {
	name, _ := args["name"].(string)
	for _, s := range t.skills {
		if strings.EqualFold(s.Name, name) {
			return map[string]any{"instructions": s.Data}, nil
		}
	}
	return nil, fmt.Errorf("skill %q not found", name)
}
