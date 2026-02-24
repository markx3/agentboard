---
status: complete
priority: p2
issue_id: "017"
tags: [code-review, quality, yagni]
dependencies: []
---

# Remove Unused GetBlockers Method (YAGNI)

## Problem Statement

`GetBlockers(ctx, taskID)` is defined on the `Service` interface, `LocalService`, and `DB` but never called anywhere. The codebase uses `GetAllDependencies()` exclusively. This violates YAGNI and adds surface area to the interface that future `Service` implementations must implement.

## Findings

- **Simplicity Reviewer:** "Dead code -- a 'just in case' method that violates YAGNI. Remove it now."

**Location:** `internal/board/service.go:24`, `internal/board/local.go:86-88`, `internal/db/tasks.go:244-262`

## Proposed Solutions

### Solution A: Remove from all three files (Recommended)
Delete the interface method, service implementation, and DB implementation. ~22 LOC removed.

- **Effort:** Small (5 min)
- **Risk:** None

**Note:** If todo #012 (CLI missing dependency data) uses `GetBlockers` in `runTaskGet`, keep the DB method but remove the unused service interface method.

## Acceptance Criteria

- [ ] No unused methods in the codebase
- [ ] Service interface is minimal

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by simplicity reviewer |
