# Agentboard UX Enhancements Brainstorm

**Date:** 2026-02-23
**Status:** Complete
**Author:** Human-in-the-loop review session

## What We're Building

A prioritized set of UX enhancements to make agentboard effective for a human coordinating multiple AI agents. The core problem: **the human-in-control can't tell what agents are doing without entering each tmux session**, and the task management UX has gaps that compound as the board gets busy.

## Why This Matters

Agentboard's value is being the "birds-eye view" for multi-agent collaboration. Right now, the board shows *that* agents are running but not *what* they're doing. A human managing 3-5 concurrent agents spends too much time context-switching into tmux sessions to check progress. The board should be informative enough that you only enter an agent session when you need to intervene.

## Key Decisions

1. **Agent activity reporting via CLI command** — Agents call `agentboard agent status <id> '<message>'` to report what they're doing. Explicit, simple, works with any agent CLI.

2. **Activity displayed on task cards AND detail view** — Truncated summary (~30 chars) on the card for at-a-glance scanning, full message in the detail overlay.

3. **Tab toggles Enter behavior** — Board has two modes: "agent mode" (default, Enter opens tmux session for active tasks) and "detail mode" (Enter always opens task detail). Tab switches between them. Current mode shown in status bar. Default is agent mode to preserve backward compatibility.

4. **Full task editing** — All fields editable after creation (title, description, assignee, branch name, PR URL).

5. **Visibility-first roadmap ordering** — Agent status reporting first, then UX fixes, then board intelligence, then task organization.

6. **Cleanup interleaved** — Dead code removal, stale TODO updates, and CLAUDE.md creation happen alongside feature work, not as a separate phase.

## Phased Roadmap

### Phase 1: Agent Activity Reporting (Highest Priority)

**Problem:** Can't tell what agents are doing without viewing their tmux session.

**Changes:**
- Add `agent_activity` column to tasks table (text, nullable)
- Add `agentboard agent status <id> '<message>'` CLI command
- Update task card rendering to show truncated activity line under title
- Update task detail overlay to show full activity message
- Update Claude runner's system prompt to instruct agents to report activity on major steps
- Update Cursor runner's prompt similarly (all runners should instruct activity reporting)
- Activity auto-clears when agent status changes to idle/completed/error

**Task card mockup:**
```
 CC 3m  Fix auth bug
  Writing unit tests...
```

**Cleanup interleaved:** Add CLAUDE.md with project conventions while touching core files.

### Phase 2: Task Detail & Edit UX

**Problem:** Can't view task details when agent is active (Enter goes to tmux). Can't edit tasks after creation.

**Changes:**
- Add board mode toggle (Tab key): "agent mode" (default) vs "detail mode"
- Show current mode in status bar: `[Agent]` or `[Detail]`
- Agent mode (default): Enter opens tmux session for active tasks, detail for idle tasks (preserves current behavior)
- Detail mode: Enter always opens task detail overlay regardless of agent status
- Add edit capability to task detail overlay (press `e` to enter edit mode)
- Edit mode: all fields editable (title, description, assignee, branch, PR URL)
- Add `agentboard task update <id> --title/--description/--assignee/--branch/--pr-url` CLI command

**Cleanup interleaved:** Remove dead code in packages touched during this phase.

### Phase 3: Board Intelligence

**Problem:** No summary metrics. Easy to miss agent state changes.

**Changes:**
- Add persistent status summary bar (top or bottom of board)
  - Format: `Agents: 3 active | Tasks: 2 brainstorm, 3 in_progress, 1 review | 5 done`
- Add agent completion notifications:
  - Terminal bell (`\a`) on agent completion or error
  - Visual flash/highlight on the task card that changed state

**Cleanup interleaved:** Update stale TODO files (mark resolved ones as resolved).

### Phase 4: Task Organization

**Problem:** No way to search, filter, or express dependencies between tasks.

**Changes:**
- Add task search: `/` key opens search input, filters visible tasks by title
- Add column filter: filter tasks by assignee or agent status
- Add task dependencies:
  - `agentboard task block <id> <blocked-by-id>` CLI command
  - Visual indicator on blocked tasks (dimmed or lock icon)
  - Dependency info in task detail overlay
  - Optional: prevent moving blocked tasks past their blockers

## Resolved Questions

1. **Activity message length limit** — Cap at 200 characters. Truncate if longer. The display layer handles further truncation for card vs detail view.

2. **Activity update frequency** — Agents report on major steps only (starting a new phase, writing code, running tests, creating PR). Not on every tool call. Balances visibility with noise.

3. **Notification preferences** — Start with terminal bell (`\a`) only. No desktop notifications in v1. Simple, universal, no platform-specific code. Can add desktop notifications later.

4. **Dependency visualization** — Blocked tasks appear dimmed with a lock icon on the board. Visually obvious they can't be worked on. Full dependency details in the task detail overlay.

## Open Questions

None — all questions resolved during brainstorm session.

## What We're NOT Building (YAGNI)

- Web UI — agentboard is terminal-native, that's a feature
- Real-time log streaming — viewing the tmux session is sufficient for deep inspection
- Gantt charts or timeline views — keep it simple, Kanban is enough
- Agent-to-agent communication — agents coordinate through the board, not directly
- AI-powered task suggestions — the human decides what to work on
