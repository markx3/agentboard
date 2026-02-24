# Ralph Loop Race Conditions When Moving Tasks Between Columns

**Date:** 2026-02-23
**Status:** Ready for planning

## What We're Building

Fixes for 5 race conditions that cause agents to get stuck in an infinite respawn cycle when tasks are moved between columns while a ralph loop is active. The core symptom: the agent gets killed, respawned, re-enters the ralph loop with a different stage prompt, rapidly moves the task again, triggers another respawn, and loops forever — requiring the user to manually abort.

## Why This Approach

Point fixes at each race condition site rather than a centralized lifecycle refactor. The codebase is young and the fixes are well-isolated. This minimizes risk per change and keeps the PR reviewable.

## Key Decisions

1. **Kill + respawn is the desired behavior** when a user moves a task with an active agent. The auto-respawn feature stays.

2. **Ralph loop is deactivated on respawn.** When an agent is killed and respawned in a new column, set `active: false` in the ralph-loop state file. The new agent runs once without looping. User can manually re-enable if needed.

3. **Fix all 5 race conditions** via targeted point fixes:

### Fix 1: Deactivate ralph loop on respawn
- **Where:** `respawnAgent()` in `internal/tui/app.go`
- **What:** Before spawning the new agent, find and deactivate the ralph-loop state file in the task's worktree directory.

### Fix 2: Set `AgentSpawnedStatus` at spawn time
- **Where:** `Spawn()` in `internal/agent/spawn.go`
- **What:** Add `task.AgentSpawnedStatus = string(task.Status)` before calling `svc.UpdateTask()`. This was in the design but missing from implementation.

### Fix 3: Update agent metadata on CLI-initiated moves
- **Where:** `runTaskMove()` in `internal/cli/task_cmd.go`
- **What:** After `svc.MoveTask()`, also clear/update agent lifecycle fields (`AgentStatus`, `AgentSpawnedStatus`) so reconciliation has accurate data.

### Fix 4: Fix reconciliation baseline comparison
- **Where:** `reconcileAgents()` in `internal/tui/app.go`
- **What:** Compare against `AgentSpawnedStatus` (captured at spawn time) instead of `columnAtDetection` (captured at window death). This correctly identifies whether the agent moved the task.

### Fix 5: Add destination-check guard in respawnAgent
- **Where:** `respawnAgent()` in `internal/tui/app.go`
- **What:** Before respawning, fetch fresh task from DB and verify the task's current status actually differs from what the agent was already working on. Skip respawn if the task is already at the target column.

## Open Questions

None — all key decisions resolved during brainstorming.
