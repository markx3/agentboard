---
status: complete
priority: p2
issue_id: "016"
tags: [code-review, security]
dependencies: []
---

# Add Title Validation in Edit Mode and CLI Update

## Problem Statement

`applyEdits()` in `task_detail.go` does not validate that the title is non-empty before saving. The SQLite CHECK constraint (`length(title) > 0`) prevents data corruption, but the error message is a raw DB error rather than a friendly UX message. Same issue in CLI `task update --title ""`.

## Findings

- **Security Reviewer:** "The error message leaks schema details. Add application-level validation."

**Location:** `internal/tui/task_detail.go:111-118`, `internal/cli/task_cmd.go:382-383`

## Proposed Solutions

### Solution A: Add validation before save (Recommended)
Check `title != ""` in `applyEdits()` before setting `saveNeeded`. In CLI, check before calling `UpdateTask`.

- **Effort:** Small (10 min)
- **Risk:** None

## Acceptance Criteria

- [ ] Empty title in TUI edit shows "Title cannot be empty" notification
- [ ] `agentboard task update --title ""` returns clear error message

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by security reviewer |
