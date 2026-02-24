---
status: complete
priority: p1
issue_id: "011"
tags: [code-review, performance, architecture]
dependencies: []
---

# Replace Recursive SQL Cycle Detection with In-Memory Traversal

## Problem Statement

`HasCycle` in `internal/db/tasks.go` uses `dfsReaches` which fires one SQL query per node visited in the DFS. This holds multiple open `*sql.Rows` cursors simultaneously on the single SQLite connection during recursion. While modernc.org/sqlite handles this in practice, it is fragile and has O(N) query overhead for a chain of depth N.

All three reviewers (architecture, performance, simplicity) flagged this independently.

## Findings

- **Architecture:** Holding nested `*sql.Rows` on `SetMaxOpenConns(1)` is fragile. Would break with any connection-pooled driver.
- **Performance:** Each DFS node = 1 SELECT query. For dense graphs, this is O(N) sequential queries.
- **Simplicity:** `GetAllDependencies()` already loads the full graph in 1 query. DFS should run in-memory.

**Location:** `internal/db/tasks.go:284-319`

## Proposed Solutions

### Solution A: In-Memory DFS using GetAllDependencies (Recommended)
Load all edges with a single `SELECT task_id, blocks_id FROM task_dependencies`, build adjacency map, run DFS in memory. Eliminates `dfsReaches` entirely.

- **Pros:** 1 query regardless of graph size, no cursor nesting, simpler code
- **Cons:** Loads full graph even for trivial checks
- **Effort:** Small (30 min)
- **Risk:** Low

## Recommended Action

_To be filled during triage_

## Technical Details

- **Affected files:** `internal/db/tasks.go`
- **Remove:** `dfsReaches` method (~28 LOC)
- **Modify:** `HasCycle` to load adjacency map and traverse in-memory

## Acceptance Criteria

- [ ] `HasCycle` uses at most 1 SQL query
- [ ] No recursive `*sql.Rows` nesting
- [ ] Cycle detection still rejects A→B→A and A→B→C→A patterns
- [ ] Self-blocking still rejected by CHECK constraint

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2026-02-23 | Created | Flagged by architecture, performance, and simplicity reviewers |

## Resources

- PR: https://github.com/markx3/agentboard/pull/16
