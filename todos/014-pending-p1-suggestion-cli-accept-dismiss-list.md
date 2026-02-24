---
status: pending
priority: p1
issue_id: "014"
tags: [code-review, agent-native, cli, suggestions]
dependencies: []
---

# Add Suggestion Accept/Dismiss/List CLI Commands

## Problem Statement

The plan creates a suggestion system where agents propose tasks, but accept/dismiss are TUI-only actions (keyboard shortcuts). There is also no way to list pending suggestions via CLI. This breaks the proposal workflow for programmatic consumers -- agents can propose but nothing can approve without a human in the TUI.

## Findings

- **Source**: agent-native-reviewer
- **Location**: Plan sections 3.2-3.4 (accept/dismiss are TUI-only)
- **Evidence**: No CLI command for: listing suggestions, accepting suggestions, dismissing suggestions
- **Impact**: The Phase 5 orchestrator agent cannot approve proposed tasks programmatically

## Proposed Solutions

### Option A: Add under `task` subcommand (Recommended, Effort: Small, Risk: Low)
```
agentboard task suggestions [--status pending|accepted|dismissed] [--json]
agentboard task suggestion accept <suggestion-id> [--json]
agentboard task suggestion dismiss <suggestion-id> [--json]
```
- **Pros**: Consistent with existing CLI structure
- **Cons**: Slightly longer command names

### Option B: Top-level `suggestion` command
- **Pros**: Shorter to type
- **Cons**: Adds complexity to root command

## Recommended Action

Option A -- keeps CLI structure consistent.

## Acceptance Criteria

- [ ] `task suggestions --json` lists all pending suggestions
- [ ] `task suggestion accept <id> --json` accepts and returns created task (for proposals)
- [ ] `task suggestion dismiss <id> --json` dismisses suggestion
- [ ] All commands work without TUI running
