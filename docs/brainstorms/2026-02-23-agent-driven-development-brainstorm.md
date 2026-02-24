---
date: 2026-02-23
topic: agent-driven-development
---

# Agent-Driven Development Powerhouse

## What We're Building

Transform agentboard from a task tracker into an intelligence layer for agent-driven development. When a human creates a task, an enrichment agent immediately correlates with everything in flight -- running agents, board state, codebase context -- to flesh out the title, description, and dependency map. The TUI becomes agent-assisted, surfacing suggestions and context. Agents can proactively create tasks when they discover work that needs doing.

Three pillars:
1. **Context-aware task enrichment** -- Auto-enrich new tasks by scanning running agents, board state, and codebase
2. **Agent-assisted TUI** -- Suggestion bar, contextual hints, smart actions in the interface
3. **Automated task creation** -- Agents propose tasks that humans approve (or auto-approve with trust levels)

## Why This Approach

We considered three scopes:

- **Incremental (enrichment only)**: Quick win but doesn't deliver the full vision
- **Board Intelligence Layer (recommended starting point)**: Covers all three pillars with human-in-the-loop guardrails. Builds on existing patterns (reconciliation loop, overlay system, WebSocket protocol)
- **Autonomous Agent Swarm (north star)**: Full autonomy but needs too many missing building blocks for v1

**Decision**: Build the Board Intelligence Layer incrementally, with the Autonomous Swarm as the north star. Each phase delivers standalone value.

## Key Decisions

- **Non-blocking enrichment via existing agent runners**: Enrichment is a background agent session -- NOT a blocking API call or inline goroutine. When `CreateTask` is called, agentboard spawns a non-blocking agent (Claude Code, Cursor, or whatever runner the user has configured) in a tmux window, just like it does for task work today. The enrichment agent runs in the background, gathers context, and updates the task via `agentboard task update` CLI commands. The human never waits -- they create the task and keep working. The TUI picks up the enriched fields on its next poll/broadcast cycle.

- **Runner-agnostic**: The enrichment uses the same `AgentRunner` interface (`ClaudeRunner`, `CursorRunner`, etc.) that already exists. Whatever agent tool the user has configured is what runs the enrichment. No direct API calls, no hardcoded LLM providers. This keeps agentboard tool-agnostic.

- **Human-in-the-loop for agent-created tasks**: Agents propose tasks via a suggestion queue. Humans approve/reject/edit from the TUI. Trust levels can be added later (auto-approve from trusted agents).

- **Background intelligence goroutine**: A new `intelligence` package runs as a background loop in the TUI (similar to the existing agent reconciliation ticker). It watches for events (task created, task moved, agent died) and reacts by spawning enrichment agents or queuing suggestions.

- **Extend WebSocket protocol**: Add `task.update` message type for broadcasting field changes (enriched descriptions, agent status, dependencies). This is a prerequisite -- currently only create/move/delete/claim are supported.

- **Task dependencies as a first-class concept**: Add `blocked_by` and `blocks` fields to the task model. The enrichment agent populates these by analyzing what's in flight. Display dependency lines or badges in the TUI.

- **Agent context gathering via CLI**: The enrichment agent uses existing CLI commands (`agentboard task list --json`, `agentboard status`) plus reads the codebase (git status, branch names, recent commits) to build context. No new APIs needed.

- **Suggestion bar in TUI**: A new TUI component (below the board or as a collapsible panel) shows agent suggestions: enrichment previews, proposed tasks, recommended actions. Keyboard shortcut to accept/dismiss.

## Existing Building Blocks

These already exist and can be leveraged:

| Building Block | Where | How It Helps |
|---|---|---|
| Agent reconciliation loop | `tui/app.go:250-326` | Pattern for background intelligence ticker |
| Stage-based prompting | `agent/runner.go` | Enrichment agent gets column-aware context |
| CLI for agents | `cli/task_cmd.go`, `cli/agent_cmd.go` | Agents interact with the board programmatically |
| Overlay system | `tui/app.go` | Suggestion approval overlay, enrichment preview |
| Comments table | `db/comments.go` | Agent observations can be stored as comments |
| WebSocket hub | `server/hub.go` | Broadcast enrichments and suggestions to peers |
| Service interface | `board/service.go` | Clean abstraction to add enrichment hooks |

## Gaps to Fill

| Gap | Why It Matters | Priority |
|---|---|---|
| No `task.update` WebSocket message | Can't broadcast enrichments to peers | P0 |
| No task dependencies | Can't express "blocked by" relationships | P1 |
| Comments not in Service interface | Agents can't leave observations | P1 |
| No agent output capture | Can't see what agents discovered | P2 |
| Peersync messages not consumed | Remote enrichments invisible | P2 |
| No agent capabilities registry | Can't route tasks intelligently | P3 |

## Delivery Phases

### Phase 1: Foundation (task.update + dependencies)
- Add `task.update` WebSocket message type
- Add `blocked_by`/`blocks` fields to task model (schema migration v5)
- Wire comments into the Service interface and TUI detail view
- **Value**: Peers can see real-time updates, tasks can express dependencies

### Phase 2: Task Enrichment Engine
- New `internal/intelligence/` package with an `Enricher` interface
- On task creation, spawn a **non-blocking agent session** (using the configured `AgentRunner` -- Claude Code, Cursor, etc.) in a tmux window with an enrichment-specific prompt
- The enrichment agent runs in the background and:
  - Reads board state (`agentboard task list --json`)
  - Scans running agents (tmux windows + their task context)
  - Analyzes codebase (git status, recent commits, branch names)
  - Updates the task via `agentboard task update` CLI commands
- TUI picks up enriched fields on next poll cycle, shows a notification
- Enrichment agent tmux window auto-closes when done (short-lived)
- **Value**: Every new task automatically gets rich context, using the user's own agent tool, without blocking anything

### Phase 3: Agent-Assisted TUI
- Suggestion bar component in the TUI
- Agents can push suggestions via `agentboard suggest <task-id> <message>` CLI command
- Keyboard shortcuts to accept/dismiss/view suggestions
- Contextual hints (e.g., "This task might be blocked by task X which is in progress")
- **Value**: TUI becomes a smart assistant, not just a display

### Phase 4: Automated Task Creation
- Agents can call `agentboard task propose --title "..." --description "..." --reason "..."`
- Proposed tasks appear in the suggestion bar with accept/reject actions
- Trust levels: manual approval (default), auto-approve for specific agents
- Agents discover work during their runs and propose it back to the board
- **Value**: The board grows organically as agents discover work

### Phase 5: Autonomous Orchestration (North Star)
- Orchestrator agent watches the board and manages workflow
- Auto-assigns tasks to available agents based on capabilities
- Dependency-aware scheduling (don't start blocked tasks)
- Agent performance tracking and intelligent routing
- **Value**: Humans set direction, agents execute autonomously

## Resolved Questions

- **LLM for enrichment**: Non-blocking agent session using the user's configured runner (Claude Code, Cursor, etc.). Same `AgentRunner` interface, same tmux spawn pattern. No direct API calls -- agentboard stays tool-agnostic. The enrichment agent is short-lived and updates the task via CLI.
- **Enrichment latency tolerance**: Non-blocking by design. Human creates the task and keeps working. Enrichment runs in the background and the TUI picks up updates on the next poll cycle. Latency is irrelevant to UX since nothing blocks.
- **Trust model for auto-creation**: Human approval by default. Agent proposals appear as suggestions in the TUI. Configurable trust levels can be added later but are not needed for v1.
- **Dependency visualization**: Badges/icons in the board view first (minimal TUI change, immediate value). A toggleable dependency panel can be added in a later phase.

## Next Steps

-> `/workflows:plan` for implementation details, starting with Phase 1 (Foundation).
