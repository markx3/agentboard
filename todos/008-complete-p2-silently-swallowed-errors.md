---
status: pending
priority: p2
issue_id: "008"
tags: [code-review, quality, reliability]
dependencies: []
---

# Fix Silently Swallowed Errors (time.Parse, JSON Marshal)

## Problem Statement

Multiple locations discard errors from `time.Parse` and `json.Marshal` with `_`. This can produce tasks with zero-value timestamps or malformed protocol messages. The `mustMarshal` function is named as if it panics but silently returns nil.

## Findings

- **Architecture Review** Finding 5.3: JSON marshaling errors silently swallowed in Hub
- **Architecture Review** Finding 8.1: time.Parse errors silently ignored in DB layer
- **Security Review** LOW-4: Suppressed JSON marshal errors

### Locations:
- `internal/db/tasks.go`: lines 67-68, 93-94, 121-122 (time.Parse)
- `internal/db/comments.go`: line 46 (time.Parse)
- `internal/server/hub.go`: lines 154, 156, 161, 167-168 (json.Marshal)
- `internal/server/client.go`: line 134 (mustMarshal)

## Proposed Solutions

### Solution A: Log errors, don't silently discard (Recommended)
- For time.Parse: log a warning and use the zero value, or return error
- For json.Marshal in Hub: log errors (these are programming bugs if they occur)
- For mustMarshal: make it actually panic (consistent with Go naming convention) or rename and handle
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] time.Parse errors logged or returned
- [ ] JSON marshal errors logged in Hub
- [ ] mustMarshal either panics or is renamed and errors are handled
- [ ] No silent error discarding with `_` for these critical paths

## Work Log

- 2026-02-23: Created from code review synthesis
