# Skip Permissions for Agents

**Date:** 2026-02-23
**Status:** Brainstorm Complete
**Task:** Allow skip-permissions for agents (4b7a7a0b)

## What We're Building

A simple toggle that lets users skip Claude Code permission prompts when spawning agents. Currently, agents run in tmux windows with default interactive permissions, causing permission prompts to block silently — users must manually attach to each tmux session to approve every file edit or shell command. This makes agents unusable without constant babysitting.

The solution: when a user presses `a` to spawn an agent, a confirmation prompt asks "Skip permissions?". If yes, the agent launches with `--dangerously-skip-permissions`, allowing it to run autonomously. The choice is stored per-task so auto-respawns use the same setting.

## Why This Approach

**Simple toggle over granular control:** Claude CLI supports multiple permission modes and tool whitelisting, but a single on/off toggle covers the primary use case (autonomous agents) with minimal complexity. Users who need granular control can configure Claude's `.claude/settings.json` directly.

**Spawn-time prompt over persistent config:** Each spawn presents a fresh choice, making the permission escalation explicit and intentional. No need to wire up config.toml parsing (which is currently dead code anyway).

**Task-level DB column over in-memory state:** Persists across agentboard restarts. Respawns (column moves, request-reset) inherit the original choice without re-prompting.

## Key Decisions

1. **Scope:** Simple boolean toggle — skip all permissions or use defaults
2. **UX:** Confirmation prompt when pressing `a` to spawn ("Skip permissions? [y/n]")
3. **Persistence:** Store `skip_permissions` boolean on the task in SQLite
4. **Respawn behavior:** Auto-respawns inherit the stored setting
5. **Implementation:** Add `--dangerously-skip-permissions` flag to the `claude` command in `agent.Spawn()`

## Changes Required

| File | Change |
|------|--------|
| `internal/db/schema.sql` | Add `skip_permissions` column to tasks table |
| `internal/db/models.go` | Add `SkipPermissions bool` field to Task model |
| `internal/db/queries.go` | Include field in CRUD operations |
| `internal/agent/spawn.go` | Accept and use `skipPermissions` parameter |
| `internal/tui/app.go` | Add confirmation prompt before spawning |

## Out of Scope

- Granular permission modes (`--permission-mode`)
- Tool whitelisting (`--allowedTools` / `--disallowedTools`)
- Config file-based defaults (config.toml parsing)
- CLI `agentboard agent spawn` command (spawning is TUI-only)
