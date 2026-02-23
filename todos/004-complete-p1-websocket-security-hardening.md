---
status: pending
priority: p1
issue_id: "004"
tags: [code-review, security, websocket]
dependencies: []
---

# WebSocket Security Hardening (Origin Check, Auth Timeout, Error Leaks)

## Problem Statement

Three WebSocket security issues that collectively weaken the server's security posture:
1. Origin check disabled (`CheckOrigin: return true`) - enables CSWSH attacks
2. No read deadline before auth message - slow-loris DoS
3. Auth error messages leak internal details (GitHub API errors)

## Findings

- **Security Review** HIGH-1: CSWSH vulnerability via disabled origin check
- **Security Review** HIGH-2: Auth error leaks GitHub API status codes
- **Security Review** MEDIUM-2: No auth timeout enables resource exhaustion

### Locations:
- `internal/server/server.go:21-27` - CheckOrigin returns true
- `internal/server/server.go:84-92` - No read deadline before auth
- `internal/server/server.go:95-98` - Raw error sent to client

## Proposed Solutions

### Solution A: Fix all three in server.go (Recommended)
1. CheckOrigin: Allow empty origin (non-browser) + localhost origins only
2. Set 5-second read deadline before auth message read, clear after
3. Log the detailed error server-side, send generic "authentication failed" to client
- **Pros**: Fixes 3 findings in one pass, defense-in-depth
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] CheckOrigin rejects non-localhost browser origins
- [ ] Auth message has 5-second timeout
- [ ] Auth error messages don't leak internal details
- [ ] Non-browser clients (no Origin header) still work

## Work Log

- 2026-02-23: Created from code review synthesis
