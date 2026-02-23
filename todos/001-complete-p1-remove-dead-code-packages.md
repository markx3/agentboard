---
status: pending
priority: p1
issue_id: "001"
tags: [code-review, architecture, simplicity]
dependencies: []
---

# Remove ~860 Lines of Dead Code (4 Unused Packages + Dead Files)

## Problem Statement

Four entire packages (`agent`, `worktree`, `config`, `tmux`) are never imported by any production code. Additionally, several files within active packages are dead (`tui/shell_popup.go`, `peersync/offline.go`, `peersync/leader.go`, `cli/peers.go`). This represents ~23% of non-test LOC. Two of the dead packages (`agent`, `worktree`) contain security vulnerabilities (CRITICAL-1: arbitrary command execution via init_script, CRITICAL-2: command injection via tmux). Removing the dead code eliminates these vulnerabilities.

## Findings

- **Simplicity Review**: 4 dead packages totaling ~470 LOC, plus ~390 LOC of dead files/symbols
- **Security Review**: CRITICAL-1 (init_script RCE) and CRITICAL-2 (tmux command injection) are both in dead packages
- **Architecture Review**: Finding 6.3 (leader election not implemented), Finding 6.4 (epoch numbers not implemented) - both in dead `leader.go`

### Dead Packages:
- `internal/agent/` (registry.go + spawn.go) - 100 LOC
- `internal/worktree/` (manager.go + setup.go) - 122 LOC
- `internal/config/` (global.go + project.go + merged.go) - 144 LOC + 63 LOC tests
- `internal/tmux/` (manager.go + capture.go) - 104 LOC

### Dead Files in Active Packages:
- `internal/tui/shell_popup.go` - 77 LOC
- `internal/peersync/offline.go` - 50 LOC
- `internal/peersync/leader.go` - 120 LOC
- `internal/cli/peers.go` - 23 LOC

### Dead Symbols:
- `overlayClosedMsg` in tui/messages.go
- `taskStyle`, `selectedTaskStyle` in tui/styles.go
- `Search` key binding in tui/keys.go
- `Conn()` in db/sqlite.go
- `MsgSyncAck`, `MsgAgentStatus`, `MsgCommentAdd`, `MsgLeaderPromote` in server/protocol.go
- `Hub()` accessor in server/server.go
- `PeerCount()` in server/hub.go (also a data race - see #003)
- `ConnectWithRetry()` in peersync/connector.go
- `GetIdentity()` in auth/github.go
- `ServerInfo.Host`/`Port` fields in peersync/discovery.go

## Proposed Solutions

### Solution A: Delete all dead code (Recommended)
- Delete the 4 dead packages entirely
- Delete dead files in active packages
- Remove unused symbols from active files
- Remove config_test.go
- Update go.mod (remove BurntSushi/toml if no longer needed)
- **Pros**: Eliminates 23% dead code, removes 2 CRITICAL security issues, simplifies codebase
- **Effort**: Small
- **Risk**: Low - code is confirmed unused

## Acceptance Criteria

- [ ] `internal/agent/` directory deleted
- [ ] `internal/worktree/` directory deleted
- [ ] `internal/config/` directory deleted
- [ ] `internal/tmux/` directory deleted
- [ ] `internal/tui/shell_popup.go` deleted
- [ ] `internal/peersync/offline.go` deleted
- [ ] `internal/peersync/leader.go` deleted
- [ ] `internal/cli/peers.go` deleted
- [ ] Unused symbols removed from active files
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes
- [ ] `go vet ./...` clean

## Work Log

- 2026-02-23: Created from code review synthesis
