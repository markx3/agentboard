// Package agent orchestrates AI agent lifecycle via tmux.
package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
	"github.com/marcosfelipeeipper/agentboard/internal/tmux"
)

const maxSlugLen = 50

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

// TaskSlug converts a task title to a filesystem-safe slug (max 50 chars).
func TaskSlug(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	s = slugUnsafe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if utf8.RuneCountInString(s) > maxSlugLen {
		s = s[:maxSlugLen]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		s = "task"
	}
	return s
}

// WindowName returns the tmux window name used for polling a task's agent.
func WindowName(task db.Task) string {
	return "agent-" + task.ID[:8]
}

// DeactivateRalphLoop sets active: false in the ralph-loop state file
// for the given task's worktree. This prevents a respawned agent from
// inheriting an active loop. No-op if the file does not exist.
func DeactivateRalphLoop(task db.Task) error {
	stateFile := filepath.Join(TaskSlug(task.Title), ".claude", "ralph-loop.local.md")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil // No state file = no ralph loop to deactivate
	}
	updated := strings.Replace(string(data), "active: true", "active: false", 1)
	return os.WriteFile(stateFile, []byte(updated), 0644)
}

// Spawn launches an AI agent in a tmux window for the given task.
// The runner determines which CLI is used and how the command is built.
func Spawn(ctx context.Context, svc board.Service, task db.Task, runner AgentRunner) error {
	// Ensure tmux session
	if err := tmux.EnsureSession(); err != nil {
		return fmt.Errorf("tmux: %w", err)
	}

	slug := TaskSlug(task.Title)
	winName := WindowName(task)

	opts := SpawnOpts{
		WorkDir: slug,
		Task:    task,
	}

	// Kill any existing window for this task (handles respawn case)
	_ = tmux.KillWindow(winName)

	cmd := runner.BuildCommand(opts)

	// For Claude, working dir is passed as -w flag in the command itself.
	// For other agents, set CWD via tmux's -c flag so the process inherits it.
	windowDir := ""
	if runner.ID() != "claude" {
		windowDir = slug
	}

	if err := tmux.NewWindow(winName, windowDir, cmd); err != nil {
		return fmt.Errorf("creating tmux window: %w", err)
	}

	// Update task in DB
	task.AgentName = runner.ID()
	task.AgentStatus = db.AgentActive
	task.AgentSpawnedStatus = string(task.Status)
	task.AgentStartedAt = time.Now().UTC().Format(time.RFC3339)
	if err := svc.UpdateTask(ctx, &task); err != nil {
		// Best-effort kill the window if DB update fails
		_ = tmux.KillWindow(winName)
		return fmt.Errorf("updating task: %w", err)
	}

	return nil
}

// EnrichmentWindowName returns the tmux window name for an enrichment agent.
func EnrichmentWindowName(task db.Task) string {
	return "enrich-" + task.ID[:8]
}

// SpawnEnrichment launches a short-lived enrichment agent in a tmux window.
// Unlike Spawn, it does NOT set the task's AgentStatus (preserving work agent state).
// Instead, it sets EnrichmentStatus and EnrichmentAgentName via partial update.
func SpawnEnrichment(ctx context.Context, svc board.Service, task db.Task, runner AgentRunner) error {
	cmd := runner.BuildEnrichmentCommand(SpawnOpts{
		WorkDir: ".",
		Task:    task,
	})
	if cmd == "" {
		return fmt.Errorf("runner %s does not support enrichment", runner.ID())
	}

	if err := tmux.EnsureSession(); err != nil {
		return fmt.Errorf("tmux: %w", err)
	}

	winName := EnrichmentWindowName(task)

	// Kill any existing enrichment window for this task
	_ = tmux.KillWindow(winName)

	if err := tmux.NewWindow(winName, ".", cmd); err != nil {
		return fmt.Errorf("creating enrichment window: %w", err)
	}

	// Use UpdateTaskFields (NOT full UpdateTask) to avoid overwriting
	// concurrent edits to title/description/status by humans or work agents.
	enriching := db.EnrichmentEnriching
	runnerID := runner.ID()
	if err := svc.UpdateTaskFields(ctx, task.ID, db.TaskFieldUpdate{
		EnrichmentStatus:    &enriching,
		EnrichmentAgentName: &runnerID,
	}); err != nil {
		_ = tmux.KillWindow(winName)
		return fmt.Errorf("updating enrichment status: %w", err)
	}

	return nil
}

// Kill terminates a running agent by killing its tmux window and updating the task.
// AgentName is preserved so the task remembers which agent was used.
func Kill(ctx context.Context, svc board.Service, task db.Task) error {
	winName := WindowName(task)

	// Best-effort kill
	_ = tmux.KillWindow(winName)

	task.AgentStatus = db.AgentIdle
	if err := svc.UpdateTask(ctx, &task); err != nil {
		return fmt.Errorf("updating task: %w", err)
	}

	return nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
