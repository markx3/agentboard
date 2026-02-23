---
title: "feat: Add multi-agent CLI support"
type: feat
date: 2026-02-23
brainstorm: docs/brainstorms/2026-02-23-multi-agent-cli-brainstorm.md
---

# feat: Add Multi-Agent CLI Support

## Overview

Refactor the agent spawning system to support multiple AI agent CLIs through an `AgentRunner` interface. Initial support for **Claude Code** (`claude`) and **Cursor CLI** (`agent`). Users select which agent to use per-task via a TUI popup overlay.

## Problem Statement / Motivation

Agentboard claims to be agent-agnostic (README lists Claude, Cursor, Antigravity as supported), but the implementation is entirely hardcoded to Claude Code. The agent binary name, CLI flags, prompt format, and even the string `"claude"` are hardcoded in `internal/agent/spawn.go`. Users cannot use any other AI coding agent.

## Proposed Solution

Introduce an `AgentRunner` Go interface with per-agent implementations, a registry for auto-detection, and a TUI picker overlay. Each runner encapsulates its CLI's quirks (command format, prompt strategy, detection).

## Technical Approach

### Architecture

```
internal/agent/
├── runner.go          # AgentRunner interface + SpawnOpts + registry
├── claude.go          # ClaudeRunner implementation
├── cursor.go          # CursorRunner implementation
├── spawn.go           # Refactored Spawn/Kill (uses runners)
└── spawn_test.go      # Tests for command building

internal/tui/
├── agent_picker.go    # New overlay component
├── app.go             # Updated spawn flow + new overlay type
├── task_item.go       # Agent name indicator in board view
└── keys.go            # Help text update
```

### Key Design Decisions

**1. AgentRunner Interface**

```go
// internal/agent/runner.go

type AgentRunner interface {
    ID() string                        // Canonical DB identifier (e.g., "claude", "cursor")
    Name() string                      // Display name (e.g., "Claude Code", "Cursor")
    Binary() string                    // Executable name for PATH lookup
    Available() bool                   // Is this agent detected and verified?
    BuildCommand(opts SpawnOpts) string // Build the full shell command
}

type SpawnOpts struct {
    WorkDir  string
    Task     db.Task    // Full task for prompt building
    ExePath  string     // Absolute path to agentboard binary
}
```

- `ID()` returns the canonical string stored in `task.AgentName` (e.g., `"claude"`, `"cursor"`)
- `Name()` is for display only (TUI, detail view)
- `BuildCommand()` receives the full task so each runner can build agent-appropriate prompts
- Each runner handles its own prompt strategy internally

**2. Registry**

```go
// internal/agent/runner.go

var runners = []AgentRunner{
    &ClaudeRunner{},
    &CursorRunner{},
}

func AvailableRunners() []AgentRunner    // Filters by Available(), cached at TUI startup
func GetRunner(id string) AgentRunner    // Lookup by ID for respawn
```

- Detection runs once at TUI startup, results cached in the `App` struct
- `CursorRunner.Available()` runs `agent --version` with a 2-second timeout and checks output contains "cursor"
- `ClaudeRunner.Available()` uses `exec.LookPath("claude")` only

**3. Prompt Strategy per Runner**

- **ClaudeRunner**: Uses `--append-system-prompt` for task context (stage guidance, `agentboard task move` commands) + initial prompt with `/workflows:*` slash commands. Same behavior as today.
- **CursorRunner**: Packs all context into a single `agent "prompt"` argument. Uses plain-language instructions instead of Claude-specific `/workflows:*` commands. Includes `agentboard task move` commands.

**4. Agent Persistence**

- `AgentName` is **preserved after kill** — only cleared on unclaim. This enables:
  - Respawn with same agent type
  - "Last used agent" display in task detail
  - Pre-selection in the picker popup
- `AgentName` stores the runner's `ID()` value (e.g., `"claude"`, `"cursor"`)

**5. Auto-Respawn Behavior**

When a task auto-respawns (column move or reset request):
- Look up `task.AgentName` → `GetRunner(id)`
- If runner is still available: respawn with same agent
- If runner is no longer available: set `AgentStatus = error`, show notification

**6. Working Directory**

- **Claude**: Passes `-w <dir>` as CLI flag (current behavior)
- **Cursor**: Passes `dir` to `tmux.NewWindow()` so tmux sets CWD (the `agent` command inherits it)

### Implementation Phases

#### Phase 1: AgentRunner Interface + Implementations

**Files to create:**

- `internal/agent/runner.go` — Interface, SpawnOpts, registry functions

```go
// AgentRunner abstracts an AI agent CLI.
type AgentRunner interface {
    ID() string
    Name() string
    Binary() string
    Available() bool
    BuildCommand(opts SpawnOpts) string
}

type SpawnOpts struct {
    WorkDir string
    Task    db.Task
    ExePath string
}

var runners = []AgentRunner{
    &ClaudeRunner{},
    &CursorRunner{},
}

func AvailableRunners() []AgentRunner {
    var available []AgentRunner
    for _, r := range runners {
        if r.Available() {
            available = append(available, r)
        }
    }
    return available
}

func GetRunner(id string) AgentRunner {
    for _, r := range runners {
        if r.ID() == id {
            return r
        }
    }
    return nil
}
```

- `internal/agent/claude.go` — ClaudeRunner

```go
type ClaudeRunner struct{}

func (c *ClaudeRunner) ID() string     { return "claude" }
func (c *ClaudeRunner) Name() string   { return "Claude Code" }
func (c *ClaudeRunner) Binary() string { return "claude" }

func (c *ClaudeRunner) Available() bool {
    _, err := exec.LookPath("claude")
    return err == nil
}

func (c *ClaudeRunner) BuildCommand(opts SpawnOpts) string {
    sysPrompt := buildClaudeSystemPrompt(opts)
    initialPrompt := buildClaudeInitialPrompt(opts)
    return fmt.Sprintf("claude -w %s --append-system-prompt %s %s",
        shellQuote(opts.WorkDir),
        shellQuote(sysPrompt),
        shellQuote(initialPrompt),
    )
}
```

- `internal/agent/cursor.go` — CursorRunner

```go
type CursorRunner struct{}

func (c *CursorRunner) ID() string     { return "cursor" }
func (c *CursorRunner) Name() string   { return "Cursor" }
func (c *CursorRunner) Binary() string { return "agent" }

func (c *CursorRunner) Available() bool {
    path, err := exec.LookPath("agent")
    if err != nil {
        return false
    }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    out, err := exec.CommandContext(ctx, path, "--version").Output()
    if err != nil {
        return false
    }
    return strings.Contains(strings.ToLower(string(out)), "cursor")
}

func (c *CursorRunner) BuildCommand(opts SpawnOpts) string {
    prompt := buildCursorPrompt(opts)
    return fmt.Sprintf("agent %s", shellQuote(prompt))
}
```

**Files to modify:**

- `internal/agent/spawn.go` — Refactor to use runners

Current `Spawn()` signature: `func Spawn(ctx, svc, task) error`
New signature: `func Spawn(ctx context.Context, svc board.Service, task db.Task, runner AgentRunner) error`

Changes:
- Remove `exec.LookPath("claude")` check (runner handles detection)
- Replace hardcoded command with `runner.BuildCommand(opts)`
- Set `task.AgentName = runner.ID()` instead of `"claude"`
- Pass `WorkDir` to `tmux.NewWindow()` for Cursor (Claude handles it via CLI flag)
- Move `buildSystemPrompt()` and `buildInitialPrompt()` into `claude.go` as `buildClaudeSystemPrompt()` / `buildClaudeInitialPrompt()`
- Add `buildCursorPrompt()` in `cursor.go`

Current `Kill()` change:
- **Stop clearing `AgentName`** — only set `AgentStatus = AgentIdle`, leave `AgentName` intact

Acceptance criteria:
- [x] `AgentRunner` interface defined in `internal/agent/runner.go`
- [x]`ClaudeRunner` in `internal/agent/claude.go` produces same command as current hardcoded version
- [x]`CursorRunner` in `internal/agent/cursor.go` with `--version` verification (2s timeout)
- [x]`Spawn()` accepts a runner parameter and delegates command building
- [x]`Kill()` preserves `AgentName`
- [x]`AvailableRunners()` and `GetRunner()` work correctly
- [x]Unit tests for `BuildCommand()` on both runners

#### Phase 2: TUI Agent Picker Overlay

**Files to create:**

- `internal/tui/agent_picker.go` — New overlay component

```go
type agentPicker struct {
    runners  []agent.AgentRunner
    selected int
    task     db.Task
    width    int
    height   int
}

func newAgentPicker(runners []agent.AgentRunner, task db.Task, w, h int) agentPicker
func (p agentPicker) Update(msg tea.Msg) (agentPicker, tea.Cmd)  // j/k navigate, Enter select, Esc cancel
func (p agentPicker) View() string  // Styled list with highlight
```

Pattern follows existing `taskForm` and `taskDetail` overlays.

**Files to modify:**

- `internal/tui/app.go`:
  - Add `overlayPicker` to `overlayType` enum
  - Add `agentPicker` field to `App` struct
  - Add `availableRunners []agent.AgentRunner` field (cached at startup)
  - Update `spawnAgent()`: detect available runners → if 0: error, if 1: auto-spawn, if 2+: show picker
  - Add `updatePicker()` method in `updateOverlay()` switch
  - Add picker rendering in `View()` → `renderOverlay()` switch
  - Handle `agentSelectedMsg` (new message type): close picker, call `Spawn()` with selected runner
  - Update `respawnAgent()`: use `GetRunner(task.AgentName)` for same-agent respawn; if unavailable, set error
  - Cache `AvailableRunners()` on `Init()`

- `internal/tui/app.go` — Reconciliation respawn:
  - In reconciliation flow where `ResetRequested` is cleared and task is respawned, use `GetRunner(task.AgentName)` instead of calling `agent.Spawn()` with hardcoded Claude
  - If runner not found or not available, set `AgentStatus = AgentError`

Acceptance criteria:
- [x]`overlayPicker` added to overlay enum
- [x]Picker shows when multiple agents available, skipped for single agent
- [x]`j/k` navigation, `Enter` selection, `Esc` cancellation
- [x]Picker follows existing overlay styling (double border, gold)
- [x]Selected agent passed to `Spawn()`
- [x]Auto-respawn uses same agent via `GetRunner(task.AgentName)`
- [x]Unavailable agent at respawn time → `AgentError` status

#### Phase 3: Display Updates

**Files to modify:**

- `internal/tui/task_item.go` — Show agent abbreviation in board view:
  - After the status dot and before elapsed time, show a short agent label
  - Claude: `CC`, Cursor: `Cu` (or first 2 chars of runner name)
  - Example: `● CC 5m Task title here`

- `internal/tui/task_detail.go` — Already shows `AgentName`, verify it uses runner's `Name()` for display (may need `GetRunner(t.AgentName).Name()` lookup)

- `internal/cli/task_cmd.go` — Update CLI `task list` AGENT column:
  - Change from showing just `AgentStatus` to `agent_name (status)`
  - Example: `claude (active)`, `cursor (idle)`, `(idle)` if no agent name

- `internal/tui/app.go` — Update help text:
  - Change `a:agent` to `a:spawn agent` in the bottom help bar
  - Update help overlay text to mention agent selection

- `internal/tui/keys.go` — Update help text for `SpawnAgent` key

Acceptance criteria:
- [x]Board view shows agent type abbreviation for active agents
- [x]Task detail displays runner display name
- [x]CLI `task list` shows agent name alongside status
- [x]Help text updated to reflect agent selection

#### Phase 4: Cleanup

**Files to modify:**

- `internal/cli/init_cmd.go` — Comment out or annotate `[agent] preferred` in config template since it is not read:
  ```toml
  # [agent]
  # preferred = "claude"  # Reserved for future use
  ```

- `internal/agent/spawn.go` — Update package doc comment from "orchestrates Claude Code agent lifecycle" to "orchestrates AI agent lifecycle"

- `internal/board/local.go` — `UnclaimTask()` should continue to clear `AgentName` (unclaim is the one case where it's cleared)

Acceptance criteria:
- [x]Config template cleaned up
- [x]Package docs updated
- [x]`UnclaimTask()` still clears `AgentName`

## Acceptance Criteria

- [x]Claude Code works exactly as before (regression-free)
- [x]Cursor CLI can be spawned on a task if `agent` binary is in PATH and verified
- [x]TUI shows agent picker popup when multiple agents available
- [x]TUI auto-selects when only one agent available
- [x]TUI shows error when no agents available
- [x]Auto-respawn preserves agent type
- [x]Board view shows which agent is running per task
- [x]CLI `task list` shows agent name
- [x]`AgentName` persisted after kill, cleared only on unclaim
- [x]Cursor detection includes `--version` verification with 2s timeout

## Dependencies & Risks

**Dependencies:**
- None — no new Go module dependencies needed (no TOML parser since config reading is out of scope)

**Risks:**
- **Cursor CLI naming**: The `agent` binary name may collide with other tools. Mitigated by `--version` check.
- **Cursor CLI evolving**: The `agent` command syntax may change. The runner encapsulates this, making updates isolated.
- **Prompt length**: Very long task descriptions packed into Cursor's single prompt argument could hit shell limits. Low risk on macOS (ARG_MAX ~262144 bytes).

## References & Research

### Internal References
- Brainstorm: `docs/brainstorms/2026-02-23-multi-agent-cli-brainstorm.md`
- Agent spawn code: `internal/agent/spawn.go` (entire file)
- TUI app: `internal/tui/app.go` (lines 22-29 overlay types, 384-399 keybindings, 579-633 agent lifecycle)
- Overlay pattern: `internal/tui/task_form.go`, `internal/tui/task_detail.go`
- DB model: `internal/db/models.go` (lines 41-45 agent fields)
- Board service: `internal/board/service.go`, `internal/board/local.go`

### External References
- Cursor CLI docs: https://cursor.com/docs/cli/using
- Cursor CLI overview: https://cursor.com/docs/cli/overview
