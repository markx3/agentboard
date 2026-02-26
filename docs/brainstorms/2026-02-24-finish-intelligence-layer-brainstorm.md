---
date: 2026-02-24
topic: finish-intelligence-layer
---

# Finish Intelligence Layer: End-to-End Proposal Flow + CLI Enrichment

## What We're Building

Complete the AI-enabled intelligence layer by wiring the existing backend (suggestions, proposals, enrichment) into the TUI and ensuring all task creation paths trigger enrichment. Fix the 4 remaining P2/P3 todos.

Two pillars:
1. **TUI suggestion/proposal integration** -- Wire suggestions into the TUI so humans can review, accept, and dismiss agent proposals directly from the board
2. **Universal enrichment** -- Ensure tasks created via CLI (not just TUI) get enriched when a TUI instance is running

## Why This Approach

The backend is 95% complete (PR #17 + #18 resolved 11/16 todos). The critical gap is that the TUI has zero suggestion/proposal integration -- agents can propose tasks via CLI, but humans can only interact with proposals via CLI too. The end-to-end loop (agent proposes -> human reviews in TUI -> accepts -> task created -> auto-enriched) doesn't work yet.

We chose "Focused Wiring + CLI Enrichment" over:
- **Polished Proposal UX** (dedicated preview overlay): Deferred -- validate the flow works first
- **Orchestrator seed** (Phase 5): Deferred to backlog -- needs more infrastructure

## Key Decisions

- **Lightweight suggestion overlay, not suggestion bar**: Use a simple list overlay (similar to existing task detail overlay) triggered by `s` key. Shows pending suggestions with accept/dismiss actions. No dedicated suggestion bar component for v1 -- defer if the overlay proves insufficient.

- **Notification-driven discovery**: TUI status bar shows "N pending suggestions" count. Notification fires when new suggestions arrive. Human presses `s` to review.

- **CLI-created tasks enriched via ticker**: The TUI's existing enrichment ticker (`checkForEnrichableNewTasks`) already scans all tasks with `enrichment_status=pending`. CLI-created tasks just need to be created with `enrichment_status=pending` by default. The TUI picks them up on next tick.

- **`--no-enrich` flag**: `task create --no-enrich` sets `enrichment_status=skipped`. Default behavior: enrich. This lets humans skip enrichment for trivial tasks.

- **Enrichment seen-set cleanup**: Prune deleted task IDs from `enrichmentSeen` map during `loadTasks` by diffing against current task list.

- **saveNeeded -> tea.Cmd**: Replace the `saveNeeded` bool flag in task_detail with a proper `taskSaveRequestedMsg` returned as a `tea.Cmd`, following Bubble Tea conventions.

- **Migration helper**: Extract `applyMigration(version, sql)` to reduce copy-paste boilerplate for future migrations.

## Scope

### Already Done (from PR #17 + #18)

- ✅ `--no-enrich` flag on `task create` CLI (`cli/task_cmd.go:146,185,311-323`)
- ✅ CLI-created tasks default to `enrichment_status=pending` (same file)

### In Scope (This PR)

**Fixes (4 remaining todos):**
1. Enrichment seen-set cleanup for deleted tasks (P2) — map exists, cleanup logic missing
2. Replace `saveNeeded` flag with `tea.Cmd` pattern (P3) — bool still in `tui/task_detail.go:37`
3. Extract migration helper to reduce boilerplate (P3) — no `applyMigration` helper yet
4. Add `--search` flag to `task list` CLI (P3) — TUI has search, CLI does not

**New Features:**
5. TUI suggestion overlay (`s` to open, `Enter` to accept, `d` to dismiss, `Esc` to close) — no `suggestion_overlay.go` yet
6. Pending suggestions count in TUI status bar — `status_cmd.go` has it, `app.go` does not
7. Suggestion arrival notifications — not yet wired
8. Proposal acceptance via TUI — `AcceptSuggestion` service exists, needs TUI wiring

### Out of Scope (Backlog)

- Phase 5: Autonomous orchestration (orchestrator agent, capabilities registry, auto-assignment)
- Polished proposal preview overlay with task card preview
- Full suggestion bar component (collapsible panel)
- Trust levels for auto-approval
- Optimistic locking for concurrent edits
- `enrichment_started_at` column for application-level timeout tracking

## Existing Building Blocks

| Building Block | Where | How It Helps |
|---|---|---|
| AcceptSuggestion service method | board/local.go:143-164 | Creates task from proposal + sets enrichment=pending |
| DismissSuggestion service method | board/local.go:166-168 | Marks suggestion as dismissed |
| ListSuggestions DB method | db/suggestions.go | Query pending/all suggestions |
| Notification overlay | tui/notification.go | Pattern for suggestion notifications |
| Task detail overlay | tui/task_detail.go | Pattern for suggestion list overlay |
| Enrichment ticker | tui/app.go:checkForEnrichableNewTasks | Already scans for enrichment_status=pending |
| Suggestion CLI commands | cli/task_cmd.go | accept/dismiss/list already work from CLI |

## Resolved Questions

- **Overlay vs suggestion bar**: Overlay for v1. Lightweight, reuses existing patterns, validates the flow before investing in a dedicated component.
- **CLI enrichment mechanism**: No special wiring needed -- CLI creates task with enrichment_status=pending, TUI ticker picks it up. Just need the default and the --no-enrich flag.
- **Proposal acceptance atomicity**: AcceptSuggestion in board/local.go already handles create task + update suggestion. Not fully transactional (no explicit tx) but acceptable for v1 since the failure mode is an orphaned task (recoverable).

## Next Steps

-> `/workflows:plan` for implementation details.
