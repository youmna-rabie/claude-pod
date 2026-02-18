package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/youmna-rabie/claude-pod/internal/config"
	"github.com/youmna-rabie/claude-pod/internal/skill"
)

func init() {
	rootCmd.AddCommand(listSkillsCmd)
}

var listSkillsCmd = &cobra.Command{
	Use:   "list-skills",
	Short: "Print registered skills",
	RunE:  listSkills,
}

func listSkills(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	reg := &skill.Registry{}
	if len(cfg.Skills.Dirs) > 0 {
		if err := reg.Scan(cfg.Skills.Dirs); err != nil {
			return fmt.Errorf("scanning skills: %w", err)
		}
	}

	skills := reg.Filter(cfg.Skills.Allowlist)

	if len(skills) == 0 {
		fmt.Println("No skills registered.")
		return nil
	}

	fmt.Printf("%-20s %-40s %s\n", "NAME", "DESCRIPTION", "PATH")
	for _, s := range skills {
		fmt.Printf("%-20s %-40s %s\n", s.Name, s.Description, s.Path)
	}
	return nil
}
