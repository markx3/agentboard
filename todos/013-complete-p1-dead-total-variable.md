---
status: complete
priority: p1
issue_id: "013"
tags: [code-review, quality]
dependencies: []
---

# Remove Dead `total` and Redundant `done` Variables in summaryBar()

## Problem Statement

In `board.go:summaryBar()`, the `total` variable is incremented but never read. The `done` variable is redundant with `statusCounts[db.StatusDone]`. Dead code that may cause linting warnings.

## Findings

- **Simplicity Reviewer:** "`total` is dead code. `done` is redundant with `statusCounts[db.StatusDone]`."

**Location:** `internal/tui/board.go:149-182`

## Proposed Solutions

### Solution A: Remove both variables (Recommended)
Remove `total` entirely. Replace `done` usage with `statusCounts[db.StatusDone]`.

- **Effort:** Small (5 min)
- **Risk:** None

## Acceptance Criteria

- [ ] No unused variables in `summaryBar()`
- [ ] Summary bar still displays correct counts

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by simplicity reviewer |
