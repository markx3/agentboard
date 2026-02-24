---
status: complete
priority: p1
issue_id: "012"
tags: [code-review, agent-native]
dependencies: []
---

# CLI `task list --json` and `task get --json` Omit Dependency Data

## Problem Statement

The TUI populates `BlockedBy` on each task by calling `GetAllDependencies()` in `loadTasks()`, but the CLI commands `task list --json` and `task get --json` never load dependency data. An agent that creates dependencies via `task block` has no way to verify them via the CLI. This is a Context Parity failure.

## Findings

- **Agent-Native Reviewer:** "The JSON output always has `blocked_by` as null/omitted. An agent that calls `task block` has no way to verify what it did."
- The TUI loads deps at `app.go:112-118`, but `task_cmd.go:runTaskList` and `runTaskGet` skip this.

**Location:** `internal/cli/task_cmd.go:126-170` (runTaskList), `230-266` (runTaskGet)

## Proposed Solutions

### Solution A: Populate BlockedBy in CLI commands (Recommended)
After fetching tasks in `runTaskList` and `runTaskGet`, call `svc.GetAllDependencies(ctx)` and populate `BlockedBy` fields, same as the TUI does.

- **Pros:** Full parity with TUI, agents can verify their work
- **Cons:** Adds 1 extra query per CLI invocation
- **Effort:** Small (15 min)
- **Risk:** Low

## Technical Details

- **Affected files:** `internal/cli/task_cmd.go`
- In `runTaskList`: after fetching tasks, call `svc.GetAllDependencies` and populate `task.BlockedBy`
- In `runTaskGet`: after fetching task, call `svc.GetBlockers` and set `task.BlockedBy`
- Also add "Blocked by:" to human-readable `task get` output

## Acceptance Criteria

- [ ] `agentboard task list --json` includes `blocked_by` array on blocked tasks
- [ ] `agentboard task get <id> --json` includes `blocked_by` array
- [ ] Human-readable `task get` shows "Blocked by: ..." line

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by agent-native reviewer |

## Resources

- PR: https://github.com/markx3/agentboard/pull/16
