---
status: pending
priority: p1
issue_id: "003"
tags: [code-review, concurrency, bug]
dependencies: ["001"]
---

# Fix Data Races (Connector.conn, Context Cancellation)

## Problem Statement

The `Connector.conn` field in `peersync/connector.go` is accessed from multiple goroutines without synchronization. Additionally, `readPump` in both `server/client.go` and `peersync/connector.go` uses an ineffective context cancellation pattern - context cancellation can take up to 30 seconds to propagate because `ReadMessage()` blocks on I/O.

## Findings

- **Architecture Review** Finding 7.1: Race on `Connector.conn` field
- **Architecture Review** Finding 7.2: Non-blocking context check before blocking ReadMessage
- **Performance Review** CRITICAL-3: 30-second goroutine leak on shutdown

### Locations:
- `internal/peersync/connector.go` - conn field accessed without mutex
- `internal/server/client.go:70-76` - ineffective context check
- `internal/peersync/connector.go:64-69` - same pattern

## Proposed Solutions

### Solution A: Mutex for conn + close-on-cancel pattern (Recommended)
1. Add `sync.Mutex` to protect `Connector.conn`
2. In both readPumps, spawn a goroutine that waits on `ctx.Done()` and calls `conn.Close()` to unblock ReadMessage
- **Pros**: Fixes both issues, clean shutdown
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] `Connector.conn` protected by mutex
- [ ] Context cancellation immediately unblocks readPump in client.go
- [ ] Context cancellation immediately unblocks readPump in connector.go
- [ ] `go test -race ./...` passes

## Work Log

- 2026-02-23: Created from code review synthesis
