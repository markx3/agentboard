---
status: pending
priority: p2
issue_id: "007"
tags: [code-review, security]
dependencies: []
---

# Harden File Permissions (server.json, DB directory)

## Problem Statement

`server.json` is written with 0644 (world-readable) and the DB directory is created with 0755. On multi-user systems, other users can discover the server address or potentially read task data.

## Findings

- **Security Review** MEDIUM-4: server.json file permissions 0644
- **Security Review** LOW-2: Database directory 0755

### Locations:
- `internal/peersync/discovery.go`, line 32 - `0o644` → `0o600`
- `internal/db/sqlite.go`, line 19 - `0o755` → `0o700`

## Proposed Solutions

### Solution A: Tighten permissions (Recommended)
Change server.json to 0600 and db directory to 0700.
- **Effort**: Trivial
- **Risk**: None

## Acceptance Criteria

- [ ] server.json written with 0600
- [ ] DB directory created with 0700

## Work Log

- 2026-02-23: Created from code review synthesis
