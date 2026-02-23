---
status: pending
priority: p1
issue_id: "002"
tags: [code-review, architecture, bug]
dependencies: []
---

# Fix serve.go Unreachable Code - Server Discovery Broken

## Problem Statement

In `internal/cli/serve.go`, `srv.Start(ctx)` blocks until the server shuts down. The `peersync.WriteServerInfo()` call after it is unreachable during normal operation, so peer discovery via `server.json` never works in dedicated server mode.

## Findings

- **Architecture Review** Finding 2.2: `srv.Start()` blocks, `WriteServerInfo` is unreachable
- Contrast with `leader.go` which correctly starts the server in a goroutine (but leader.go is dead code)

### Location:
`internal/cli/serve.go`, lines 55-64

## Proposed Solutions

### Solution A: Start server in goroutine (Recommended)
Start `srv.Start(ctx)` in a goroutine, wait for the address, write server.json, then block on context cancellation.
- **Pros**: Matches the pattern in leader.go, enables peer discovery
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] Server starts in a goroutine
- [ ] `server.json` is written after server binds
- [ ] `server.json` is cleaned up on shutdown
- [ ] Signal handling still works correctly

## Work Log

- 2026-02-23: Created from code review synthesis
