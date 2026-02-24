---
status: pending
priority: p1
issue_id: "012"
tags: [code-review, sqlite, migration, data-integrity]
dependencies: []
---

# Fix Migration v5: PRAGMA foreign_keys=OFF Outside Transaction

## Problem Statement

`PRAGMA foreign_keys=OFF` cannot be executed inside a SQLite transaction -- it is silently ignored. The existing migration runner in `sqlite.go:86-101` wraps migrations in `BeginTx`. If the `PRAGMA foreign_keys=OFF` is inside the migration SQL (as currently written in the plan), the DROP TABLE will cascade-delete all comments.

## Findings

- **Source**: performance-oracle review + data-migration-expert (deepen-plan)
- **Location**: `internal/db/sqlite.go:86-101` (migration runner uses `BeginTx`)
- **Evidence**: SQLite docs state PRAGMA foreign_keys cannot run inside a transaction
- **Risk**: CRITICAL data loss -- all comments deleted during migration

## Proposed Solutions

### Option A: Execute PRAGMA outside transaction (Recommended, Effort: Small, Risk: Low)
Modify the migration runner to:
1. Execute `PRAGMA foreign_keys=OFF` before `BeginTx`
2. Run table rebuild inside transaction
3. Execute `PRAGMA foreign_keys=ON` after `Commit`

### Option B: Add migration-specific runner
Create a `migrateV4toV5` function that handles the PRAGMA/transaction ordering explicitly, separate from the generic migration runner.

## Recommended Action

Option A -- minimal change to the migration runner, maximum safety.

## Acceptance Criteria

- [ ] Migration v5 tested on database WITH existing comments
- [ ] Comments survive migration (count before == count after)
- [ ] Foreign keys re-enabled after migration
- [ ] Integration test: create tasks with comments, run migration, verify comments intact
