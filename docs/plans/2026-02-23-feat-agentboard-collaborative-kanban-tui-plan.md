---
title: "feat: Agentboard - Collaborative Agentic Task Management TUI"
type: feat
date: 2026-02-23
deepened: 2026-02-23
---

# Agentboard - Collaborative Agentic Task Management TUI

## Enhancement Summary

**Deepened on:** 2026-02-23
**Sections enhanced:** 7 major sections
**Review agents used:** Architecture Strategist, Security Sentinel, Performance Oracle, Code Simplicity Reviewer, Data Integrity Guardian, Pattern Recognition Specialist, Agent-Native Reviewer

### Key Improvements
1. **Simplified architecture**: Renamed `internal/sync/` to `internal/peersync/` to avoid stdlib shadow; extracted domain logic from tui/board.go into a BoardService interface
2. **Security hardened**: Token hashing, tmux command sanitization, per-message auth via HMAC, parameterized SQL queries, bind to localhost by default, rate limiting
3. **Performance optimized**: Buffered WebSocket channels with batch drain, 2Hz tmux capture throttle, SQLite SetMaxOpenConns(1) + busy_timeout, compressed sync.full
4. **Data integrity**: CHECK constraints on status/agent_status, composite uniqueness for position, epoch-aware offline queue with idempotency keys, schema versioning
5. **Agent-native parity**: CLI subcommand tree for programmatic access (`agentboard task {list,create,move,...}`), AgentAdapter interface for multi-agent support
6. **Reduced MVP scope**: Cut Compound column, theme support, and review flow from MVP; focus on 4 core columns (Backlog, Planning, In Progress, Done)

### New Considerations Discovered
- `internal/sync/` shadows Go stdlib `sync` package — renamed to `internal/peersync/`
- `listenForWS` as `tea.Cmd` with infinite loop violates Bubble Tea contract — must return after single read
- Heartbeat timeout reduced from 90s to 30s for faster failure detection
- Need `context.Context` propagation throughout for clean shutdown
- Agent-native CLI subcommands critical for agent-to-agent collaboration

## Overview

Build a terminal-based collaborative Kanban board in Go for managing agentic coding tasks across a team. Multiple developers connect to a shared project board in real-time via WebSocket, each running their own AI coding agents (Claude Code, Cursor CLI, etc.) in local worktrees, while everyone sees the full project's progress. Think agtx, but multiplayer.

## Problem Statement

Teams using AI coding agents lack shared visibility. Each developer runs agents in isolation with no central view of who's working on what, which tasks are in progress, or where bottlenecks exist. Existing project management tools are too heavy for this lightweight, terminal-native workflow. agtx solved single-player agent orchestration; Agentboard extends this to teams.

## Proposed Solution

A single Go binary (`agentboard`) that acts as both client and server. The first person to open a project becomes the peer-leader (WebSocket server). Others connect automatically via a repo config file or manually via `--connect`. The TUI renders a Kanban board using Bubble Tea + Lipgloss, with real-time updates via WebSocket. Tasks are persisted in SQLite and agent sessions run inside tmux.

## Technical Approach

### Architecture

```
cmd/agentboard/
  main.go              -- Entry point, Cobra root command setup

internal/
  cli/
    root.go            -- Cobra root command (default: start TUI)
    serve.go           -- `agentboard serve` (persistent server mode)
    init.go            -- `agentboard init` (project setup)
    peers.go           -- `agentboard peers` (list connected peers)

  tui/
    app.go             -- Top-level Bubble Tea model, message routing
    board.go           -- Kanban board view (columns + navigation)
    column.go          -- Single column wrapping bubbles/list
    task_item.go       -- Task implementing list.Item interface
    task_detail.go     -- Task detail view (overlay/popup)
    task_form.go       -- Task creation/edit form (textinput + textarea)
    shell_popup.go     -- tmux pane viewer (viewport)
    notification.go    -- Toast notification bar
    styles.go          -- Lipgloss style definitions
    keys.go            -- Key bindings (help.KeyMap)
    messages.go        -- TUI-internal message types

  server/
    server.go          -- WebSocket server (gorilla/websocket)
    hub.go             -- Connection hub (register/unregister/broadcast)
    client.go          -- Per-connection reader/writer goroutines
    protocol.go        -- Wire protocol message types (JSON)
    sequencer.go       -- Server-side sequence numbering for conflict resolution

  peersync/              -- Renamed from sync/ to avoid shadowing stdlib sync package
    connector.go       -- WebSocket client, reconnection logic
    discovery.go       -- Peer discovery via .agentboard/server.json
    leader.go          -- Leader election and promotion
    offline.go         -- Offline queue and replay

  db/
    sqlite.go          -- SQLite setup (WAL mode, pragmas)
    schema.go          -- Schema creation and migrations
    tasks.go           -- Task CRUD operations
    comments.go        -- Comment operations
    models.go          -- Task, Comment, Project Go types

  agent/
    registry.go        -- Agent definitions and detection (exec.LookPath)
    spawn.go           -- Launch agent in tmux session

  tmux/
    manager.go         -- Tmux session/window/pane operations
    capture.go         -- Pane content capture for shell popup

  worktree/
    manager.go         -- Git worktree create/remove/list
    setup.go           -- File copying (.env) and init script execution

  config/
    global.go          -- ~/.agentboard/config.toml
    project.go         -- .agentboard/config.toml
    merged.go          -- Merged config with precedence

  auth/
    github.go          -- gh auth token extraction + GitHub API verification
```

### Key Dependencies

| Dependency | Purpose | Version |
|---|---|---|
| `github.com/charmbracelet/bubbletea` | TUI framework (Elm architecture) | latest stable |
| `github.com/charmbracelet/lipgloss` | Terminal styling/layout | latest stable |
| `github.com/charmbracelet/bubbles` | TUI components (list, textinput, textarea, viewport) | latest stable |
| `github.com/coder/websocket` | WebSocket server/client (gorilla/websocket successor) | latest |
| `modernc.org/sqlite` | Pure-Go SQLite (no CGO, easier cross-compile) | latest |
| `github.com/spf13/cobra` | CLI framework | v1.9+ |
| `github.com/BurntSushi/toml` | TOML config parsing | latest |
| `github.com/google/uuid` | Task ID generation | latest |

**SQLite choice**: `modernc.org/sqlite` (pure Go) over `mattn/go-sqlite3` (CGO) for single-binary distribution via `brew install`. Performance difference is negligible for this workload (small dataset, low write frequency).

### WebSocket Protocol

JSON messages with typed envelope:

```json
{
  "type": "task.move",
  "seq": 42,
  "sender": "alice",
  "payload": {
    "task_id": "uuid",
    "from_column": "planning",
    "to_column": "in_progress"
  }
}
```

**Message types:**
- `sync.full` -- Full board state (sent on initial connection)
- `sync.ack` -- Server acknowledges a client action with assigned sequence number
- `sync.reject` -- Server rejects action (conflict)
- `task.create`, `task.update`, `task.move`, `task.delete`
- `task.claim`, `task.unclaim`
- `task.agent_status` -- Agent idle/active/error status update
- `comment.add`
- `peer.join`, `peer.leave`
- `leader.promote` -- Leader migration notification
- `ping`, `pong` -- Heartbeat (30s interval)

**Conflict resolution**: Server-side sequencing. Server assigns monotonically increasing sequence numbers. First write wins; conflicting writes get `sync.reject` with current state.

### SQLite Schema

```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL CHECK(length(title) > 0 AND length(title) <= 500),
    description TEXT DEFAULT '',
    status TEXT NOT NULL DEFAULT 'backlog'
        CHECK(status IN ('backlog','planning','in_progress','review','done')),
    assignee TEXT DEFAULT '',
    branch_name TEXT DEFAULT '',
    pr_url TEXT DEFAULT '',
    pr_number INTEGER DEFAULT 0,
    agent_name TEXT DEFAULT '',
    agent_status TEXT DEFAULT 'idle'
        CHECK(agent_status IN ('idle','active','error')),
    position INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE comments (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    author TEXT NOT NULL CHECK(length(author) > 0),
    body TEXT NOT NULL CHECK(length(body) > 0),
    created_at TEXT NOT NULL
);

CREATE TABLE meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Indexes
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_assignee ON tasks(assignee);
CREATE UNIQUE INDEX idx_tasks_status_position ON tasks(status, position);
CREATE INDEX idx_comments_task_id ON comments(task_id);
```

**Pragmas** (applied on open):
```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -8000;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

**Critical**: Call `db.SetMaxOpenConns(1)` on the `*sql.DB` to prevent SQLite concurrent write contention. SQLite handles only one writer at a time; connection pooling causes `SQLITE_BUSY` errors.

### Bubble Tea Architecture

The top-level `App` model owns:
- `board Board` -- Kanban board with columns
- `wsChan chan protocol.Message` -- WebSocket message channel
- `overlay Overlay` -- Current popup (task detail, shell, form) or nil
- `notifications []Notification` -- Toast messages
- `connStatus ConnectionStatus` -- Server/client/offline status
- `width, height int` -- Terminal dimensions

**Message routing:**
1. `tea.WindowSizeMsg` → broadcast to all children
2. `tea.KeyMsg` → if overlay open, route to overlay; else route to board
3. `wsMessageMsg` → apply remote state change to board
4. All other msgs → route to focused component

**WebSocket integration** uses the channel+Cmd relay pattern. Note: `tea.Cmd` functions must return after producing a single message (Bubble Tea contract). A background goroutine pumps WebSocket messages into a buffered channel, and a Cmd reads one message at a time:
```go
// Background goroutine (started once, NOT a tea.Cmd)
func pumpWS(ctx context.Context, conn *websocket.Conn, ch chan<- protocol.Message) {
    defer close(ch)
    for {
        var msg protocol.Message
        if err := conn.ReadJSON(&msg); err != nil {
            select {
            case ch <- protocol.Message{Type: "error"}:
            case <-ctx.Done():
            }
            return
        }
        select {
        case ch <- msg:
        case <-ctx.Done():
            return
        }
    }
}

// tea.Cmd that reads ONE message and returns (correct Bubble Tea contract)
func waitForWS(ch <-chan protocol.Message) tea.Cmd {
    return func() tea.Msg {
        msg, ok := <-ch
        if !ok {
            return wsDisconnectedMsg{}
        }
        return wsMessageMsg(msg)
    }
}
```

### Peer-Leader Protocol

1. **Startup**: Check `.agentboard/server.json` for existing server
2. **If no server**: Start WebSocket server on random port, write to `server.json`, become leader
3. **If server exists**: Try to connect. If connection fails, become new leader
4. **Leader disconnect**: Server sends `leader.promote` to the longest-connected peer before graceful shutdown. On crash, peers detect via heartbeat timeout (30s), longest-connected peer self-promotes
5. **State transfer**: New leader receives full SQLite dump via `sync.full` message, writes to local SQLite, starts accepting connections
6. **Epoch numbers**: Each leader increments an epoch counter (persisted in `meta` table). Clients reject messages from stale epochs to prevent split-brain

### Worktree Lifecycle

1. **Claim (Backlog → Planning)**:
   - `git worktree add .agentboard/worktrees/<branch-name> -b agentboard/<task-title-slug>`
   - Copy files from project config `copy_files` list
   - Run `init_script` if configured
   - Launch agent in tmux window
2. **Unclaim / reassign**:
   - Check for uncommitted changes; warn user if present
   - Kill tmux window for this task
   - `git worktree remove .agentboard/worktrees/<branch-name>`
3. **Done**:
   - Kill tmux window
   - Remove worktree (branch preserved on remote)

### tmux Integration

Use a dedicated tmux server named `"agentboard"` (via `tmux -L agentboard`) to isolate from user sessions (pattern from agtx):

- **Session per project**: `agentboard:myproject`
- **Window per task**: Named after task title
- **Agent launch**: `tmux -L agentboard new-window -t agentboard:myproject -n "task-title" -c <worktree-path> <agent-command>`
- **Capture pane**: `tmux -L agentboard capture-pane -t agentboard:myproject:task-title -p -S -500`
- **Send keys**: `tmux -L agentboard send-keys -t agentboard:myproject:task-title "command" Enter`

### Authentication Flow

1. Run `gh auth token` to get the user's GitHub PAT
2. On WebSocket handshake, client sends token as first message
3. Server calls `GET https://api.github.com/user` with the token
4. If valid, extract `login` field as the user's identity
5. If invalid/expired, reject connection with error message

**Security hardening:**
- Never log or store PATs in plaintext — hash with SHA-256 for session tracking
- Bind WebSocket server to `127.0.0.1` by default (not `0.0.0.0`); use `--bind 0.0.0.0` flag for LAN
- Use parameterized SQL queries exclusively — never interpolate user input
- Sanitize task titles before passing to tmux commands (strip shell metacharacters)
- Add rate limiting: max 60 messages/minute per connection, 10 connections/IP
- Validate JSON message sizes (max 64KB per message, max 1MB for `sync.full`)

### Agent-Native CLI Interface

For programmatic access (enabling agent-to-agent collaboration), expose all capabilities as CLI subcommands:

```
agentboard task list [--status=<status>] [--assignee=<user>] [--json]
agentboard task create --title=<title> [--description=<desc>]
agentboard task move <task-id> <target-column>
agentboard task claim <task-id>
agentboard task unclaim <task-id>
agentboard task get <task-id> [--json]
agentboard task delete <task-id>
agentboard status [--json]         # Board summary, peer count
```

All commands support `--json` output for machine consumption. These commands connect to the running server via WebSocket (same protocol as TUI).

## Implementation Phases

### Phase 1: Foundation (Local-only TUI + SQLite)

Build the single-player Kanban board without networking. This validates the TUI architecture and data model.

**Tasks:**
- [ ] **Project scaffolding**: `go mod init`, Cobra CLI setup, directory structure - `cmd/agentboard/main.go`, `internal/cli/root.go`
- [ ] **SQLite layer**: Schema creation with CHECK constraints, Task CRUD with parameterized queries, WAL mode, `SetMaxOpenConns(1)`, `busy_timeout=5000` - `internal/db/sqlite.go`, `internal/db/schema.go`, `internal/db/tasks.go`, `internal/db/models.go`
- [ ] **BoardService interface**: Domain logic layer between TUI and DB. All task operations go through this interface (enables swapping local DB for WebSocket in Phase 3) - `internal/board/service.go`, `internal/board/local.go`
- [ ] **Config system**: Global + project TOML parsing with merge - `internal/config/global.go`, `internal/config/project.go`, `internal/config/merged.go`
- [ ] **Bubble Tea board**: Board model with 4 columns (Backlog, Planning, In Progress, Done), vim navigation (h/j/k/l), focused column highlighting - `internal/tui/app.go`, `internal/tui/board.go`, `internal/tui/column.go`, `internal/tui/styles.go`, `internal/tui/keys.go`
- [ ] **Task item**: Task implementing `list.Item` with status indicators - `internal/tui/task_item.go`
- [ ] **Task creation form**: Overlay for creating new tasks (title + description) - `internal/tui/task_form.go`
- [ ] **Task detail popup**: View task details, agent status - `internal/tui/task_detail.go`
- [ ] **Task movement**: `m` to move right, support all column transitions - `internal/tui/board.go`
- [ ] **Task search**: `/` for fuzzy search using bubbles/list built-in filter - `internal/tui/board.go`
- [ ] **Notification bar**: Toast messages at bottom of screen - `internal/tui/notification.go`
- [ ] **`agentboard init`**: Create `.agentboard/` directory and default config - `internal/cli/init.go`
- [ ] **Context propagation**: Use `context.Context` throughout for clean shutdown - all packages

**Success criteria:** User can create, view, move, and delete tasks across 4 columns. Data persists in SQLite across restarts. All task operations go through BoardService interface.

### Phase 2: Agent Orchestration (tmux + worktrees + agents)

Add local agent session management. Still single-player.

**Tasks:**
- [ ] **Agent registry**: Define supported agents, detect via `exec.LookPath` - `internal/agent/registry.go`
- [ ] **tmux manager**: Create/kill windows, capture pane, send keys - `internal/tmux/manager.go`, `internal/tmux/capture.go`
- [ ] **Worktree manager**: Create/remove worktrees, copy files, run init scripts - `internal/worktree/manager.go`, `internal/worktree/setup.go`
- [ ] **Claim flow**: Moving Backlog→Planning creates worktree + launches agent in tmux - `internal/tui/board.go` (wire to worktree + tmux)
- [ ] **Shell popup**: Live tmux pane viewer via viewport (press `s` on active task) - `internal/tui/shell_popup.go`
- [ ] **Agent status**: Track idle/active/error state, show indicator on task card - `internal/tui/task_item.go`
- [ ] **Done cleanup**: Remove worktree and kill tmux window on Done transition - `internal/worktree/manager.go`
- [ ] **Unclaim with safety**: Warn on uncommitted changes before worktree removal - `internal/worktree/manager.go`
- [ ] **GitHub auth**: Extract token via `gh auth token`, verify via API - `internal/auth/github.go`

**Success criteria:** User can claim a task, see the agent running in a tmux popup, track agent status, and complete the full lifecycle.

### Phase 3: Networking (WebSocket collaboration)

Add multiplayer. This is the core differentiator over agtx.

**Tasks:**
- [ ] **WebSocket server**: Hub pattern with register/unregister/broadcast channels, bind to 127.0.0.1 by default - `internal/server/server.go`, `internal/server/hub.go`, `internal/server/client.go`
- [ ] **Wire protocol**: JSON message types, envelope with type/seq/sender/payload, max 64KB per message - `internal/server/protocol.go`
- [ ] **Server-side sequencer**: Monotonic sequence numbers, first-write-wins - `internal/server/sequencer.go`
- [ ] **WebSocket client**: Connect, authenticate, receive updates via buffered channel - `internal/peersync/connector.go`
- [ ] **Peer discovery**: Write/read `.agentboard/server.json` with file locking - `internal/peersync/discovery.go`
- [ ] **Bubble Tea integration**: Channel+Cmd relay pattern for WS messages into TUI - `internal/tui/app.go`
- [ ] **Full state sync**: Send complete board state to newly connected peers - `internal/server/hub.go`
- [ ] **Heartbeat**: 30s ping/pong, 30s timeout for disconnect detection - `internal/server/client.go`
- [ ] **Rate limiting**: Max 60 messages/minute per connection - `internal/server/client.go`
- [ ] **Connection status bar**: Show peer count, role (server/client), connection state - `internal/tui/app.go`
- [ ] **`agentboard serve`**: Persistent server mode (no TUI, background daemon) - `internal/cli/serve.go`
- [ ] **`agentboard peers`**: List connected peers via WS API call - `internal/cli/peers.go`
- [ ] **`--connect` flag**: Manual server address - `internal/cli/root.go`
- [ ] **Authentication on connect**: Token verification during WS handshake - `internal/server/server.go`, `internal/auth/github.go`

**Success criteria:** Two developers on the same repo see each other's changes in real-time. Task moves by one appear instantly on the other's board.

### Phase 4: Resilience (Leader migration + offline mode)

Make it production-ready for teams.

**Tasks:**
- [ ] **Leader election**: Longest-connected peer promotes on leader disconnect - `internal/peersync/leader.go`
- [ ] **State transfer**: Compressed JSON state → new leader via `sync.full` - `internal/peersync/leader.go`
- [ ] **Epoch numbers**: Prevent split-brain with monotonic epoch validation - `internal/server/sequencer.go`, `internal/db/sqlite.go`
- [ ] **Graceful shutdown**: Leader sends `leader.promote` before exit - `internal/server/server.go`
- [ ] **Offline mode**: Queue local changes with idempotency keys, epoch-aware replay on reconnect - `internal/peersync/offline.go`
- [ ] **Reconnection**: Exponential backoff with jitter, auto-reconnect on server change - `internal/peersync/connector.go`
- [ ] **Conflict resolution UX**: Toast notifications for rejected actions ("Task already claimed by @bob") - `internal/tui/notification.go`

**Success criteria:** Leader can disconnect and another peer seamlessly takes over. Offline changes sync correctly on reconnect.

### Phase 5: Polish (Agent CLI + UX + Extended Features)

**Tasks:**
- [ ] **Agent-native CLI subcommands**: `agentboard task {list,create,move,claim,unclaim,get,delete}` with `--json` output - `internal/cli/task.go`
- [ ] **`agentboard status --json`**: Board summary for agent consumption - `internal/cli/status.go`
- [ ] **"My Tasks" filter**: Toggle between all tasks and current user's tasks - `internal/tui/board.go`
- [ ] **Task comments**: Add/view comments on cards with author + timestamp - `internal/tui/task_detail.go`, `internal/db/comments.go`
- [ ] **Diff popup**: Show `git diff` for task's worktree - `internal/tui/board.go`
- [ ] **Terminal resize handling**: Respond to `tea.WindowSizeMsg`, recalculate column widths - `internal/tui/board.go`
- [ ] **Help overlay**: `?` to show key bindings - `internal/tui/app.go`
- [ ] **Review column**: Optional 5th column, launch local agent against PR/branch - `internal/tui/board.go`
- [ ] **Homebrew formula**: Package for `brew install agentboard` - `Formula/agentboard.rb`

**Deferred to post-MVP:**
- Compound column (optional 6th column)
- Theme support (configurable colors)
- Review flow (agent-based PR review)

**Success criteria:** Full agent-native CLI for programmatic access. Polished TUI UX. Installable via Homebrew.

## Edge Cases & Mitigations

| Edge Case | Mitigation |
|---|---|
| Two people claim same task simultaneously | Server-side sequencing: first claim wins, second gets `sync.reject` with toast notification |
| Leader crashes mid-operation | Heartbeat timeout (30s) triggers re-election. Peers queue changes during transition |
| Uncommitted changes on unclaim | Check `git status` in worktree; show confirmation dialog before cleanup |
| tmux session crashes while task "In Progress" | Agent status polling detects dead tmux window; mark agent_status as "error"; user can restart |
| `gh auth` token expired | Detect 401 from GitHub API; show error in TUI with "run `gh auth login` to fix" |
| Network partition (some peers see server) | Partitioned peers enter offline mode; queue changes; sync on reconnect. Epoch numbers prevent stale leaders |
| SQLite corruption during leader migration | Transfer full state as JSON (not raw SQLite file); new leader writes fresh SQLite from received state |
| Agent binary not installed | Check at claim time via `exec.LookPath`; show error with available alternatives |
| Terminal resize during popup | All views handle `tea.WindowSizeMsg`; popups recalculate dimensions |
| Stale `server.json` pointing to dead server | Connection timeout (5s); fall back to becoming new leader |

## Success Metrics

- Real-time board updates visible within 100ms between peers on same network
- Support 5-15 concurrent peers without degradation
- Leader migration completes within 5 seconds
- Offline changes sync correctly on reconnect
- Single binary < 20MB, installs via `brew install`

## Dependencies & Prerequisites

- Go 1.22+ (for modern stdlib features)
- tmux 3.0+ (for dedicated server support)
- gh CLI 2.0+ (for authentication)
- At least one supported AI CLI tool (claude, cursor, aider, etc.)
- Access to shared Git remote (GitHub)

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-23-agentboard-brainstorm.md`

### External References
- [agtx - Terminal Kanban for Coding Agents](https://github.com/fynnfluegge/agtx)
- [Charm Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [kancli - Kanban Tutorial](https://github.com/charm-and-friends/kancli)
- [Lipgloss Layout Example](https://github.com/charmbracelet/lipgloss/blob/master/examples/layout/main.go)
- [Tips for Building Bubble Tea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/)
- [Bubble Tea Realtime Example](https://github.com/charmbracelet/bubbletea/blob/master/examples/realtime/main.go)
- [Multi-View Interfaces in Bubble Tea](https://shi.foo/weblog/multi-view-interfaces-in-bubble-tea)
- [bubbletea-overlay](https://libraries.io/go/github.com%2Frmhubbert%2Fbubbletea-overlay)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)
- [gotmux - Go Library for tmux](https://github.com/GianlucaP106/gotmux)
- [Go Project Structure Best Practices](https://www.glukhov.org/post/2025/12/go-project-structure/)
- [Offline Sync & Conflict Resolution Patterns](https://www.sachith.co.uk/offline-sync-conflict-resolution-patterns-architecture-trade%E2%80%91offs-practical-guide-feb-19-2026/)
- [Split-Brain Prevention](https://www.systemoverflow.com/learn/distributed-primitives/leader-election/preventing-split-brain-quorums-fencing-and-epochs)
- [git-worktree-runner File Copying Patterns](https://deepwiki.com/coderabbitai/git-worktree-runner/7.1-file-copying-patterns)
