---
status: pending
priority: p3
issue_id: "019"
tags: [code-review, architecture]
dependencies: []
---

# Replace saveNeeded Flag with tea.Cmd Pattern

## Problem Statement

`taskDetail.saveNeeded` is a side-channel flag checked after `Update()` returns. Cleaner Bubble Tea pattern: return a `tea.Cmd` that produces a `taskSaveRequestedMsg` directly from `applyEdits()`.

**Location:** `internal/tui/task_detail.go:33`, `internal/tui/app.go:492-496`

## Acceptance Criteria

- [ ] No `saveNeeded` flag; save intent communicated via tea.Cmd/Msg
