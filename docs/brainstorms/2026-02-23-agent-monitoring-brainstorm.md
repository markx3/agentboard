---
date: 2026-02-23
topic: agent-monitoring-status-display
---

# Agent Monitoring & Status Display

## What We're Building

A monitoring system that tracks agent lifecycle across stages and displays color-coded status on the kanban board. The system detects whether agents completed successfully, failed, or are still running — replacing the current binary alive/dead check with nuanced state awareness.

The board will show task cards with both status indicator dots and card border/background coloring, plus elapsed time for actively running agents.

## Why This Approach

We considered four detection strategies: artifact detection, tmux output parsing, combined artifact+output, and agent self-reporting via CLI. **Agent self-reporting via CLI** was chosen because:

- Agents already receive instructions to call `agentboard task move` — this extends that pattern
- No brittle output parsing or file-system polling needed
- The agent knows best when it's done — let it declare completion
- Simplest implementation with fewest moving parts

## Key Decisions

### 1. Agent Self-Reporting for Stage Completion
Agents call `agentboard task move <id> <next-stage>` when they complete a stage. The TUI trusts this signal. No output parsing or artifact detection.

### 2. Grace Period for Reconciliation
When a tmux window disappears:
1. Immediately check if the task was moved to a new column
2. If not moved, wait a grace period (~5-10s) and check again
3. Only mark as error if task is still in its original column after the grace period

This handles the race condition where the agent moves the task right before exiting.

### 3. Context Reset via CLI Command
Add `agentboard agent request-reset <task-id>`:
- Agent calls this when it finishes a stage but needs a fresh context for the next one
- Sets a flag on the task in the DB
- TUI detects window gone + reset flag → spawns fresh agent with next-stage prompt
- Versus window gone + no flag + no move → error state

### 4. Flexible Agent Lifecycle
Agents can handle one or multiple stages in a single session:
- **Continue**: Agent moves the task and keeps working in the same tmux window
- **Reset**: Agent calls `request-reset`, exits, TUI respawns with next-stage prompt
- **Only spawn if no agent is attached**: Prevents duplicate agents on the same task

### 5. Five-Color Status System

| Color         | Dot | Meaning                                      |
|---------------|-----|----------------------------------------------|
| **Green**     | `●` | Task in Done column (fully complete)         |
| **Blue/Teal** | `●` | Agent completed current stage successfully   |
| **Yellow**    | `●` | Agent actively running (+ elapsed time)      |
| **Red**       | `✖` | Agent failed/crashed                         |
| **Gray**      | `○` | No agent has run yet                         |

### 6. Dual Visual Treatment
- **Status dots**: Small indicator on each task card (existing pattern, refined colors)
- **Card border/background**: Subtle coloring matching the status — reinforces the dot for at-a-glance recognition

### 7. Elapsed Time for Active Agents Only
Running agents show elapsed time (e.g., `● 8m`). All other states show just the color dot — no timestamps for completed/failed/idle states.

### 8. Absolute Binary Path at Spawn
Fix the `agentboard` PATH issue by resolving `os.Executable()` at spawn time and injecting the absolute path into the agent's system prompt. No install-to-PATH ceremony needed.

## Agent Status State Machine

```
                    spawn
  idle (gray) ──────────────► active (yellow)
                                   │
                         ┌─────────┼──────────┐
                         │         │           │
                    task moved  window dies  request-reset
                    + window    + no move    + exits
                    alive       (after grace)
                         │         │           │
                         ▼         ▼           ▼
                   active      error (red)   reset flag
                   (yellow)                  → respawn
                         │                   → active (yellow)
                    window dies
                    + task moved
                         │
                         ▼
                   completed (blue/teal)
                   or done (green) if in Done column
```

## Open Questions

*None — all questions resolved during brainstorming.*

## Next Steps

→ `/workflows:plan` for implementation details
