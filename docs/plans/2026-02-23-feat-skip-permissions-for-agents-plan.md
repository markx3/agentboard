---
title: "feat: Allow skip-permissions for agents"
type: feat
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-skip-permissions-brainstorm.md
---

# feat: Allow skip-permissions for agents

## Overview

Agents spawned by agentboard run in tmux windows with Claude Code's default interactive permission mode. Permission prompts block silently since no user is watching the tmux pane — agents cannot work autonomously without constant babysitting.

This feature adds a confirmation prompt when spawning agents ("Allow agent to run without asking? y/n") that optionally passes `--dangerously-skip-permissions` to the `claude` CLI. The choice is stored per-task in SQLite so auto-respawns inherit it.

## Problem Statement

The spawn command at `internal/agent/spawn.go:64` is:

```go
cmd := fmt.Sprintf("claude -w %s --append-system-prompt %s %s",
    shellQuote(slug),
    shellQuote(sysPrompt),
    shellQuote(initialPrompt),
)
```

No permission flags are passed. When the agent needs to edit a file or run a shell command, Claude Code prompts for approval. That prompt is invisible in the tmux window — the agent silently blocks until the user attaches with `v` and manually approves.

## Proposed Solution

### Architecture

Add a new `overlayConfirm` overlay type to the TUI. When the user presses `a` to spawn, the confirm overlay appears instead of spawning directly. On `y`, the task's `SkipPermissions` field is set to `true`, persisted to SQLite, and the agent spawns with `--dangerously-skip-permissions`. On `n`, it spawns normally. On `esc`, spawn is cancelled.

Auto-respawns (task column moves) silently inherit the stored flag — no prompt needed since `respawnAgent` reads the fresh task from DB.

### Data Flow

```
User presses 'a'
    → Show overlayConfirm: "Allow agent to run without asking? (y/n)"
    → User presses 'y':
        task.SkipPermissions = true
        svc.UpdateTask(task)  -- persist to SQLite
        agent.Spawn(task)     -- reads task.SkipPermissions, adds flag
    → User presses 'n':
        task.SkipPermissions = false
        svc.UpdateTask(task)  -- persist to SQLite
        agent.Spawn(task)     -- no flag
    → User presses 'esc':
        cancel, return to board

Auto-respawn (task moved):
    respawnAgent → ListTasks → finds task with SkipPermissions from DB
    → agent.Spawn(task) → reads task.SkipPermissions → adds flag if true
```

## Technical Approach

### Phase 1: Database Layer

**`internal/db/schema.go`**

Bump `schemaVersion` from `2` to `3`. Add `skip_permissions INTEGER DEFAULT 0` to the `schemaSQL` tasks table definition. Add a `migrateV2toV3` constant using simple `ALTER TABLE ADD COLUMN` (no CHECK constraint needed for a plain integer default):

```sql
ALTER TABLE tasks ADD COLUMN skip_permissions INTEGER DEFAULT 0;
```

Note: The v1→v2 migration used the full table-rebuild pattern because it added CHECK constraints. A simple `ALTER TABLE ADD COLUMN` with a default value is safe and sufficient here.

**`internal/db/sqlite.go`**

Add a `currentVersion < 3` block in `migrate()` following the existing v1→v2 pattern at line 95. Run `migrateV2toV3` in a transaction and insert version 3 into `schema_version`.

**`internal/db/models.go`**

Add field to the `Task` struct:

```go
SkipPermissions    bool        `json:"skip_permissions"`
```

**`internal/db/tasks.go`**

- Add `skip_permissions` to the `taskColumns` constant
- Add scan in `scanTask`: `var skipPermissions int` → `task.SkipPermissions = skipPermissions != 0` (same pattern as `resetRequested`)
- Add to `CreateTask` INSERT values
- Add to `UpdateTask` SET clause

### Phase 2: Agent Spawn

**`internal/agent/spawn.go`**

Modify the command construction at line 64. If `task.SkipPermissions` is true, insert `--dangerously-skip-permissions` before the positional argument:

```go
skipFlag := ""
if task.SkipPermissions {
    skipFlag = "--dangerously-skip-permissions "
}
cmd := fmt.Sprintf("claude %s-w %s --append-system-prompt %s %s",
    skipFlag,
    shellQuote(slug),
    shellQuote(sysPrompt),
    shellQuote(initialPrompt),
)
```

No signature change needed — `Spawn` already receives the full `db.Task`.

### Phase 3: TUI Confirmation Overlay

**`internal/tui/app.go`**

Add new overlay type and state:

```go
const (
    overlayNone overlayType = iota
    overlayForm
    overlayDetail
    overlayHelp
    overlayConfirm  // new
)
```

Add a `pendingSpawnTask *db.Task` field on `App` to hold the task waiting for confirmation.

**Spawn trigger changes** (both board view at line 384 and detail overlay at line 322):

Instead of calling `a.spawnAgent(*task)` directly, set `a.pendingSpawnTask = &task`, set `a.overlay = overlayConfirm`, and return. The prompt text should indicate the current stored value if re-spawning: `"Allow agent to run without asking? (y/n)"` with `"[currently: yes]"` appended when `task.SkipPermissions` is already true.

**Confirm overlay key handling** (new case in `updateOverlay`):

```
case overlayConfirm:
    'y' → set pendingSpawnTask.SkipPermissions = true, persist, spawn, close overlay
    'n' → set pendingSpawnTask.SkipPermissions = false, persist, spawn, close overlay
    'esc' → close overlay, clear pendingSpawnTask
    anything else → ignore
```

**Confirm overlay rendering** (in `View`):

Render a small centered box using the existing `overlayStyle` (gold border). Contents:

```
Skip permissions?

Allow agent to run commands
without asking for approval.

  y - yes    n - no    esc - cancel
```

### Phase 4: Visual Indicator

**`internal/tui/task_item.go`**

When rendering a task item's status prefix, if `task.SkipPermissions == true` and `task.AgentStatus == db.AgentActive`, append a `!` indicator or use a distinct color to signal the agent is running in dangerous mode.

**`internal/tui/task_detail.go`**

In the detail view, add a line showing `Permissions: skipped` when `SkipPermissions` is true and the agent is active.

## Acceptance Criteria

- [x] Pressing `a` on a task shows a confirmation prompt instead of spawning directly
- [x] Pressing `y` spawns the agent with `--dangerously-skip-permissions` and persists the choice
- [x] Pressing `n` spawns the agent normally (default permissions) and persists the choice
- [x] Pressing `esc` cancels the spawn
- [x] Auto-respawn (task moved to new column) inherits the stored `skip_permissions` flag silently
- [x] The prompt works from both board view and detail overlay
- [x] Fresh databases include the `skip_permissions` column
- [x] Existing v2 databases are migrated to v3 with the new column
- [x] A visual indicator shows when an active agent has skipped permissions
- [x] Re-spawning a task that previously had `skip_permissions=true` shows the current value in the prompt

## Dependencies & Risks

**Migration safety:** Using `ALTER TABLE ADD COLUMN` instead of table-rebuild. This is safe for SQLite when adding a column with a default value. No data loss risk.

**Peer sync:** The `skip_permissions` field will be included in JSON serialization via the `json:"skip_permissions"` tag. When peersync is implemented, this means one user's choice propagates to peers. For now this is acceptable — peersync is not yet active. Future work should consider making this a local-only setting.

**Security:** `--dangerously-skip-permissions` gives the agent full autonomy. The confirmation prompt makes this an explicit, per-spawn choice. No global config to disable it for now — can be added later if needed for team environments.

## References & Research

### Internal References

- Agent spawn command: `internal/agent/spawn.go:64`
- Schema migration pattern: `internal/db/schema.go:51` (v1→v2)
- Migration runner: `internal/db/sqlite.go:65-105`
- Task model: `internal/db/models.go:32-49`
- Task CRUD: `internal/db/tasks.go:18-41` (scanTask), `:81-89` (CreateTask), `:148-163` (UpdateTask)
- TUI overlay system: `internal/tui/app.go:22-29`
- Board spawn handler: `internal/tui/app.go:384-391`
- Detail spawn handler: `internal/tui/app.go:322-326`
- spawnAgent helper: `internal/tui/app.go:579-586`
- respawnAgent helper: `internal/tui/app.go:606-624`
- Keybindings: `internal/tui/keys.go:60-63`
- Brainstorm: `docs/brainstorms/2026-02-23-skip-permissions-brainstorm.md`

### External References

- Claude Code `--dangerously-skip-permissions` flag: bypasses all permission checks for sandboxed environments
