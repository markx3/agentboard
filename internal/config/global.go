package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type GlobalConfig struct {
	User  UserConfig  `toml:"user"`
	Agent AgentConfig `toml:"agent"`
	Theme ThemeConfig `toml:"theme"`
}

type UserConfig struct {
	GitHubUsername string `toml:"github_username"`
}

type AgentConfig struct {
	Default   string `toml:"default"`
	Preferred string `toml:"preferred"`
}

type ThemeConfig struct {
	Border string `toml:"border"`
	Text   string `toml:"text"`
	Accent string `toml:"accent"`
}

func LoadGlobal() (*GlobalConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultGlobal(), nil
	}

	path := filepath.Join(home, ".agentboard", "config.toml")
	cfg := defaultGlobal()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultGlobal() *GlobalConfig {
	return &GlobalConfig{
		Agent: AgentConfig{Default: "claude"},
		Theme: ThemeConfig{
			Border: "#4a9a8a",
			Text:   "#d4d4d4",
			Accent: "#e6b450",
		},
	}
}
