---
status: pending
priority: p3
issue_id: "022"
tags: [code-review, agent-native]
dependencies: []
---

# Add --search Flag to task list for CLI/TUI Parity

## Problem Statement

TUI search matches title/description/assignee via substring. CLI only has `--status` and `--assignee` exact-match filters. Agents cannot do free-text search from CLI.

**Location:** `internal/cli/task_cmd.go:runTaskList`

## Acceptance Criteria

- [ ] `agentboard task list --search "query"` filters by title/description/assignee substring
