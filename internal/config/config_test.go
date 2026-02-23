package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/marcosfelipeeipper/agentboard/internal/config"
)

func TestLoadGlobalDefaults(t *testing.T) {
	cfg, err := config.LoadGlobal()
	if err != nil {
		t.Fatalf("loading global config: %v", err)
	}

	if cfg.Agent.Default != "claude" {
		t.Errorf("default agent: got %q, want %q", cfg.Agent.Default, "claude")
	}
	if cfg.Theme.Border != "#4a9a8a" {
		t.Errorf("border color: got %q, want %q", cfg.Theme.Border, "#4a9a8a")
	}
}

func TestLoadProjectDefaults(t *testing.T) {
	cfg, err := config.LoadProject()
	if err != nil {
		t.Fatalf("loading project config: %v", err)
	}

	if cfg.Agent.Preferred != "claude" {
		t.Errorf("preferred agent: got %q, want %q", cfg.Agent.Preferred, "claude")
	}
}

func TestMergedConfig(t *testing.T) {
	// Create a temp project config
	dir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	os.MkdirAll(".agentboard", 0o755)
	os.WriteFile(filepath.Join(".agentboard", "config.toml"), []byte(`
[project]
name = "test-project"

[agent]
preferred = "aider"
`), 0o644)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading merged config: %v", err)
	}

	if cfg.ProjectName != "test-project" {
		t.Errorf("project name: got %q, want %q", cfg.ProjectName, "test-project")
	}
	if cfg.AgentDefault != "aider" {
		t.Errorf("agent: got %q, want %q", cfg.AgentDefault, "aider")
	}
}
