---
status: pending
priority: p3
issue_id: "020"
tags: [code-review, quality]
dependencies: []
---

# Extract Migration Helper to Reduce Boilerplate

## Problem Statement

5 identical 16-line migration blocks in `sqlite.go`. An `applyMigration(ctx, version, sql)` helper + loop would save ~55 LOC.

**Location:** `internal/db/sqlite.go:86-178`

## Acceptance Criteria

- [ ] Single `applyMigration` helper replaces all copy-pasted blocks
