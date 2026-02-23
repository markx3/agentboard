package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type ProjectConfig struct {
	Project  ProjectInfo     `toml:"project"`
	Agent    AgentConfig     `toml:"agent"`
	Worktree WorktreeConfig  `toml:"worktree"`
}

type ProjectInfo struct {
	Name string `toml:"name"`
}

type WorktreeConfig struct {
	CopyFiles  []string `toml:"copy_files"`
	InitScript string   `toml:"init_script"`
}

func LoadProject() (*ProjectConfig, error) {
	path := filepath.Join(".agentboard", "config.toml")
	cfg := defaultProject()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultProject() *ProjectConfig {
	return &ProjectConfig{
		Agent: AgentConfig{Preferred: "claude"},
		Worktree: WorktreeConfig{
			CopyFiles: []string{".env", ".env.local"},
		},
	}
}
