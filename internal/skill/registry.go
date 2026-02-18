package skill

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/youmna-rabie/claude-pod/internal/types"
	"gopkg.in/yaml.v3"
)

// Registry discovers and manages skills by scanning for SKILL.md files.
type Registry struct {
	skills []types.Skill
}

// frontmatter holds the YAML fields parsed from SKILL.md front matter.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Scan walks each directory in dirs looking for SKILL.md files.
// It parses YAML frontmatter (delimited by "---") from each file
// to extract the skill name and description.
func (r *Registry) Scan(dirs []string) error {
	r.skills = nil

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip inaccessible paths
			}
			if d.IsDir() || d.Name() != "SKILL.md" {
				return nil
			}

			sk, err := parseSkillFile(path)
			if err != nil {
				return nil // skip malformed files
			}

			r.skills = append(r.skills, sk)
			return nil
		})
		if err != nil {
			return fmt.Errorf("scanning %s: %w", dir, err)
		}
	}

	return nil
}

// Skills returns all discovered skills.
func (r *Registry) Skills() []types.Skill {
	return r.skills
}

// Filter returns only the skills whose names appear in the allowlist.
// An empty allowlist returns all skills (no filtering).
func (r *Registry) Filter(allowlist []string) []types.Skill {
	if len(allowlist) == 0 {
		return r.skills
	}

	allowed := make(map[string]struct{}, len(allowlist))
	for _, name := range allowlist {
		allowed[name] = struct{}{}
	}

	var filtered []types.Skill
	for _, sk := range r.skills {
		if _, ok := allowed[sk.Name]; ok {
			filtered = append(filtered, sk)
		}
	}
	return filtered
}

// parseSkillFile reads a SKILL.md file and extracts name and description
// from YAML frontmatter delimited by "---" lines.
func parseSkillFile(path string) (types.Skill, error) {
	f, err := os.Open(path)
	if err != nil {
		return types.Skill{}, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// First line must be "---"
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return types.Skill{}, fmt.Errorf("%s: missing opening frontmatter delimiter", path)
	}

	// Collect lines until closing "---"
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return types.Skill{}, fmt.Errorf("%s: empty frontmatter", path)
	}

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines, "\n")), &fm); err != nil {
		return types.Skill{}, fmt.Errorf("%s: parsing frontmatter: %w", path, err)
	}

	if fm.Name == "" {
		return types.Skill{}, fmt.Errorf("%s: frontmatter missing name", path)
	}

	return types.Skill{
		Name:        fm.Name,
		Description: fm.Description,
		Path:        path,
	}, nil
}
