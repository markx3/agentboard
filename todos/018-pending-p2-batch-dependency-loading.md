---
status: pending
priority: p2
issue_id: "018"
tags: [code-review, performance, sqlite]
dependencies: []
---

# Batch-Load Dependencies to Avoid N+1 Queries

## Problem Statement

The TUI board view will show dependency badges on each task card. If each task calls `ListDependencies()` individually, that's 30 extra queries per tick for a 30-task board (N+1 problem).

## Findings

- **Source**: performance-oracle
- **Location**: Plan section 1.4 (dependency CRUD)
- **Evidence**: No batch loading mechanism defined in the plan
- **Risk**: 30+ extra queries per 2.5s tick cycle

## Proposed Solutions

### Option A: Single batch query (Recommended, Effort: Small, Risk: Low)
After `ListTasks()`, run a single:
```sql
SELECT task_id, depends_on FROM task_dependencies
```
Build `map[string][]string` in memory and attach to tasks. The table is tiny (max hundreds of rows).

## Acceptance Criteria

- [ ] Dependencies loaded in a single query after `ListTasks()`
- [ ] `map[string][]string` structure available to TUI rendering
- [ ] No per-task dependency queries during board rendering
