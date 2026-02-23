---
status: pending
priority: p2
issue_id: "006"
tags: [code-review, security, validation]
dependencies: []
---

# Validate TaskStatus in WebSocket Hub Handlers

## Problem Statement

The Hub's `handleMessage` for `task.move` casts `p.ToColumn` directly to `db.TaskStatus` without validation. While the DB CHECK constraint catches invalid values, the error messages may leak schema details. The CLI validates with `Valid()` but the WebSocket path does not.

## Findings

- **Security Review** MEDIUM-3: TaskStatus not validated on WebSocket task.move
- **Architecture Review** Finding 4.3: Hub has direct knowledge of domain logic

### Location:
`internal/server/hub.go`, line 95

## Proposed Solutions

### Solution A: Add Valid() check before MoveTask call (Recommended)
```go
if !db.TaskStatus(p.ToColumn).Valid() {
    h.sendReject(cm.client, "invalid status")
    return
}
```
- **Effort**: Trivial
- **Risk**: None

## Acceptance Criteria

- [ ] Invalid status values rejected with clean error message
- [ ] Valid status values still work correctly

## Work Log

- 2026-02-23: Created from code review synthesis
