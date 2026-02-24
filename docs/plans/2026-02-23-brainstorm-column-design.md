# Brainstorm Column Design

## Problem

The board currently uses the Backlog column for brainstorming, with agents receiving brainstorm prompts when spawned on Backlog tasks. This conflates "unplanned ideas" with "active brainstorming." We want a dedicated Brainstorm column between Backlog and Planning to separate these concerns.

## Decisions

| Decision | Answer |
|----------|--------|
| Backlog agent spawning | Keep spawning, but no brainstorm prompt |
| Backlog agent prompt | "This task is in backlog. Move it to brainstorm to begin work." |
| Brainstorm column prompt | Same as current Backlog: `Run /workflows:brainstorm` |
| Brainstorm transition | Move to Planning when complete |
| Claim behavior | Claim → Brainstorm (instead of Planning) |
| Approach | Add `brainstorm` status to existing column system |

## New Column Order

```
Backlog → Brainstorm → Planning → In Progress → Review → Done
```

## Data Model & Schema

**New status constant** in `internal/db/models.go`:

```go
StatusBrainstorm TaskStatus = "brainstorm"
```

**Schema migration** — add `'brainstorm'` to CHECK constraint in `internal/db/schema.go`:

```sql
CHECK(status IN ('backlog','brainstorm','planning','in_progress','review','done'))
```

Existing SQLite databases need a migration to update the CHECK constraint.

## Agent Prompts

### Backlog (updated)

- Stage: `STAGE: Backlog — Unplanned`
- Transition: `Move to brainstorm to begin work: agentboard task move <id> brainstorm`
- Initial prompt: `"This task is in backlog. Move it to brainstorm to begin work."`

### Brainstorm (new)

- Stage: `STAGE: Brainstorm — Exploring Ideas`
- Transition: `When brainstorming is complete, move to planning: agentboard task move <id> planning`
- Initial prompt: `"Run /workflows:brainstorm to explore ideas for this task."`

### All other columns

Unchanged.

## Claim Behavior

- `ClaimTask` target changes from `StatusPlanning` to `StatusBrainstorm`
- `UnclaimTask` stays as-is (reverts to `StatusBacklog`)

## Files to Change

| File | Change |
|------|--------|
| `internal/db/models.go` | Add `StatusBrainstorm` constant |
| `internal/db/schema.go` | Add `'brainstorm'` to CHECK constraint + migration |
| `internal/tui/board.go` | Insert `StatusBrainstorm` in `columnOrder` and `columnTitles` |
| `internal/agent/claude.go` | Add Brainstorm case to stage prompt switch, update Backlog case, add Brainstorm initial prompt |
| `internal/board/local.go` | Change `ClaimTask` target from `StatusPlanning` to `StatusBrainstorm` |

## No Changes Needed

- Movement logic (`nextStatus`/`prevStatus`) — works automatically from `columnOrder`
- Agent reconciliation — status-agnostic
- WebSocket sync — transmits task status as string
- TUI rendering — renders from `columnOrder` dynamically
- CLI `task move` — validates against DB CHECK constraint
