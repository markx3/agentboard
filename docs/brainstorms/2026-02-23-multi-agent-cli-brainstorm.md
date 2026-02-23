# Multi-Agent CLI Support Brainstorm

**Date:** 2026-02-23
**Status:** Draft
**Task:** Add support for multiple AI Agent CLIs (agentboard task 8f227006)

## What We're Building

Agentboard currently only supports Claude Code despite claiming to be agent-agnostic. We're adding real support for multiple AI agent CLIs, starting with **Claude Code** and **Cursor CLI**, so users can choose which agent to spawn per-task via a TUI popup.

### Core Capabilities

1. **AgentRunner interface** — A Go interface that abstracts how each agent CLI is invoked
2. **Per-agent implementations** — `ClaudeRunner` and `CursorRunner` with agent-specific command building
3. **Agent registry** — Auto-detects available agents via PATH, provides list to TUI
4. **TUI agent picker** — Inline list popup when spawning an agent, showing only available agents
5. **Unified prompt strategy** — Each agent embeds task context (title, description, stage guidance) into its initial prompt, adapted to the CLI's capabilities

## Why This Approach

**AgentRunner Interface** was chosen over config-driven templates (too complex, error-prone) and a simple switch statement (unmaintainable as agent count grows). The interface approach gives:

- Clean separation of concerns — each agent's quirks are encapsulated
- Easy to add new agents — implement the interface, register it
- Testable — mock runners for unit tests
- Consistent with the project's existing interface-driven patterns (e.g., `board.Service`)

## Key Decisions

### 1. Agent Selection: Per-Task via TUI Popup
- When user presses the spawn keybind, an inline popup lists detected agents
- User picks with arrow keys, Enter confirms
- No global default needed — keeps it simple

### 2. Context Passing: Embed in Initial Prompt
- All agents receive task context (title, ID, description, stage guidance, `agentboard task move` commands) via the initial prompt
- Claude Code additionally uses `--append-system-prompt` for richer integration
- Cursor CLI gets everything packed into the `agent "prompt"` argument
- This avoids generating temporary rule files or modifying project config

### 3. Agent CLI Details

| Agent | Binary | Detection | Command Pattern |
|-------|--------|-----------|-----------------|
| Claude Code | `claude` | `exec.LookPath("claude")` | `claude -w <dir> --append-system-prompt <sysPrompt> <prompt>` |
| Cursor CLI | `agent` | `exec.LookPath("agent")` | `agent "<prompt>"` |

### 4. Interface Design

```go
// AgentRunner abstracts an AI agent CLI.
type AgentRunner interface {
    Name() string                    // Display name (e.g., "Claude Code")
    Binary() string                  // Binary name for PATH lookup
    Available() bool                 // Is the binary in PATH?
    BuildCommand(opts SpawnOpts) string // Build the full CLI command
}

type SpawnOpts struct {
    WorkDir      string
    TaskTitle    string
    TaskID       string
    TaskDesc     string
    TaskStage    string
}
```

### 5. Registry Pattern

```go
// Registry holds all known agent runners.
var runners = []AgentRunner{
    &ClaudeRunner{},
    &CursorRunner{},
}

func AvailableRunners() []AgentRunner { ... } // filters by Available()
func GetRunner(name string) AgentRunner { ... }
```

### 6. TUI Popup Behavior
- If only one agent is available: skip popup, use that agent directly
- If no agents detected: show error message
- If multiple available: show popup list with agent names
- Popup appears as an overlay on the current view

### 7. What Stays the Same
- **tmux layer** — Already agent-agnostic, no changes needed
- **DB schema** — `AgentName` field already stores any string
- **Agent monitoring** — Window-name-based reconciliation is agent-agnostic
- **Kill/view** — tmux window operations don't depend on agent type

## Scope

### In Scope
- `AgentRunner` interface and implementations for Claude + Cursor
- Agent registry with PATH-based detection
- Refactor `internal/agent/spawn.go` to use runners
- TUI agent picker popup
- Update package docs and comments

### Out of Scope
- Config file reading (not needed for per-task selection)
- Antigravity support (future work)
- Custom/user-defined agents via config
- Agent-specific monitoring or output parsing

## Resolved Questions

1. **Cursor CLI working directory** — `tmux.NewWindow` already accepts a `dir` parameter and passes `-c dir` to tmux, which sets the CWD for the spawned command. No changes needed — Cursor's `agent` command will inherit the correct working directory.

2. **Agent binary name collision** — Cursor's CLI is called `agent`, a generic name. **Decision:** Verify with `--version` — run `agent --version` and check the output contains "cursor" before registering it as available. This adds a small overhead at detection time but prevents false positives from other tools named `agent`.

## Open Questions

None — all questions resolved.
