---
status: pending
priority: p1
issue_id: "013"
tags: [code-review, agent-native, cli]
dependencies: []
---

# Add `agent start` and `agent kill` CLI Commands

## Problem Statement

The TUI has `a` (spawn agent) and `A` (kill agent) keyboard shortcuts, but there are no CLI equivalents. This is the largest agent-native gap in the codebase. Without these commands, no agent can programmatically start or stop other agents, and the Phase 5 orchestrator has no foundation.

## Findings

- **Source**: agent-native-reviewer
- **Location**: `internal/cli/agent_cmd.go` (only has `request-reset`)
- **Evidence**: TUI spawn at `internal/tui/app.go:361-387`, kill at `app.go:391-410` -- no CLI equivalent exists
- **Agent-Native Score**: 5 of 21 capabilities are completely inaccessible to agents

## Proposed Solutions

### Option A: Add both commands now (Recommended, Effort: Medium, Risk: Low)
```
agentboard agent start <task-id> [--runner claude|cursor] [--skip-permissions] [--json]
agentboard agent kill <task-id> [--json]
```
- **Pros**: Complete agent-native parity for agent lifecycle
- **Cons**: Adds ~80 LOC

### Option B: Defer to Phase 5
- **Pros**: Less scope in Phase 1
- **Cons**: Breaks the principle that every TUI action has a CLI equivalent

## Recommended Action

Option A -- these are prerequisites for the enrichment workflow and the north star autonomous orchestration.

## Acceptance Criteria

- [ ] `agent start <task-id> --runner claude --json` spawns agent and returns task JSON
- [ ] `agent kill <task-id> --json` kills agent and returns task JSON
- [ ] `--skip-permissions` flag available on `agent start`
- [ ] Both commands work without TUI running
