# Agentboard - Collaborative Agentic Task Management TUI

**Date**: 2026-02-23
**Status**: Brainstorm
**Inspired by**: [agtx](https://github.com/fynnfluegge/agtx)

## What We're Building

A terminal-based collaborative Kanban board for managing agentic coding tasks across a team. Think agtx, but multiplayer. Multiple developers connect to a shared project board in real-time, each running their own AI coding agents (Claude Code, Cursor CLI, Antigravity, etc.) in local worktrees, while everyone has visibility into the full project's progress.

### Core Value Proposition

- **Birds-eye view**: Everyone sees the full Kanban board in real-time
- **Local execution, shared visibility**: Worktrees and agent sessions are local; task state is shared
- **Agent-agnostic**: Each developer uses their preferred AI CLI tool
- **Lightweight collaboration**: No heavy project management tool - just a TUI and a WebSocket connection

## Why This Approach

### Tech Stack: Go + Bubble Tea

- **Goroutines** are a natural fit for juggling WebSocket connections, TUI rendering, subprocess management (tmux/agents), and git operations concurrently
- **Single binary** distribution - `brew install agentboard` and done
- **Charmbracelet ecosystem** (Bubble Tea + Lipgloss + Bubbles) provides battle-tested TUI components
- **Elm architecture** (Model-Update-View) maps cleanly to a Kanban board receiving real-time WebSocket events
- Server and client can live in the same codebase, same language

### Sync Model: WebSocket with Peer-Leader

- **Primary**: First person to open a project becomes the server (peer leader). Others connect to them.
- **Discovery**: Leader writes `host:port` to `.agentboard/server.json` (gitignored). Peers on the same repo read this file automatically. Remote peers use `agentboard --connect host:port`.
- **Auto-migration**: If the leader disconnects, another connected peer promotes to server automatically. Clients reconnect seamlessly. New leader takes over the SQLite state.
- **Persistent mode**: For teams that want always-on, run `agentboard serve` as a background daemon on a shared machine. Same protocol, just a dedicated host.
- **Offline resilience**: When no server is reachable, the TUI works in local-only mode. Changes queue up and sync when a peer is available again.
- The WebSocket protocol is identical regardless of who's hosting - the "server" is just a role any instance can take.

### Authentication: GitHub Identity

- Uses `gh auth` token already on the machine
- Server verifies via GitHub API
- Users identified by GitHub username
- Natural fit since the workflow is already GitHub-centric (PRs, branches, worktrees)

## Key Decisions

### 1. Kanban Columns

```
Backlog | Planning | In Progress | Review | Compound | Done
```

- **Compound is optional** per-project configuration
- When enabled, follows the compound-engineering plugin pattern
- Users can trigger `/workflows:compound` from the Compound column but can skip it - not a gate before Done

### 2. Shared vs. Local State

| Shared (via WebSocket) | Local (per machine) |
|---|---|
| Task titles, descriptions, status | Tmux session content |
| Assignee (GitHub username) | Agent process/session |
| Branch name, PR links | Worktree files |
| Column position | Local config overrides |
| Card comments (author + timestamp) | Agent CLI output |
| Agent status per task (idle/active/error) | |

### 3. Task Assignment: Claim-Based

- Tasks start unassigned in Backlog
- Anyone can claim a task by moving it from Backlog to Planning
- Claiming creates the local worktree on the assignee's machine and launches the agent
- Once claimed, only the assignee can move it forward
- "My Tasks" filter shows tasks claimed by the current user
- Tasks can be unclaimed/reassigned (worktree cleaned up on unclaim)

### 4. Agent Configuration: Per-User Global + Per-Project Override

- Each user sets their default agent globally (`~/.agentboard/config.toml`)
- Projects can suggest a preferred agent in project config (`.agentboard/config.toml`)
- Users can override the project suggestion with their personal preference
- Supported agents detected via `which` (same pattern as agtx)

### 5. Review Flow: Local Agent Trigger

- When someone opens a card in the Review column, the app launches their configured CLI tool against that task's PR/branch
- The review runs locally using whatever skills/capabilities the user's agent has (e.g., Claude Code's `/review-pr` skill)
- Review output shown in a popup within the TUI
- Each reviewer uses their own agent - review output is not shared to the board

### 6. Multi-Project: One Project Per Server

- Each project has its own server instance (peer-leader or persistent)
- The TUI connects to one project at a time
- Switch projects by connecting to a different server
- Clean separation of concerns

### 7. Team Scale: Medium (5-15 people)

- Designed for teams of 5-15 developers
- WebSocket protocol should handle concurrent state updates gracefully
- Conflict resolution needed for simultaneous task moves

## Architecture Overview

```
+------------------+     +------------------+     +------------------+
|   Developer A    |     |   Developer B    |     |   Developer C    |
|                  |     |                  |     |                  |
| +==============+ |     | +==============+ |     | +==============+ |
| | agentboard   | |     | | agentboard   | |     | | agentboard   | |
| | TUI (bubbletea)| |   | | TUI          | |     | | TUI          | |
| +==============+ |     | +==============+ |     | +==============+ |
|       |          |     |       |          |     |       |          |
| [WebSocket Client]     | [WebSocket Client]     | [WebSocket Client]
|       |          |     |       |          |     |       |          |
| [tmux + claude]  |     | [tmux + cursor]  |     | [tmux + claude]  |
| [local worktrees]|     | [local worktrees]|     | [local worktrees]|
+--------|---------+     +--------|----------+    +--------|----------+
         |                        |                        |
         +------------+-----------+------------------------+
                      |
              [WebSocket Server]
              (peer-leader or
               dedicated host)
                      |
              [Shared State]
              - Task board
              - Assignments
              - PR links
              - Progress
```

## Task Lifecycle

```
                    anyone claims
  [Backlog] ----------------------> [Planning]
  (unassigned)                      (assignee's agent creates plan)
                                         |
                                         | assignee approves plan
                                         v
                                    [In Progress]
                                    (agent implements)
                                         |
                                         | assignee moves to review
                                         v
                                    [Review]
                                    (team reviews via their agents)
                                         |
                              +----------+----------+
                              |                     |
                              v                     v
                         [Compound]              [Done]
                         (optional:              (cleanup: worktree
                          document               removed, branch
                          learnings)             preserved)
```

## Key UX Patterns (inherited from agtx, adapted)

- **Vim-style navigation**: `h/l` columns, `j/k` rows
- **Quick actions**: `o` new task, `m` move right, `Enter` open, `d` diff, `x` delete
- **Shell popup**: Live tmux pane viewer for active tasks (local only)
- **Task search**: `/` for fuzzy search across all tasks
- **User filter**: Toggle between "All Tasks" and "My Tasks" view
- **Status indicators**: Show which tasks have active agents running
- **Connection status**: Show peer count and server/client role in the status bar

## CLI Commands

```bash
# Start TUI for current repo (becomes peer-leader if first)
agentboard

# Start as dedicated server (persistent mode)
agentboard serve

# Connect to a specific server
agentboard --connect <host:port>

# Initialize project config
agentboard init

# Show connected peers
agentboard peers
```

## Configuration

### Global (`~/.agentboard/config.toml`)

```toml
[user]
# Populated from gh auth
github_username = "auto-detected"

[agent]
default = "claude"  # or "cursor", "antigravity", etc.

[theme]
# Color customization (hex values, same pattern as agtx)
border = "#4a9a8a"
text = "#d4d4d4"
accent = "#e6b450"
```

### Project (`.agentboard/config.toml`)

```toml
[project]
name = "my-project"

[agent]
preferred = "claude"  # suggestion, not enforcement

[worktree]
copy_files = [".env", ".env.local"]
init_script = "npm install"

[columns]
compound = true  # enable/disable the optional Compound column
```

## Prerequisites

All team members need:
- **tmux** installed (agent sessions run inside tmux windows)
- **gh** CLI installed and authenticated (`gh auth login`)
- Access to the same git remote (GitHub repo)
- At least one supported AI CLI tool installed (claude, cursor, etc.)

## Resolved Questions

1. **Conflict resolution**: Server-side sequencing. The server assigns a sequence number to every state change. First to arrive wins. Second gets rejected with a "task already moved" notification. Simple and deterministic.
2. **Persistence**: SQLite (same pattern as agtx). Server writes state to SQLite. Survives restarts. During peer-leader migration, the new leader receives full state from the outgoing leader (or replays from the SQLite DB on that machine).
3. **Notifications**: In-TUI toast notifications. A notification bar at the bottom of the TUI shows recent events (e.g., "@alice moved Task X to Review"). Non-intrusive, no OS integration needed.
4. **Task comments/discussion**: Lightweight comments on cards. Simple text comments visible to all - flat list with author + timestamp. Good for quick status notes like "blocked on API key" or "PR ready". Not threaded.
5. **Peer discovery**: Repo config file + manual fallback. Leader writes `host:port` to `.agentboard/server.json` (gitignored). Peers on the same machine read it automatically. Remote peers use `--connect host:port`.
6. **Worktree lifecycle**: Created on claim (Backlog -> Planning). The assignee's machine creates the local worktree and launches the agent. Only the assignee has a local worktree for a given task.
