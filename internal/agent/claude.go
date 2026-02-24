package agent

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

// ClaudeRunner implements AgentRunner for Claude Code CLI.
type ClaudeRunner struct{}

func (c *ClaudeRunner) ID() string     { return "claude" }
func (c *ClaudeRunner) Name() string   { return "Claude Code" }
func (c *ClaudeRunner) Binary() string { return "claude" }

func (c *ClaudeRunner) Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeRunner) BuildCommand(opts SpawnOpts) string {
	sysPrompt := buildClaudeSystemPrompt(opts)
	initialPrompt := buildClaudeInitialPrompt(opts)
	skipFlag := ""
	if opts.Task.SkipPermissions {
		skipFlag = "--dangerously-skip-permissions "
	}
	return fmt.Sprintf("claude %s-w %s --append-system-prompt %s %s",
		skipFlag,
		shellQuote(opts.WorkDir),
		shellQuote(sysPrompt),
		shellQuote(initialPrompt),
	)
}

func buildClaudeSystemPrompt(opts SpawnOpts) string {
	task := opts.Task
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
		b.WriteString("STAGE: Backlog — Unplanned\n")
		b.WriteString("Move to brainstorm to begin work:\n")
		fmt.Fprintf(&b, "  agentboard task move %s brainstorm\n", shortID)
	case db.StatusBrainstorm:
		b.WriteString("STAGE: Brainstorm — Exploring Ideas\n")
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

func buildClaudeInitialPrompt(opts SpawnOpts) string {
	switch opts.Task.Status {
	case db.StatusBacklog:
		return "This task is in backlog. Move it to brainstorm to begin work."
	case db.StatusBrainstorm:
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

func (c *ClaudeRunner) BuildEnrichmentCommand(opts SpawnOpts) string {
	sysPrompt := buildEnrichmentSystemPrompt(opts)
	initialPrompt := buildEnrichmentInitialPrompt(opts)
	return fmt.Sprintf("timeout 90 claude --dangerously-skip-permissions -w %s --append-system-prompt %s %s",
		shellQuote(opts.WorkDir),
		shellQuote(sysPrompt),
		shellQuote(initialPrompt),
	)
}

func buildEnrichmentSystemPrompt(opts SpawnOpts) string {
	task := opts.Task
	shortID := task.ID[:8]

	var b strings.Builder
	b.WriteString("You are an enrichment agent for agentboard.\n")
	b.WriteString("Your job is to enrich a newly created task with context from the codebase and board.\n")
	b.WriteString("You are short-lived -- gather context, update the task, and exit.\n\n")
	fmt.Fprintf(&b, "Task to enrich: %s (ID: %s)\n", task.Title, shortID)
	if task.Description != "" {
		fmt.Fprintf(&b, "Current description: %s\n", task.Description)
	}
	b.WriteString("\nIMPORTANT: Do NOT move the task, spawn agents, or do implementation work.\n")
	b.WriteString("Only read, analyze, and update the task description/dependencies/comments.\n")
	fmt.Fprintf(&b, "\nCLI commands available:\n")
	fmt.Fprintf(&b, "  agentboard task list --json          # See all tasks\n")
	fmt.Fprintf(&b, "  agentboard task get <id> --json      # Get task details with deps/comments\n")
	fmt.Fprintf(&b, "  agentboard status --json             # Board summary\n")
	fmt.Fprintf(&b, "  agentboard task update %s --description \"...\" --add-dep <dep-id>\n", shortID)
	fmt.Fprintf(&b, "  agentboard task comment %s --author enrichment --body \"...\"\n", shortID)
	b.WriteString("\nRetry with jitter if you get database-locked errors:\n")
	b.WriteString("  for i in 1 2 3; do agentboard task update ... && break || sleep $((RANDOM %% 3 + 1)); done\n")

	return b.String()
}

func buildEnrichmentInitialPrompt(opts SpawnOpts) string {
	task := opts.Task
	shortID := task.ID[:8]

	var b strings.Builder
	b.WriteString("Enrich this task by:\n\n")
	b.WriteString("1. Run `agentboard task list --json` to see all tasks and their statuses\n")
	b.WriteString("2. Run `agentboard status --json` to see the board summary\n")
	b.WriteString("3. Scan `git log --oneline -20`, `git branch -a`, `git status`\n")
	b.WriteString("4. Read relevant files in the codebase that relate to the task title\n")
	b.WriteString("5. Identify dependencies with existing in-flight tasks\n")
	fmt.Fprintf(&b, "6. Update the task: `agentboard task update %s --description \"enriched description\"`\n", shortID)
	fmt.Fprintf(&b, "7. Add dependencies: `agentboard task update %s --add-dep <dep-id>`\n", shortID)
	fmt.Fprintf(&b, "8. Leave a comment: `agentboard task comment %s --author enrichment --body \"analysis summary\"`\n", shortID)
	b.WriteString("9. Exit when done\n\n")
	b.WriteString("Be concise. Focus on actionable context: what this task depends on, what's related, and any codebase insights.\n")

	return b.String()
}
