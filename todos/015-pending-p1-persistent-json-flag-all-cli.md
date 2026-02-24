---
status: pending
priority: p1
issue_id: "015"
tags: [code-review, agent-native, cli]
dependencies: []
---

# Make --json a Persistent Flag and Retrofit on All Commands

## Problem Statement

Only `task list` and `task get` support `--json` output. The 5 existing mutation commands (`task create`, `task move`, `task delete`, `task claim`, `task unclaim`) output human-readable text that agents must regex-parse. The plan adds `--json` to new commands but doesn't retrofit existing ones.

## Findings

- **Source**: agent-native-reviewer
- **Location**: `internal/cli/task_cmd.go` -- `taskOutputJSON` flag only registered on `taskListCmd` (line 82) and `taskGetCmd` (line 88)
- **Evidence**: `task create --json` is silently ignored because the flag isn't registered on `taskCreateCmd`
- **Impact**: Enrichment agents calling `task create` or `task move` can't parse structured output

## Proposed Solutions

### Option A: Persistent flag on root task command (Recommended, Effort: Small, Risk: Low)
Register `--json` as a persistent flag on `taskCmd` so all subcommands inherit it.
```go
taskCmd.PersistentFlags().BoolVar(&taskOutputJSON, "json", false, "Output as JSON")
```
Then add JSON output to each existing mutation command (return affected task object).
- **Pros**: Single registration, all commands get it, future commands inherit automatically
- **Cons**: Must update 5 existing commands' output paths

## Recommended Action

Option A. This is a foundational agent-native requirement.

## Acceptance Criteria

- [ ] `--json` is a persistent flag on `taskCmd`
- [ ] `task create --json` returns created task as JSON
- [ ] `task move --json` returns moved task as JSON
- [ ] `task delete --json` returns `{"deleted": true, "task_id": "..."}`
- [ ] `task claim --json` and `task unclaim --json` return updated task
- [ ] `agent request-reset --json` returns updated task
