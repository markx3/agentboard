package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/markx3/agentboard/internal/db"
)

func TestClaudeRunnerBuildCommand(t *testing.T) {
	runner := &ClaudeRunner{}

	if runner.ID() != "claude" {
		t.Errorf("ID() = %q, want %q", runner.ID(), "claude")
	}
	if runner.Name() != "Claude Code" {
		t.Errorf("Name() = %q, want %q", runner.Name(), "Claude Code")
	}
	if runner.Binary() != "claude" {
		t.Errorf("Binary() = %q, want %q", runner.Binary(), "claude")
	}

	task := db.Task{
		ID:          "abcdef1234567890",
		Title:       "Test Task",
		Description: "A test description",
		Status:      db.StatusInProgress,
	}

	opts := SpawnOpts{
		WorkDir: "test-task",
		Task:    task,
	}

	cmd := runner.BuildCommand(opts)

	// Should start with claude -w
	if !strings.HasPrefix(cmd, "claude -w ") {
		t.Errorf("BuildCommand should start with 'claude -w', got: %s", cmd)
	}

	// Should contain --append-system-prompt
	if !strings.Contains(cmd, "--append-system-prompt") {
		t.Error("BuildCommand should contain --append-system-prompt")
	}

	// Should contain the work dir
	if !strings.Contains(cmd, "test-task") {
		t.Error("BuildCommand should contain work dir")
	}

	// Should reference the task ID
	if !strings.Contains(cmd, "abcdef12") {
		t.Error("BuildCommand should contain task short ID")
	}

	// Should contain stage-appropriate guidance
	if !strings.Contains(cmd, "In Progress") {
		t.Error("BuildCommand system prompt should contain stage info")
	}

	// Should contain workflow command
	if !strings.Contains(cmd, "/workflows:work") {
		t.Error("BuildCommand initial prompt should contain /workflows:work for in_progress stage")
	}
}

func TestCursorRunnerBuildCommand(t *testing.T) {
	runner := &CursorRunner{}

	if runner.ID() != "cursor" {
		t.Errorf("ID() = %q, want %q", runner.ID(), "cursor")
	}
	if runner.Name() != "Cursor" {
		t.Errorf("Name() = %q, want %q", runner.Name(), "Cursor")
	}
	if runner.Binary() != "agent" {
		t.Errorf("Binary() = %q, want %q", runner.Binary(), "agent")
	}

	task := db.Task{
		ID:          "abcdef1234567890",
		Title:       "Test Task",
		Description: "A test description",
		Status:      db.StatusPlanning,
	}

	opts := SpawnOpts{
		WorkDir: "test-task",
		Task:    task,
	}

	cmd := runner.BuildCommand(opts)

	// Should start with agent
	if !strings.HasPrefix(cmd, "agent ") {
		t.Errorf("BuildCommand should start with 'agent', got: %s", cmd)
	}

	// Should contain the task title
	if !strings.Contains(cmd, "Test Task") {
		t.Error("BuildCommand should contain task title")
	}

	// Should reference the task ID
	if !strings.Contains(cmd, "abcdef12") {
		t.Error("BuildCommand should contain task short ID")
	}

	// Should contain stage-appropriate guidance
	if !strings.Contains(cmd, "Planning") {
		t.Error("BuildCommand prompt should contain stage info")
	}

	// Cursor prompt should use plain language, not /workflows commands
	if strings.Contains(cmd, "/workflows:") {
		t.Error("Cursor BuildCommand should not contain /workflows: commands")
	}

	// Should contain agentboard move command
	if !strings.Contains(cmd, "agentboard task move") {
		t.Error("BuildCommand should contain agentboard task move command")
	}
}

func TestClaudeRunnerStagePrompts(t *testing.T) {
	runner := &ClaudeRunner{}

	tests := []struct {
		status       db.TaskStatus
		wantSysStage string
		wantInitial  string
	}{
		{db.StatusBacklog, "Backlog", "Move it to brainstorm"},
		{db.StatusBrainstorm, "Brainstorm", "/workflows:brainstorm"},
		{db.StatusPlanning, "Planning", "/workflows:plan"},
		{db.StatusInProgress, "In Progress", "/workflows:work"},
		{db.StatusDone, "Done", "Verify the pull request"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			task := db.Task{
				ID:     "abcdef1234567890",
				Title:  "Test",
				Status: tt.status,
			}
			opts := SpawnOpts{WorkDir: "test", Task: task}
			cmd := runner.BuildCommand(opts)

			if !strings.Contains(cmd, tt.wantSysStage) {
				t.Errorf("stage %s: command should contain %q", tt.status, tt.wantSysStage)
			}
			if !strings.Contains(cmd, tt.wantInitial) {
				t.Errorf("stage %s: command should contain %q", tt.status, tt.wantInitial)
			}
		})
	}
}

func TestClaudeRunnerBuildEnrichmentCommand(t *testing.T) {
	runner := &ClaudeRunner{}

	task := db.Task{
		ID:          "abcdef1234567890",
		Title:       "Test Task",
		Description: "A test description",
		Status:      db.StatusInProgress,
	}

	opts := SpawnOpts{
		WorkDir: ".",
		Task:    task,
	}

	cmd := runner.BuildEnrichmentCommand(opts)

	// Should not be empty (Claude supports enrichment)
	if cmd == "" {
		t.Fatal("BuildEnrichmentCommand returned empty string")
	}

	// Should use --print for non-interactive one-shot execution
	if !strings.Contains(cmd, "--print") {
		t.Error("enrichment command should use --print for non-interactive execution")
	}

	// Should use --dangerously-skip-permissions for autonomous operation
	if !strings.Contains(cmd, "--dangerously-skip-permissions") {
		t.Error("enrichment command should use --dangerously-skip-permissions")
	}

	// Should reference the task ID
	if !strings.Contains(cmd, "abcdef12") {
		t.Error("enrichment command should contain task short ID")
	}

	// Should contain enrichment instructions
	if !strings.Contains(cmd, "enrich") || !strings.Contains(cmd, "task") {
		t.Error("enrichment command should contain enrichment instructions")
	}
}

func TestCursorRunnerBuildEnrichmentCommand(t *testing.T) {
	runner := &CursorRunner{}

	task := db.Task{
		ID:    "abcdef1234567890",
		Title: "Test Task",
	}

	opts := SpawnOpts{
		WorkDir: ".",
		Task:    task,
	}

	cmd := runner.BuildEnrichmentCommand(opts)

	// Cursor does not support enrichment
	if cmd != "" {
		t.Errorf("CursorRunner.BuildEnrichmentCommand should return empty, got %q", cmd)
	}
}

func TestEnrichmentWindowName(t *testing.T) {
	task := db.Task{ID: "abcdef1234567890"}
	name := EnrichmentWindowName(task)
	if name != "enrich-abcdef12" {
		t.Errorf("EnrichmentWindowName: got %q, want %q", name, "enrich-abcdef12")
	}
}

func TestGetRunner(t *testing.T) {
	claude := GetRunner("claude")
	if claude == nil {
		t.Fatal("GetRunner(\"claude\") returned nil")
	}
	if claude.ID() != "claude" {
		t.Errorf("GetRunner(\"claude\").ID() = %q", claude.ID())
	}

	cursor := GetRunner("cursor")
	if cursor == nil {
		t.Fatal("GetRunner(\"cursor\") returned nil")
	}
	if cursor.ID() != "cursor" {
		t.Errorf("GetRunner(\"cursor\").ID() = %q", cursor.ID())
	}

	unknown := GetRunner("unknown")
	if unknown != nil {
		t.Errorf("GetRunner(\"unknown\") should return nil, got %v", unknown)
	}
}

func TestTaskSlug(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
		{"special!@#chars", "special-chars"},
		{"", "task"},
	}

	for _, tt := range tests {
		got := TaskSlug(tt.input)
		if got != tt.want {
			t.Errorf("TaskSlug(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"with space", "'with space'"},
		{"it's", "'it'\"'\"'s'"},
	}

	for _, tt := range tests {
		got := shellQuote(tt.input)
		if got != tt.want {
			t.Errorf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDeactivateRalphLoop(t *testing.T) {
	t.Run("file exists with active true", func(t *testing.T) {
		dir := t.TempDir()
		// DeactivateRalphLoop uses TaskSlug(task.Title) as base dir,
		// so we need to chdir to the temp dir and use a matching task title.
		origDir, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(origDir)

		slug := "test-task"
		stateDir := filepath.Join(slug, ".claude")
		os.MkdirAll(stateDir, 0755)

		content := "---\nactive: true\niteration: 3\nmax_iterations: 10\n---\n"
		os.WriteFile(filepath.Join(stateDir, "ralph-loop.local.md"), []byte(content), 0644)

		task := db.Task{Title: "Test Task"}
		err := DeactivateRalphLoop(task)
		if err != nil {
			t.Fatalf("DeactivateRalphLoop() error = %v", err)
		}

		data, _ := os.ReadFile(filepath.Join(stateDir, "ralph-loop.local.md"))
		if strings.Contains(string(data), "active: true") {
			t.Error("expected active: true to be replaced with active: false")
		}
		if !strings.Contains(string(data), "active: false") {
			t.Error("expected file to contain active: false")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(origDir)

		task := db.Task{Title: "Nonexistent Task"}
		err := DeactivateRalphLoop(task)
		if err != nil {
			t.Fatalf("DeactivateRalphLoop() should return nil for missing file, got %v", err)
		}
	})

	t.Run("file exists with active false", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		os.Chdir(dir)
		defer os.Chdir(origDir)

		slug := "already-inactive"
		stateDir := filepath.Join(slug, ".claude")
		os.MkdirAll(stateDir, 0755)

		content := "---\nactive: false\niteration: 5\n---\n"
		os.WriteFile(filepath.Join(stateDir, "ralph-loop.local.md"), []byte(content), 0644)

		task := db.Task{Title: "Already Inactive"}
		err := DeactivateRalphLoop(task)
		if err != nil {
			t.Fatalf("DeactivateRalphLoop() error = %v", err)
		}

		data, _ := os.ReadFile(filepath.Join(stateDir, "ralph-loop.local.md"))
		got := string(data)
		if got != content {
			t.Errorf("file content changed unexpectedly:\ngot:  %q\nwant: %q", got, content)
		}
	})
}
