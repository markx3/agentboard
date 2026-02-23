---
status: pending
priority: p2
issue_id: "009"
tags: [code-review, quality, duplication]
dependencies: []
---

# Extract scanTask Helper to Deduplicate DB Column Scanning

## Problem Statement

The exact same 13-column scan logic is repeated 3 times in `db/tasks.go` (GetTask, ListTasks, ListTasksByStatus). Any schema change requires updating all three locations identically.

## Findings

- **Architecture Review** Finding 8.4: Scan column list duplicated across 4 functions
- **Simplicity Review**: Duplicated scan boilerplate

### Location:
`internal/db/tasks.go`: lines 56-69, 83-98, 110-126

## Proposed Solutions

### Solution A: Extract scanTask helper (Recommended)
Create a private `scanTask(scanner interface{ Scan(...interface{}) error }) (Task, error)` that handles the scan and time parsing for all three functions.
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] Single `scanTask` helper function
- [ ] All three query functions use the helper
- [ ] ~20 LOC reduction
- [ ] All existing tests pass

## Work Log

- 2026-02-23: Created from code review synthesis
