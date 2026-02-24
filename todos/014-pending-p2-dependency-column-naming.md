---
status: pending
priority: p2
issue_id: "014"
tags: [code-review, architecture]
dependencies: []
---

# Fix Inverted Column Naming in task_dependencies Table

## Problem Statement

The `task_dependencies` table uses `task_id` and `blocks_id` columns, but the Go API uses `AddDependency(taskID, blockerID)` where `taskID` = blocked task, `blockerID` = blocker. The SQL INSERT swaps parameter order: `INSERT INTO task_dependencies (task_id, blocks_id) VALUES (blockerID, taskID)`. This inverted mapping is a cognitive trap for future maintainers.

## Findings

- **Simplicity Reviewer:** "Every future developer touching this code will misread it. The swap is consistent but is a cognitive landmine."
- **Architecture Reviewer:** Column naming confusion flagged independently.

**Location:** `internal/db/schema.go` (table definition), `internal/db/tasks.go` (all dependency CRUD)

## Proposed Solutions

### Solution A: Rename columns to `blocker_id` and `blocked_id` (Recommended)
Since v6 migration hasn't shipped to users yet, rename columns in the migration and base schema to match Go semantics. Update all SQL queries.

- **Pros:** Eliminates confusing parameter swapping forever
- **Cons:** Must update schema, migration, and all queries
- **Effort:** Medium (30 min)
- **Risk:** Low (no users on v6 yet)

### Solution B: Add comments documenting the swap
- **Pros:** Zero code change
- **Cons:** Comments rot, the trap persists
- **Effort:** Small
- **Risk:** Medium (future bugs)

## Acceptance Criteria

- [ ] SQL column names match Go parameter names (no swapping)
- [ ] All dependency queries use consistent naming

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by simplicity + architecture reviewers |
