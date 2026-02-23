---
status: pending
priority: p2
issue_id: "005"
tags: [code-review, database, bug]
dependencies: []
---

# Wrap CreateTask Position Calculation in Transaction

## Problem Statement

`CreateTask` in `db/tasks.go` reads `MAX(position)` and then INSERTs in two separate queries without a transaction. Under concurrent CLI usage, two goroutines can get the same MAX value, causing a UNIQUE constraint violation on `idx_tasks_status_position`. `MoveTask` already uses a transaction correctly.

## Findings

- **Architecture Review** Finding 8.2: TOCTOU race on position calculation
- **Performance Review** OPT-2: Non-atomic position in CreateTask

### Location:
`internal/db/tasks.go`, lines 12-51

## Proposed Solutions

### Solution A: Wrap in BeginTx (Recommended)
Use `d.conn.BeginTx(ctx, nil)` to wrap the SELECT MAX + INSERT in a transaction, matching `MoveTask` pattern.
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] CreateTask uses a transaction
- [ ] Concurrent create calls don't produce constraint violations
- [ ] Existing tests still pass

## Work Log

- 2026-02-23: Created from code review synthesis
