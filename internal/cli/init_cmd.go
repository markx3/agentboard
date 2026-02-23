package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project configuration",
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := ".agentboard"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s directory: %w", dir, err)
	}

	configPath := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf("%s already exists, skipping\n", configPath)
		return nil
	}

	defaultConfig := `[project]
name = ""

# [agent]
# preferred = "claude"  # Reserved for future use

[worktree]
copy_files = [".env", ".env.local"]
init_script = ""
`
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	// Add server.json to gitignore
	gitignorePath := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(gitignorePath, []byte("server.json\nworktrees/\n"), 0o644); err != nil {
		return fmt.Errorf("writing gitignore: %w", err)
	}

	fmt.Printf("Initialized agentboard in %s/\n", dir)
	return nil
}
