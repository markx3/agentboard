---
title: "fix: Enter key on active-agent task opens tmux session directly"
type: fix
date: 2026-02-23
---

# fix: Enter key on active-agent task opens tmux session directly

## Overview

Currently pressing `Enter` on any task always opens the task detail overlay, regardless of agent status. For tasks with an active agent, users must press `Enter` then `v` -- two keystrokes -- to view the agent's tmux session. Since viewing the agent session is the primary action for active tasks, `Enter` should open the tmux session directly (like `v` does), falling back to the task detail overlay for non-active tasks.

## Problem Statement

The `Enter` key handler in `updateBoard()` (`internal/tui/app.go:271-277`) unconditionally opens the task detail overlay:

```go
case key.Matches(msg, keys.Enter):
    if task := a.board.SelectedTask(); task != nil {
        a.detail = newTaskDetail(*task)
        a.detail.SetSize(a.width, a.height)
        a.overlay = overlayDetail
    }
    return a, nil
```

The `v` key handler (`internal/tui/app.go:304-311`) already implements the desired behavior -- opening the agent tmux session with proper in-tmux (split pane) vs outside-tmux (full-screen attach) handling. The fix is to make `Enter` delegate to `viewAgent()` when the task has `AgentStatus == db.AgentActive`.

## Proposed Solution

Add a conditional check in the `Enter` key handler: if the selected task has an active agent, call `viewAgent()` directly; otherwise, open the task detail overlay as before.

### Implementation

**File: `internal/tui/app.go`**

1. **Update `updateBoard()` Enter handler** (lines 271-277):

```go
case key.Matches(msg, keys.Enter):
    if task := a.board.SelectedTask(); task != nil {
        if task.AgentStatus == db.AgentActive {
            return a, a.viewAgent(*task)
        }
        a.detail = newTaskDetail(*task)
        a.detail.SetSize(a.width, a.height)
        a.overlay = overlayDetail
    }
    return a, nil
```

2. **Update help bar** (line 340) -- change `enter:open` to `enter:open/view`:

```go
help := helpStyle.Render(" h/l:columns  j/k:tasks  o:new  m/M:move  a:agent  v:view  A:kill  enter:open/view  x:delete  ?:help  q:quit")
```

3. **Update help overlay** (line 380) -- make Enter behavior explicit:

```
  enter     Open task (view agent if active)
```

**File: `internal/tui/keys.go`**

4. **Update key help string** (line 54):

```go
key.WithHelp("enter", "open/view agent"),
```

## Edge Cases

| Scenario | Behavior | Rationale |
|---|---|---|
| `AgentStatus == AgentActive` | Opens tmux session (split or full-screen) | Primary use case -- this is the fix |
| `AgentStatus == AgentIdle` | Opens task detail overlay | No agent to view, show metadata |
| `AgentStatus == AgentError` | Opens task detail overlay | Agent crashed; user needs detail to see error state and restart via `a` |
| No task selected (empty column) | No-op | `SelectedTask()` returns nil, guard already handles this |
| `v` key on active task | Still opens tmux session | `v` remains unchanged as explicit agent-view key |

## Acceptance Criteria

- [x] `Enter` on task with `AgentStatus == AgentActive` opens agent tmux session (split pane inside tmux, full-screen attach outside tmux)
- [x] `Enter` on task with `AgentStatus == AgentIdle` or `AgentError` opens task detail overlay (unchanged behavior)
- [x] `v` key behavior is completely unchanged
- [x] Help bar text updated to reflect conditional behavior
- [x] Help overlay text updated to reflect conditional behavior
- [x] Key binding help string updated
- [x] Existing task detail overlay `v` handler still works (Enter -> detail for idle tasks, then `v` from detail)

## Files to Modify

| File | Change |
|---|---|
| `internal/tui/app.go:271-277` | Add `AgentActive` check before opening detail overlay |
| `internal/tui/app.go:340` | Update help bar text |
| `internal/tui/app.go:380` | Update help overlay text |
| `internal/tui/keys.go:54` | Update Enter key help string |

## Future Considerations

The SpecFlow analysis identified these items for later work (not part of this fix):

- **Dedicated detail key**: Adding `i` (info/inspect) to always open task detail regardless of agent status. Currently `Enter` is the only way to open detail from the board, so active-agent tasks lose direct detail access. Acceptable for now since the detail overlay is rarely needed when the agent is actively running.
- **Multiplayer guard**: In multiplayer mode, `AgentActive` could be set by a remote peer with no local tmux window. A local window existence check (`tmux.ListWindows()`) would guard against this. Not needed yet since multiplayer tmux viewing is not implemented.
- **Error propagation in `viewAgent()`**: The `tea.ExecProcess` callback at `app.go:482-484` swallows errors. Minor improvement opportunity.

## References

- Agentboard brainstorm: `docs/brainstorms/2026-02-23-agentboard-brainstorm.md`
- Agentboard implementation plan: `docs/plans/2026-02-23-feat-agentboard-collaborative-kanban-tui-plan.md`
- `viewAgent()` implementation: `internal/tui/app.go:469-485`
- Agent status model: `internal/db/models.go:23-29`
- tmux split/attach: `internal/tmux/manager.go:79-95`
