---
status: pending
priority: p1
issue_id: "011"
tags: [code-review, sqlite, performance, correctness]
dependencies: []
---

# Verify `_txlock=immediate` Compatibility with modernc.org/sqlite

## Problem Statement

The plan proposes adding `_txlock=immediate` to the SQLite DSN to prevent lock escalation failures under multi-process writes. However, `_txlock` is a `mattn/go-sqlite3` DSN parameter. The codebase uses `modernc.org/sqlite` (pure Go, no CGO). If the driver silently ignores this parameter, the entire multi-process write safety story collapses.

Additionally, implicit transactions from `ExecContext()` (used by `UpdateTask` and the new `UpdateTaskFields`) default to `BEGIN DEFERRED`, which means `busy_timeout` doesn't apply during lock escalation even if the DSN parameter works for explicit transactions.

## Findings

- **Source**: performance-oracle review
- **Location**: `internal/db/sqlite.go:17-43` (current `Open()` function)
- **Evidence**: `sql.Open("sqlite", dbPath)` uses a plain file path with no DSN parameters. The modernc.org/sqlite driver documentation must be checked for DSN parameter support.
- **Risk**: Without `BEGIN IMMEDIATE`, concurrent CLI processes (enrichment agents + human) hitting SQLite writes will encounter "database is locked" errors that `busy_timeout` cannot resolve.

## Proposed Solutions

### Option A: Verify modernc.org/sqlite DSN support (Effort: Small, Risk: Low)
Check the driver documentation and test whether `_txlock=immediate` is supported. If yes, add to DSN.
- **Pros**: Minimal code change
- **Cons**: If unsupported, wasted effort

### Option B: Add `BeginImmediate` helper (Effort: Medium, Risk: Low)
Create a wrapper in `sqlite.go` that executes `BEGIN IMMEDIATE` manually before write operations:
```go
func (d *DB) execImmediate(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
    if _, err := d.conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
        return nil, fmt.Errorf("begin immediate: %w", err)
    }
    result, err := d.conn.ExecContext(ctx, query, args...)
    if err != nil {
        d.conn.ExecContext(ctx, "ROLLBACK")
        return nil, err
    }
    _, commitErr := d.conn.ExecContext(ctx, "COMMIT")
    return result, commitErr
}
```
- **Pros**: Works regardless of driver DSN support
- **Cons**: More invasive, must be used for all write operations

### Option C: Both (Recommended, Effort: Medium, Risk: Low)
Verify DSN support AND add the helper as a safety net. Belt and suspenders.

## Recommended Action

Option C. This is P0 -- must be resolved before Phase 2 (enrichment) can work.

## Acceptance Criteria

- [ ] modernc.org/sqlite DSN parameter compatibility documented
- [ ] Write operations use `BEGIN IMMEDIATE` (via DSN or helper)
- [ ] Test with 3 concurrent processes writing to same DB file
- [ ] `go test -race ./internal/db/...` passes
