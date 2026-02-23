---
title: "feat: Add Review column to agentboard kanban"
type: feat
date: 2026-02-23
---

# Add Review Column to Agentboard Kanban

## Overview

Add a 5th "Review" column to the agentboard kanban board, positioned between "In Progress" and "Done". Moving a task to Review (or spawning an agent from within Review) runs a `workflows:review` session. The database, model, and CLI validation already support the `review` status -- only the TUI, agent prompts, CLI status display, and tests need updating.

## Problem Statement / Motivation

The task lifecycle currently jumps from "In Progress" straight to "Done", with no dedicated code review step. The brainstorm (2026-02-23) designed a Review column where team members can launch review agents against a task's PR/branch. This feature closes that gap by wiring the existing `StatusReview` data model into the TUI and agent system.

## Proposed Solution

A surgical 4-file change that follows the exact same patterns used by the existing 4 columns:

1. Register the Review column in the TUI's `columnOrder` and `columnTitles`
2. Add review-specific agent prompts (system prompt + initial prompt)
3. Update the In Progress agent prompt to direct tasks to Review instead of Done
4. Add Review to the CLI status summary
5. Update test coverage

No database migration, protocol changes, or new files required.

## Technical Considerations

### Column Width Impact

With 5 columns, each gets `w/5` instead of `w/4`. On an 80-char terminal: 16 chars per column, ~12 usable after borders/padding. Task titles will truncate more aggressively. This is acceptable for the initial implementation -- narrow terminal layout improvements can be addressed separately.

### Behavioral Change: Done backward movement

Previously, pressing `M` on a Done task moved it to In Progress. With Review inserted, `M` now moves it to Review. This follows the natural column order and is the correct behavior -- Review sits between In Progress and Done. Users needing to go back to In Progress press `M` twice.

### Agent Prompt Context

The review system prompt includes `PRUrl` and `BranchName` from the task when available, giving the review agent the context it needs. When these are empty, the prompt degrades gracefully by instructing the agent to locate the relevant PR/branch.

### `/workflows:review` Dependency

The initial prompt tells the agent to run `/workflows:review`. This skill exists in the compound-engineering plugin (confirmed in the skill registry). If a user's Claude Code installation lacks this skill, the agent receives the instruction but may fall back to manual review.

## Acceptance Criteria

- [x] Review column appears in the TUI between "In Progress" and "Done"
- [x] `m` on a task in "In Progress" moves it to "Review"
- [x] `m` on a task in "Review" moves it to "Done"
- [x] `M` on a task in "Review" moves it back to "In Progress"
- [x] `M` on a task in "Done" moves it to "Review"
- [ ] Spawning an agent (`a`) on a Review task runs `/workflows:review` (deferred: agent system not yet built)
- [ ] Auto-respawn: active agent on task moved to Review restarts with review workflow (deferred: agent system not yet built)
- [ ] In Progress agent prompt now says "move to review" (not "move to done") (deferred: agent system not yet built)
- [x] `agentboard status` shows Review count (text and JSON)
- [x] `agentboard task move <id> review` continues to work (already does)
- [x] Tests cover full progression: Planning → In Progress → Review → Done
- [x] All existing tests pass

## Success Metrics

- All acceptance criteria met
- `go test ./...` passes
- Board renders correctly with 5 columns on a standard terminal (120+ chars)

## Dependencies & Risks

**No blockers.** The database schema (`CHECK(status IN (...,'review',...))`) and the Go model (`StatusReview`) already exist. The `Valid()` method already accepts `review`. The CLI `task move` command already works with `review`.

**Risk: Column width on narrow terminals.** Mitigated by accepting truncation for now. A future enhancement could add responsive column sizing or horizontal scrolling.

## Implementation Plan

### File 1: `internal/tui/board.go` (2 line changes)

Add `db.StatusReview` to `columnOrder` (between `StatusInProgress` and `StatusDone`, line 14):

```go
var columnOrder = []db.TaskStatus{
    db.StatusBacklog,
    db.StatusPlanning,
    db.StatusInProgress,
    db.StatusReview,     // NEW
    db.StatusDone,
}
```

Add `"Review"` title to `columnTitles` (line 22):

```go
var columnTitles = map[db.TaskStatus]string{
    db.StatusBacklog:    "Backlog",
    db.StatusPlanning:   "Planning",
    db.StatusInProgress: "In Progress",
    db.StatusReview:     "Review",     // NEW
    db.StatusDone:       "Done",
}
```

**Everything else (column rendering, navigation, width calculation, task loading, status bar) is driven by these two data structures and works automatically.**

### File 2: `internal/agent/spawn.go` (3 changes)

**Change A: Update In Progress system prompt** (line 122-125) to direct agents to Review:

```go
case db.StatusInProgress:
    b.WriteString("STAGE: In Progress — Implementation\n")
    b.WriteString("When implementation is complete and a PR is opened, move to review:\n")
    fmt.Fprintf(&b, "  agentboard task move %s review\n", shortID)
```

**Change B: Add Review case to `buildSystemPrompt`** (after the InProgress case):

```go
case db.StatusReview:
    b.WriteString("STAGE: Review — Code Review & Validation\n")
    if task.PRUrl != "" {
        fmt.Fprintf(&b, "PR: %s\n", task.PRUrl)
    }
    if task.BranchName != "" {
        fmt.Fprintf(&b, "Branch: %s\n", task.BranchName)
    }
    b.WriteString("Review the code changes, run tests, and validate the implementation.\n")
    b.WriteString("If approved, move to done:\n")
    fmt.Fprintf(&b, "  agentboard task move %s done\n", shortID)
    b.WriteString("If changes are needed, move back to in progress:\n")
    fmt.Fprintf(&b, "  agentboard task move %s in_progress\n", shortID)
```

**Change C: Add Review case to `buildInitialPrompt`** (after the InProgress case):

```go
case db.StatusReview:
    return "Run /workflows:review to perform a comprehensive code review of this task."
```

### File 3: `internal/cli/status_cmd.go` (2 changes)

**Change A: Add Review to summary map** (line 55):

```go
summary := boardSummary{
    Columns: map[string]int{
        string(db.StatusBacklog):    counts[string(db.StatusBacklog)],
        string(db.StatusPlanning):   counts[string(db.StatusPlanning)],
        string(db.StatusInProgress): counts[string(db.StatusInProgress)],
        string(db.StatusReview):     counts[string(db.StatusReview)],     // NEW
        string(db.StatusDone):       counts[string(db.StatusDone)],
    },
    Total: len(tasks),
}
```

**Change B: Add Review to text output** (between In Progress and Done lines):

```go
fmt.Printf("Review:      %d\n", summary.Columns[string(db.StatusReview)])
```

### File 4: `internal/board/local_test.go` (1 change)

**Update `TestMoveTaskService`** to include Review in the progression (line 89-93):

```go
statuses := []db.TaskStatus{
    db.StatusPlanning,
    db.StatusInProgress,
    db.StatusReview,      // NEW
    db.StatusDone,
}
```

## Summary of Changes

| File | Lines Changed | What |
|---|---|---|
| `internal/tui/board.go` | +2 | Add Review to `columnOrder` and `columnTitles` |
| `internal/agent/spawn.go` | +14, ~2 | Add Review prompts, update InProgress prompt |
| `internal/cli/status_cmd.go` | +2 | Add Review to status summary |
| `internal/board/local_test.go` | +1 | Add Review to test progression |

**Total: ~19 lines added, ~2 lines modified, 0 files created, 0 database migrations.**

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-23-agentboard-brainstorm.md` (Review flow: lines 83-89)
- Data model: `internal/db/models.go:11` (`StatusReview` already defined)
- Schema: `internal/db/schema.go:11` (CHECK constraint already includes `'review'`)
- Column definitions: `internal/tui/board.go:11-23`
- Agent prompts: `internal/agent/spawn.go:102-151`
- CLI status: `internal/cli/status_cmd.go:50-73`
- Test: `internal/board/local_test.go:83-104`
