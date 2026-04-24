package agent

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

//go:embed skills/*.md
var skillsFS embed.FS

type skill struct {
	Name        string
	Description string
	Data        string
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

func loadSkills() ([]skill, error) {
	var skills []skill

	entries, err := fs.ReadDir(skillsFS, "skills")
	if err != nil {
		return nil, fmt.Errorf("failed to read skills directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		data, err := skillsFS.ReadFile("skills/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("failed to read skill file %s: %w", entry.Name(), err)
		}

		s, err := parseSkillFile(data)
		if err != nil {
			return nil, fmt.Errorf("failed to parse skill file %s: %w", entry.Name(), err)
		}

		skills = append(skills, s)
	}

	return skills, nil
}

func parseSkillFile(data []byte) (skill, error) {
	const sep = "---"

	content := bytes.TrimSpace(data)
	if !bytes.HasPrefix(content, []byte(sep)) {
		return skill{}, fmt.Errorf("missing frontmatter")
	}

	rest := content[len(sep):]
	idx := bytes.Index(rest, []byte("\n"+sep))
	if idx < 0 {
		return skill{}, fmt.Errorf("missing closing frontmatter delimiter")
	}

	var fm skillFrontmatter
	if err := yaml.Unmarshal(rest[:idx], &fm); err != nil {
		return skill{}, fmt.Errorf("invalid frontmatter YAML: %w", err)
	}

	body := bytes.TrimSpace(rest[idx+len("\n"+sep):])

	return skill{
		Name:        fm.Name,
		Description: fm.Description,
		Data:        string(body),
	}, nil
}
