---
title: "feat: Agentboard UX Enhancements"
type: feat
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-agentboard-ux-enhancements-brainstorm.md
---

# Agentboard UX Enhancements

## Overview

A phased set of UX improvements to make agentboard effective for a human coordinating multiple AI agents. The core problem: the board shows *that* agents are running but not *what* they're doing, and the task management UX has gaps that compound as the board gets busy.

## Problem Statement

A human managing 3-5 concurrent agents spends too much time context-switching into tmux sessions to check progress. Additionally, the Enter key is hijacked by active agents (can't access task detail), tasks can't be edited after creation, there are no board-level metrics, and no way to search or express dependencies.

## Proposed Solution

Four phases, ordered by impact: Agent Activity Reporting → Task Detail & Edit UX → Board Intelligence → Task Organization. Cleanup (dead code, stale TODOs, CLAUDE.md) is interleaved with feature work.

## Technical Approach

### Architecture

All changes stay within the existing architecture:
- `internal/db/` — Schema migration, model, SQL queries
- `internal/cli/` — New CLI subcommands
- `internal/tui/` — Card rendering, overlays, key bindings, mode system
- `internal/agent/` — System prompt updates
- `internal/board/` — Service interface (minimal changes)

**Critical architectural decision:** Add targeted SQL update methods (`UpdateAgentActivity`, `UpdateAgentStatus`) to `internal/db/tasks.go` instead of relying on the full-row `UpdateTask` for agent-originated writes. The TUI's 2.5s reconciliation loop reads tasks and calls `UpdateTask` with the full row, which would silently overwrite any agent activity set between polls.

### Implementation Phases

#### Phase 1: Agent Activity Reporting

The highest-impact change. Agents report what they're doing via CLI; the board displays it.

##### DB Changes

**`internal/db/schema.go`**
- Bump `schemaVersion` from `4` to `5`
- Add `agent_activity TEXT DEFAULT ''` to the `schemaSQL` CREATE TABLE statement
- Add migration constant:
  ```sql
  const migrateV4toV5 = `ALTER TABLE tasks ADD COLUMN agent_activity TEXT DEFAULT '';`
  ```

**`internal/db/sqlite.go`**
- Add `if currentVersion < 5 { ... }` migration block following the existing v3→v4 pattern (transaction-wrapped)

**`internal/db/models.go`**
- Add `AgentActivity string` field to Task struct (after `AgentSpawnedStatus`)

**`internal/db/tasks.go`**
- Add `agent_activity` to `taskColumns` constant
- Add `&t.AgentActivity` to `scanTask()` Scan call
- Add `agent_activity` column and value to `CreateTask()` INSERT
- Add `agent_activity=?` and value to `UpdateTask()` UPDATE
- **New function** `UpdateAgentActivity(ctx, id, activity string) error` — column-specific UPDATE:
  ```go
  func (d *DB) UpdateAgentActivity(ctx context.Context, id, activity string) error {
      _, err := d.conn.ExecContext(ctx,
          `UPDATE tasks SET agent_activity=?, updated_at=? WHERE id=?`,
          activity, time.Now().UTC().Format(time.RFC3339), id)
      return err
  }
  ```

##### Service Layer

**`internal/board/service.go`**
- Add `UpdateAgentActivity(ctx context.Context, id, activity string) error` to the `Service` interface

**`internal/board/local.go`**
- Implement: delegate to `s.db.UpdateAgentActivity(ctx, id, activity)`

##### CLI

**`internal/cli/agent_cmd.go`**
- Add `agentStatusCmd` subcommand:
  - `Use: "status <task-id> <message...>"`
  - `Args: cobra.MinimumNArgs(2)`
  - Joins args[1:] into a single message string
  - Truncates at 200 characters
  - Calls `svc.UpdateAgentActivity(ctx, task.ID, message)`
- Register in `init()`: `agentCmd.AddCommand(agentStatusCmd)`

##### TUI Display

**`internal/tui/task_item.go`**
- Update `Description()` to show activity when present:
  ```go
  func (t taskItem) Description() string {
      if t.task.AgentActivity != "" {
          activity := t.task.AgentActivity
          if len(activity) > 30 {
              activity = activity[:27] + "..."
          }
          if t.task.Assignee != "" {
              return fmt.Sprintf("@%s | %s", t.task.Assignee, activity)
          }
          return activity
      }
      if t.task.Assignee != "" {
          return fmt.Sprintf("@%s", t.task.Assignee)
      }
      return ""
  }
  ```

**`internal/tui/task_detail.go`**
- Add activity display in `View()` after agent info block:
  ```go
  if t.AgentActivity != "" {
      lines = append(lines, fmt.Sprintf("Activity: %s", t.AgentActivity))
  }
  ```

##### Agent Prompt

**`internal/agent/claude.go`**
- Append to `buildClaudeSystemPrompt()` before the return:
  ```
  ACTIVITY REPORTING:
  Periodically update your activity status so the board shows what you're doing:
    agentboard agent status <shortID> "<brief activity description>"
  Update when starting a new major step (reading code, writing implementation, running tests, creating PR).
  Keep messages under 200 characters.
  ```

**`internal/agent/cursor.go`**
- Add equivalent activity reporting instructions to the Cursor runner's prompt

##### Activity Auto-Clear

**`internal/tui/app.go`** (in `reconcileAgents()`)
- When marking agent as `completed` or `error`, also clear activity:
  ```go
  task.AgentActivity = ""
  ```
- When marking agent as `idle` (kill), also clear activity

##### Acceptance Criteria

- [ ] `agentboard agent status <id> "message"` sets activity on the task
- [ ] Activity truncated at 200 chars on write
- [ ] Task cards show truncated activity (~30 chars) as second line
- [ ] Task detail shows full activity message
- [ ] Activity auto-clears when agent completes, errors, or is killed
- [ ] Claude runner instructs agents to report activity
- [ ] Cursor runner instructs agents to report activity
- [ ] DB migration from v4→v5 runs cleanly on existing databases
- [ ] Concurrent activity updates from agents don't get overwritten by TUI reconciliation

---

#### Phase 2: Task Detail & Edit UX

Fix the Enter key conflict and add task editing.

##### Board Mode Toggle

**`internal/tui/keys.go`**
- Add `Tab` key binding:
  ```go
  Tab: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "toggle mode")),
  ```

**`internal/tui/app.go`**
- Add `boardMode` field to `App` struct (type `int`, constants `modeAgent = 0`, `modeDetail = 1`)
- Default: `modeAgent` (preserves backward compatibility)
- Handle Tab in `updateBoard()`: toggle `a.boardMode`
- Modify Enter handler (lines 458-467):
  ```go
  case key.Matches(msg, keys.Enter):
      task := a.board.SelectedTask()
      if task == nil {
          return a, nil
      }
      if a.boardMode == modeAgent && task.AgentStatus == db.AgentActive {
          return a, a.viewAgent(*task)
      }
      a.detail = newTaskDetail(*task)
      a.detail.SetSize(a.width, a.height)
      a.overlay = overlayDetail
      return a, nil
  ```
- Update status bar to show mode: append `[Agent]` or `[Detail]` to status text
- Update help bar to show `tab:mode` binding

##### Task Editing

**`internal/tui/task_detail.go`**
- Add edit mode state:
  ```go
  type taskDetail struct {
      task      db.Task
      width     int
      height    int
      editing   bool
      editField int  // 0=title, 1=description, 2=assignee, 3=branch, 4=pr_url
      inputs    []textinput.Model  // 4 single-line inputs
      textarea  textarea.Model     // 1 multi-line for description
  }
  ```
- `e` key enters edit mode: populates inputs from task fields
- `Tab` cycles between fields (no conflict — Tab in overlay context, not board)
- `Enter` or `Ctrl+S` saves: calls `svc.UpdateTask()` with modified task
- `Esc` cancels edit mode (returns to detail view, not closes overlay)
- `View()` renders editable fields when `editing=true`:
  - Active field highlighted, others dimmed
  - Help bar: `tab:next field  enter:save  esc:cancel`

**`internal/tui/app.go`**
- Pass `svc` (board.Service) reference to `taskDetail` so it can save
- In `updateDetail()`: route key events to edit mode handlers when `a.detail.editing`

##### CLI Update Command

**`internal/cli/task_cmd.go`**
- Add `taskUpdateCmd`:
  - `Use: "update <task-id>"`
  - Flags: `--title`, `--description`, `--assignee`, `--branch`, `--pr-url`
  - Gets task, applies non-empty flags, calls `svc.UpdateTask()`
- Register in `init()`

##### Acceptance Criteria

- [ ] Tab toggles between agent mode and detail mode
- [ ] Status bar shows current mode
- [ ] In detail mode, Enter opens task detail for all tasks (active or idle)
- [ ] In agent mode, Enter opens tmux for active tasks (preserves current behavior)
- [ ] `e` in detail overlay enters edit mode
- [ ] All fields editable: title, description, assignee, branch, PR URL
- [ ] Tab cycles between edit fields
- [ ] Enter/Ctrl+S saves, Esc cancels (returns to detail, not closes overlay)
- [ ] `agentboard task update <id> --title "new"` works from CLI
- [ ] Editing a task while agent is active doesn't interfere with agent

---

#### Phase 3: Board Intelligence

Summary metrics and notifications for state changes.

##### Status Summary Bar

**`internal/tui/board.go`**
- Add `summaryBar() string` method:
  - Count tasks per status and agents per status from loaded tasks
  - Format: `Agents: 3 active | Tasks: 2 brainstorm, 3 in_progress, 1 review | 5 done`

**`internal/tui/app.go`**
- Add summary bar above the board in `View()`:
  ```go
  summaryBar := summaryBarStyle.Render(a.board.summaryBar())
  mainView := lipgloss.JoinVertical(lipgloss.Left, summaryBar, boardView, statusBar, help)
  ```

##### Agent Completion Notifications

**`internal/tui/app.go`**
- Add `prevAgentStates map[string]db.AgentStatus` to `App` struct
- In `reconcileAgents()`, after loading tasks:
  - Compare each task's `AgentStatus` with `prevAgentStates[task.ID]`
  - If transitioned to `completed` or `error`: fire notification
  - Update `prevAgentStates`
- Notification = terminal bell (`\a` via `tea.Printf`) + notification bar message
- Visual flash: set a `flashTaskID` + `flashUntil` timestamp on the task card
  - In `task_item.go`, check if task ID matches flash and render with highlight style

##### Acceptance Criteria

- [ ] Summary bar shows at top of board with task/agent counts
- [ ] Summary updates on every refresh cycle
- [ ] Terminal bell rings once when an agent completes or errors
- [ ] Bell does NOT repeat on subsequent poll cycles
- [ ] Task card briefly highlights (flash) on state transition
- [ ] Notification bar shows "Agent completed: <task title>" briefly

---

#### Phase 4: Task Organization

Search, filter, and dependency management.

##### Task Search

**`internal/tui/app.go`**
- Add `searching bool` and `searchInput textinput.Model` to App
- `/` key activates search mode (shows input at bottom)
- Input filters visible tasks across all columns by title substring match
- `Esc` clears search and exits search mode
- `Enter` closes input but keeps filter active

**`internal/tui/column.go`**
- Add `filterFn func(db.Task) bool` that columns use when building visible items

##### Task Dependencies

**`internal/db/schema.go`**
- New table in schema v6:
  ```sql
  CREATE TABLE IF NOT EXISTS task_dependencies (
      task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
      blocked_by_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
      PRIMARY KEY (task_id, blocked_by_id),
      CHECK(task_id != blocked_by_id)
  );
  ```

**`internal/db/tasks.go`**
- New functions: `AddDependency`, `RemoveDependency`, `GetBlockedBy`, `GetBlocking`
- Cycle detection: before adding dependency, check if it would create a cycle (BFS/DFS from blocked_by_id looking for task_id)

**`internal/cli/task_cmd.go`**
- Add `taskBlockCmd`: `agentboard task block <id> <blocked-by-id>`
- Add `taskUnblockCmd`: `agentboard task unblock <id> <blocked-by-id>`

**`internal/tui/task_item.go`**
- Blocked tasks rendered dimmed with lock prefix: `🔒 Task title`

**`internal/tui/task_detail.go`**
- Show "Blocked by: <task titles>" and "Blocking: <task titles>" sections

##### Acceptance Criteria

- [x] `/` opens search, filters tasks by title across all columns
- [x] Esc clears search filter
- [x] `agentboard task block <a> <b>` creates a dependency
- [x] `agentboard task unblock <a> <b>` removes a dependency
- [x] Circular dependencies are rejected with clear error message
- [x] Blocked tasks appear dimmed with lock icon on board
- [x] Detail view shows dependency relationships
- [x] Self-blocking is rejected (`task block X X`)

## Dependencies & Prerequisites

- No external dependencies — all changes use existing libraries
- Phase 2 depends on Phase 1 (activity display is part of the improved detail view)
- Phase 3 is independent of Phase 2 (can be parallelized)
- Phase 4 is independent of Phase 3 (can be parallelized after Phase 2)

## Risk Analysis & Mitigation

| Risk | Impact | Mitigation |
|------|--------|------------|
| Concurrent writes overwrite activity | Data loss | Column-specific `UpdateAgentActivity` SQL method |
| Terminal bell rings every poll cycle | Annoying UX | Track previous agent states, fire once per transition |
| Tab key conflict (board vs edit) | UX confusion | Different contexts: Tab on board = mode, Tab in overlay = field |
| Edit mode + active agent conflict | Data corruption | Edit mode reads fresh task from DB, saves atomically |
| Activity messages too verbose | Noisy cards | 200 char limit on write, 30 char truncation on display |

## References

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-23-agentboard-ux-enhancements-brainstorm.md`
- Agent CLI pattern: `internal/cli/agent_cmd.go:30-65`
- DB migration pattern: `internal/db/schema.go`, `internal/db/sqlite.go:65-143`
- Task card rendering: `internal/tui/task_item.go:16-25`
- Enter key handler: `internal/tui/app.go:458-467`
- System prompt builder: `internal/agent/claude.go:38-77`
- Service interface: `internal/board/service.go:11-21`
