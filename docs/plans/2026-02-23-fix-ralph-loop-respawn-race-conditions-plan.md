---
title: "fix: Prevent ralph loop infinite respawn on task column move"
type: fix
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-ralph-loop-race-conditions-brainstorm.md
---

# fix: Prevent ralph loop infinite respawn on task column move

## Overview

When a task with an active ralph-loop-enabled agent is moved between kanban columns, the agent enters an infinite kill-respawn cycle. The new agent inherits the old ralph-loop state file (`active: true`), re-enters the loop with a different stage prompt, rapidly moves the task again, and triggers another respawn — looping forever until the user manually aborts.

This plan addresses 5 race conditions with targeted point fixes in 3 files.

## Problem Statement

The ralph loop is a Claude Code plugin that prevents session exit and feeds the same prompt back, creating an iterative development loop. Its state lives in `.claude/ralph-loop.local.md` inside the agent's working directory. When agentboard kills and respawns an agent (the desired behavior on column move), the new agent process inherits this state file and gets trapped.

**Reproduction:**
1. Spawn a Claude agent on a task in the "planning" column
2. The agent activates a ralph loop (via `/ralph-loop`)
3. Move the task to "in_progress" using `m` in the TUI
4. Agent is killed and respawned — but immediately re-enters the ralph loop
5. The ralph loop feeds the prompt, agent moves the task again
6. Infinite cycle — user must abort

**Root causes:**
1. Ralph loop state file is not deactivated on respawn
2. `AgentSpawnedStatus` is never set, breaking reconciliation logic
3. CLI-initiated moves don't update agent metadata
4. Reconciliation compares against wrong baseline (`columnAtDetection` vs `AgentSpawnedStatus`)
5. No guard prevents redundant respawns when task is already at target column

## Technical Approach

### Architecture

All 5 fixes ship atomically in one PR. The fixes are independent at the code level but Fix 4 and Fix 5 depend on Fix 2 populating `AgentSpawnedStatus` to function correctly.

```
Fix 1 (ralph loop deactivation) ─── independent, highest impact
Fix 2 (set AgentSpawnedStatus)  ─── prerequisite for Fix 4 & 5
Fix 3 (CLI move metadata)       ─── independent
Fix 4 (reconciliation baseline) ─── depends on Fix 2
Fix 5 (respawn guard)           ─── depends on Fix 2
```

### Files Modified

| File | Fixes | Changes |
|------|-------|---------|
| `internal/agent/spawn.go` | Fix 1, Fix 2 | Add `DeactivateRalphLoop()` helper, set `AgentSpawnedStatus` + `AgentStartedAt` in `Spawn()` |
| `internal/tui/app.go` | Fix 1, Fix 4, Fix 5 | Call deactivation in `respawnAgent()`, fix reconciliation comparison, add respawn guard |
| `internal/cli/task_cmd.go` | Fix 3 | Update agent metadata after CLI-initiated move |
| `internal/agent/spawn_test.go` | Tests | Test `DeactivateRalphLoop()`, test `AgentSpawnedStatus` is set |

### Implementation Phases

#### Phase 1: Ralph loop deactivation helper + spawn metadata (Fixes 1 & 2)

**Fix 1 — `DeactivateRalphLoop()` in `internal/agent/spawn.go`**

Add a new exported function that finds and deactivates the ralph-loop state file in a task's worktree directory. The path is `<TaskSlug(task.Title)>/.claude/ralph-loop.local.md`. The function modifies the YAML frontmatter, replacing `active: true` with `active: false`.

```go
// internal/agent/spawn.go

// DeactivateRalphLoop sets active: false in the ralph-loop state file
// for the given task's worktree. This prevents a respawned agent from
// inheriting an active loop. No-op if the file does not exist.
func DeactivateRalphLoop(task db.Task) error {
    stateFile := filepath.Join(TaskSlug(task.Title), ".claude", "ralph-loop.local.md")
    data, err := os.ReadFile(stateFile)
    if err != nil {
        return nil // No state file = no ralph loop to deactivate
    }
    updated := strings.Replace(string(data), "active: true", "active: false", 1)
    return os.WriteFile(stateFile, []byte(updated), 0644)
}
```

Design decisions:
- Placed in `agent` package alongside `Spawn()` since it's agent lifecycle logic
- Returns `nil` (not an error) when file doesn't exist — no ralph loop means nothing to deactivate
- Uses simple string replacement rather than a YAML parser (the frontmatter format is fixed and well-known from the plugin)
- Not conditional on agent type — if a non-Claude agent happens to have this file, deactivating it is harmless

**Fix 2 — Set `AgentSpawnedStatus` + `AgentStartedAt` in `Spawn()`**

In `internal/agent/spawn.go`, add two lines before the `UpdateTask` call:

```go
// internal/agent/spawn.go, inside Spawn(), before svc.UpdateTask()

task.AgentName = runner.ID()
task.AgentStatus = db.AgentActive
task.AgentSpawnedStatus = string(task.Status) // NEW: record column at spawn time
task.AgentStartedAt = time.Now().UTC().Format(time.RFC3339) // NEW: record spawn time
```

This records which column the agent was spawned in, enabling reconciliation (Fix 4) and the respawn guard (Fix 5) to make correct decisions.

- [x] Add `DeactivateRalphLoop()` function to `internal/agent/spawn.go`
- [x] Add `"os"`, `"path/filepath"`, `"time"` imports to `internal/agent/spawn.go`
- [x] Set `task.AgentSpawnedStatus` and `task.AgentStartedAt` in `Spawn()` before `UpdateTask()`
- [x] Add unit tests for `DeactivateRalphLoop()` (file exists, file missing, file without active flag)

#### Phase 2: Call deactivation from respawnAgent + respawn guard (Fixes 1 call site & 5)

**Fix 1 call site — `respawnAgent()` in `internal/tui/app.go`**

After fetching the fresh task and before calling `agent.Spawn()`, deactivate the ralph loop:

```go
// internal/tui/app.go, inside respawnAgent(), after GetTask and runner lookup

// Deactivate any active ralph loop so the new agent runs once without looping
_ = agent.DeactivateRalphLoop(*task)
```

The error is intentionally discarded — deactivation is best-effort. A failure here (e.g., permission error) should not block the respawn. The fix still breaks the infinite cycle because even if deactivation fails once, the combination of Fix 5's guard prevents the cascade.

**Fix 5 — Destination-check guard in `respawnAgent()`**

Before spawning, verify the task isn't already at the column the agent was spawned for:

```go
// internal/tui/app.go, inside respawnAgent(), after GetTask

// Guard: skip respawn if agent was already spawned for this column
if task.AgentSpawnedStatus == string(task.Status) {
    return notifyMsg{text: "Agent already working on this column — skipping respawn"}
}
```

This prevents redundant respawns when:
- A task is moved back to its original column
- A concurrent move puts the task back where it started
- The agent has already completed and the reconciliation hasn't caught up yet

- [x] Add `agent.DeactivateRalphLoop(*task)` call in `respawnAgent()` after runner lookup
- [x] Add destination-check guard in `respawnAgent()` before `agent.Spawn()` call
- [ ] Remove unused `newStatus` parameter from `respawnAgent()` (clean up — the guard uses DB state, not the parameter)

Wait — `newStatus` is passed from `taskMovedMsg` handler. Let me reconsider: keep the parameter for now since it's used in the message handler. The guard compares `task.AgentSpawnedStatus` (where agent was spawned) against `task.Status` (where task is now), which is the correct comparison.

- [x] Add `agent.DeactivateRalphLoop(*task)` call in `respawnAgent()` after runner lookup
- [x] Add destination-check guard in `respawnAgent()` before `agent.Spawn()` call

#### Phase 3: Fix reconciliation baseline (Fix 4)

**Fix 4 — `reconcileAgents()` in `internal/tui/app.go`**

Change the comparison on line 278 from `columnAtDetection` to `AgentSpawnedStatus`, with a fallback for tasks spawned before this fix:

```go
// internal/tui/app.go, inside reconcileAgents(), grace period elapsed section

// Determine baseline: prefer AgentSpawnedStatus (set at spawn time),
// fall back to columnAtDetection (set when window death was detected)
baseline := db.TaskStatus(freshTask.AgentSpawnedStatus)
if baseline == "" {
    baseline = pending.columnAtDetection
}

if freshTask.ResetRequested {
    // ... existing reset logic
} else if freshTask.Status != baseline {
    // Task moved to a new column — agent completed successfully
    freshTask.AgentStatus = db.AgentCompleted
    // ...
} else {
    // Task still in same column — agent crashed/failed
    freshTask.AgentStatus = db.AgentError
    // ...
}
```

The fallback ensures backward compatibility: tasks spawned before Fix 2 (with empty `AgentSpawnedStatus`) use the old `columnAtDetection` behavior, which is correct for the non-ralph-loop case.

- [x] Add `baseline` variable with fallback logic in `reconcileAgents()`
- [x] Replace `pending.columnAtDetection` with `baseline` in the comparison
- [x] Keep `columnAtDetection` in `pendingRecon` struct (used as fallback)

#### Phase 4: CLI move metadata update (Fix 3)

**Fix 3 — `runTaskMove()` in `internal/cli/task_cmd.go`**

After `svc.MoveTask()`, fetch the task and update agent lifecycle metadata if an agent is active:

```go
// internal/cli/task_cmd.go, inside runTaskMove(), after svc.MoveTask()

// Update agent metadata so reconciliation has accurate baseline
task, err := svc.GetTask(context.Background(), fullID)
if err == nil && task.AgentStatus == db.AgentActive {
    task.AgentSpawnedStatus = string(newStatus)
    svc.UpdateTask(context.Background(), task)
}
```

This tells the reconciliation system: "the agent moved itself to this column." When reconciliation later checks whether the task moved, it compares against this updated `AgentSpawnedStatus` and correctly concludes the agent completed its stage (rather than marking it as errored because the column changed).

Only updates when `AgentStatus == AgentActive` to avoid corrupting metadata for non-agent-initiated moves.

- [x] Add `GetTask` + conditional `UpdateTask` after `MoveTask` in `runTaskMove()`
- [x] Import `context` if not already imported (already imported)

#### Phase 5: Tests

Add tests to `internal/agent/spawn_test.go`:

```go
func TestDeactivateRalphLoop(t *testing.T) {
    // Test 1: File exists with active: true → sets active: false
    // Test 2: File does not exist → returns nil (no error)
    // Test 3: File exists with active: false → no change
}
```

These tests use `t.TempDir()` to create isolated worktree directories with mock state files.

Also verify `AgentSpawnedStatus` is set by checking the task struct passed to the service mock in spawn logic. However, since `Spawn()` requires tmux and a real service, the most practical approach is to test the new `DeactivateRalphLoop()` function in isolation and verify the `Spawn()` changes via code review.

- [x] Add `TestDeactivateRalphLoop` with 3 subtests (exists+active, missing, exists+inactive)
- [x] Run `go test ./internal/agent/...` to verify all tests pass

## Acceptance Criteria

- [ ] Moving a task with an active ralph-loop agent does NOT cause an infinite respawn cycle
- [ ] The ralph-loop state file is deactivated (`active: false`) in the task's worktree after respawn
- [ ] `AgentSpawnedStatus` is set to the current column name when an agent is spawned
- [ ] CLI-initiated moves (`agentboard task move`) update `AgentSpawnedStatus` when an agent is active
- [ ] Reconciliation correctly identifies completed vs crashed agents using `AgentSpawnedStatus`
- [ ] Respawn is skipped when the task is already at the agent's spawned column
- [ ] All existing tests pass (`go test ./...`)
- [ ] New tests cover `DeactivateRalphLoop()` function
- [ ] Non-ralph-loop agents (no state file) are unaffected by Fix 1
- [ ] Tasks spawned before this fix (empty `AgentSpawnedStatus`) use fallback behavior in reconciliation

## Dependencies & Risks

**Risks:**
- The ralph-loop state file format is owned by the external `claude-plugins-official` plugin. If the plugin changes the frontmatter format, Fix 1's string replacement could silently fail. Mitigation: the replacement is a simple `active: true` → `active: false` swap, which is robust against additional fields.
- SQLite concurrent writes from TUI + CLI are handled by SQLite's WAL mode, but there's no application-level locking. The 5-second grace period provides a buffer. For a future improvement, a mutex on per-task agent lifecycle operations would be safer.

**Not in scope:**
- Mutual exclusion on agent lifecycle operations (future improvement)
- The unused `AgentStartedAt` timeout detection (setting it is included, using it for staleness detection is not)
- Removing dead `columnAtDetection` field from `pendingRecon` struct (kept as fallback)
- Deactivating ralph loop on manual kill + respawn (only auto-respawn on column move)

## References & Research

### Internal References

- Brainstorm: `docs/brainstorms/2026-02-23-ralph-loop-race-conditions-brainstorm.md`
- Agent spawn: `internal/agent/spawn.go:42` — `Spawn()` function
- Respawn: `internal/tui/app.go:733` — `respawnAgent()` function
- Reconciliation: `internal/tui/app.go:227` — `reconcileAgents()` function
- CLI move: `internal/cli/task_cmd.go:172` — `runTaskMove()` function
- Ralph loop setup: `~/.claude/plugins/marketplaces/claude-plugins-official/plugins/ralph-loop/scripts/setup-ralph-loop.sh`
- Ralph loop stop hook: `~/.claude/plugins/marketplaces/claude-plugins-official/plugins/ralph-loop/hooks/stop-hook.sh`
- State file format: YAML frontmatter with `active: true/false`, `iteration: N`, `max_iterations: N`, `completion_promise: "..."`, `started_at: "..."`
