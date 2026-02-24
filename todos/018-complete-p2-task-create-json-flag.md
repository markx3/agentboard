---
status: complete
priority: p2
issue_id: "018"
tags: [code-review, agent-native]
dependencies: []
---

# Register --json Flag on task create Command

## Problem Statement

`runTaskCreate` already checks `taskOutputJSON` and outputs JSON when set, but `taskCreateCmd` never registers the `--json` flag. An agent calling `task create --json` gets an unknown flag error, yet the code path exists and would work.

## Findings

- **Agent-Native Reviewer:** "The handler code already supports it."

**Location:** `internal/cli/task_cmd.go` init() function

## Proposed Solutions

### Solution A: Add flag registration (Recommended)
Add `taskCreateCmd.Flags().BoolVar(&taskOutputJSON, "json", false, "output as JSON")` in `init()`.

- **Effort:** Small (2 min)
- **Risk:** None

## Acceptance Criteria

- [ ] `agentboard task create --title "test" --json` outputs JSON

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by agent-native reviewer |
