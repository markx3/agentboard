// Package agent orchestrates Claude Code agent lifecycle via tmux.
package agent

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
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

// Spawn launches a Claude Code agent in a tmux window for the given task.
// It checks that `claude` is in PATH, ensures the tmux session exists,
// launches the agent, and updates the task status to active.
func Spawn(ctx context.Context, svc board.Service, task db.Task) error {
	// Check claude is available
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("claude CLI not found in PATH")
	}

	// Ensure tmux session
	if err := tmux.EnsureSession(); err != nil {
		return fmt.Errorf("tmux: %w", err)
	}

	slug := TaskSlug(task.Title)
	winName := WindowName(task)
	sysPrompt := buildSystemPrompt(task)
	initialPrompt := buildInitialPrompt(task)

	// Kill any existing window for this task (handles respawn case)
	_ = tmux.KillWindow(winName)

	// Build the claude command with both system context and an initial user message
	cmd := fmt.Sprintf("claude -w %s --append-system-prompt %s %s",
		shellQuote(slug),
		shellQuote(sysPrompt),
		shellQuote(initialPrompt),
	)

	if err := tmux.NewWindow(winName, "", cmd); err != nil {
		return fmt.Errorf("creating tmux window: %w", err)
	}

	// Update task in DB
	task.AgentName = "claude"
	task.AgentStatus = db.AgentActive
	if err := svc.UpdateTask(ctx, &task); err != nil {
		// Best-effort kill the window if DB update fails
		_ = tmux.KillWindow(winName)
		return fmt.Errorf("updating task: %w", err)
	}

	return nil
}

// Kill terminates a running agent by killing its tmux window and updating the task.
func Kill(ctx context.Context, svc board.Service, task db.Task) error {
	winName := WindowName(task)

	// Best-effort kill
	_ = tmux.KillWindow(winName)

	task.AgentStatus = db.AgentIdle
	task.AgentName = ""
	if err := svc.UpdateTask(ctx, &task); err != nil {
		return fmt.Errorf("updating task: %w", err)
	}

	return nil
}

func buildSystemPrompt(task db.Task) string {
	shortID := task.ID[:8]

	var b strings.Builder
	b.WriteString("You are working on an agentboard task.\n")
	fmt.Fprintf(&b, "Task: %s  |  ID: %s\n", task.Title, shortID)
	if task.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", task.Description)
	}
	b.WriteString("\n")

	switch task.Status {
	case db.StatusBacklog:
		b.WriteString("STAGE: Backlog — Project Ideation\n")
		b.WriteString("When brainstorming is complete, move to planning:\n")
		fmt.Fprintf(&b, "  agentboard task move %s planning\n", shortID)
	case db.StatusPlanning:
		b.WriteString("STAGE: Planning — Implementation Design\n")
		b.WriteString("When the plan is ready, move to in progress:\n")
		fmt.Fprintf(&b, "  agentboard task move %s in_progress\n", shortID)
	case db.StatusInProgress:
		b.WriteString("STAGE: In Progress — Implementation\n")
		b.WriteString("When implementation is complete and a PR is opened, move to done:\n")
		fmt.Fprintf(&b, "  agentboard task move %s done\n", shortID)
	case db.StatusDone:
		b.WriteString("STAGE: Done — Verification & Cleanup\n")
		b.WriteString("Verify that the pull request has been opened and merged to main.\n")
		b.WriteString("Then clean up the git worktree for this task.\n")
	default:
		b.WriteString("When you are done, move the task to the next column using the agentboard CLI:\n")
		fmt.Fprintf(&b, "  agentboard task move %s <status>\n", shortID)
	}

	return b.String()
}

func buildInitialPrompt(task db.Task) string {
	switch task.Status {
	case db.StatusBacklog:
		return "Run /workflows:brainstorm to explore ideas for this task."
	case db.StatusPlanning:
		return "Run /workflows:plan to create a detailed implementation plan for this task."
	case db.StatusInProgress:
		return "Run /workflows:work to implement this task based on the plan."
	case db.StatusDone:
		return "Verify the pull request is merged and clean up the git worktree."
	default:
		return "Begin working on this task."
	}
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
