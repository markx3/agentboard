---
title: "feat: Finish Intelligence Layer — TUI Suggestion Overlay + CLI Enrichment"
type: feat
date: 2026-02-25
brainstorm: docs/brainstorms/2026-02-24-finish-intelligence-layer-brainstorm.md
---

# feat: Finish Intelligence Layer — TUI Suggestion Overlay + CLI Enrichment

## Overview

Complete the AI-enabled intelligence layer by wiring the existing backend (suggestions, proposals, enrichment) into the TUI and shipping 4 deferred P2/P3 fixes. The backend (DB, service, CLI) is 100% done. This PR closes the gap: humans can now review and act on agent proposals directly from the board, and tasks created from the CLI are automatically enriched when a TUI is running.

**End-to-end loop this unlocks:**
```
Agent proposes task via CLI
  → human sees "2 pending" in status bar
  → human presses `s` to open suggestion overlay
  → reviews proposal, presses Enter to accept
  → task created + auto-enriched (enrichment_status=pending)
  → TUI enrichment ticker picks it up on next cycle
```

## Problem Statement

PR #17 and #18 closed 12 of the original 16 todos. Four remain:

| # | Todo | Priority | Location |
|---|------|----------|----------|
| 1 | Enrichment seen-set leaks deleted task IDs | P2 | `tui/app.go:enrichmentSeen` |
| 2 | `saveNeeded` bool violates Bubble Tea conventions | P3 | `tui/task_detail.go:37` |
| 3 | No `applyMigration` helper — copy-paste boilerplate | P3 | `internal/db/sqlite.go` |
| 4 | `task list --search` missing from CLI | P3 | `internal/cli/task_cmd.go` |

In addition, the suggestion flow exists entirely in the backend but has no TUI surface. Agents can propose tasks via `agentboard task propose` — but humans have no way to see or act on these proposals from the TUI.

## Proposed Solution

Two parallel workstreams, each with standalone value:

**Workstream A — Fixes (P2/P3 todos)**
Fix the four deferred issues. None block Workstream B, but delivering them together keeps the branch clean.

**Workstream B — TUI Suggestion Integration**
1. Poll `ListPendingSuggestions` periodically; show count in status bar
2. Fire a notification when new suggestions arrive
3. `s` key opens a lightweight suggestion overlay
4. Overlay: accept (`Enter`) / dismiss (`d`) / close (`Esc`)
5. Acceptance calls `AcceptSuggestion` which creates the task + sets `enrichment_status=pending`; TUI enrichment ticker handles the rest

## Technical Approach

### Architecture

```
CLI: agentboard task propose ...
  ↓ db.CreateSuggestion (type=proposal, status=pending)

TUI ticker (new) → board.ListPendingSuggestions()
  ↓ suggestionsLoadedMsg{items: []db.Suggestion}

app.go:
  - a.pendingSuggestions = items
  - a.pendingCount = len(items)
  - if pendingCount > a.lastPendingCount → a.notify("N new proposals")
  - status bar shows "N pending" badge

`s` key → overlaysuggestions
  ↓ tui/suggestion_overlay.go
  - list view of pending suggestions
  - `Enter` → board.AcceptSuggestion(id)
              → suggestionAcceptedMsg → reload tasks + reload suggestions
  - `d`     → board.DismissSuggestion(id)
              → suggestionDismissedMsg → reload suggestions
  - `Esc`   → close overlay
```

### File Map

| File | Action | What changes |
|------|--------|--------------|
| `internal/tui/suggestion_overlay.go` | **CREATE** | New overlay component |
| `internal/tui/messages.go` | **MODIFY** | Add suggestion message types |
| `internal/tui/app.go` | **MODIFY** | Poll, status bar count, `s` key, overlay routing |
| `internal/tui/task_detail.go` | **MODIFY** | Replace `saveNeeded` bool with `taskSaveRequestedMsg` |
| `internal/db/sqlite.go` | **MODIFY** | Extract `applyMigration(version, sql)` helper |
| `internal/cli/task_cmd.go` | **MODIFY** | Add `--search` flag to `task list` |

No new packages. No DB schema changes. No WebSocket protocol changes.

---

## Implementation Phases

### Phase 1: P2/P3 Fixes

#### 1a. Enrichment Seen-Set Cleanup (`tui/app.go`)

**Problem**: `enrichmentSeen` is a `map[string]db.EnrichmentStatus` that grows forever. When tasks are deleted, their IDs stay in the map. On very long sessions this is a memory leak and could cause subtle reconciliation bugs.

**Fix**: During `loadTasks` (the periodic task-reload path), diff the current task list against `enrichmentSeen` and prune any IDs not present in the loaded tasks.

```go
// internal/tui/app.go — in the loadTasks Cmd or in the tasksLoadedMsg handler
func pruneEnrichmentSeen(seen map[string]db.EnrichmentStatus, tasks []db.Task) {
    live := make(map[string]bool, len(tasks))
    for _, t := range tasks {
        live[t.ID] = true
    }
    for id := range seen {
        if !live[id] {
            delete(seen, id)
        }
    }
}
```

Call `pruneEnrichmentSeen(a.enrichmentSeen, tasks)` inside the `tasksLoadedMsg` handler in `app.go`, before the existing enrichment-ticker logic runs.

#### 1b. Replace `saveNeeded` Bool with `taskSaveRequestedMsg` (`tui/task_detail.go`)

**Problem**: `taskDetail.saveNeeded` (line 37) is a boolean flag that `app.go` checks on every `Update` cycle (`if a.detail.saveNeeded { a.detail.saveNeeded = false; a.saveTask() }`). This is a polling pattern — it works but violates Bubble Tea's message-passing conventions.

**Fix**: Remove the bool. When the detail overlay wants to save, it returns a `taskSaveRequestedMsg` as a `tea.Cmd`.

```go
// internal/tui/messages.go — add:
type taskSaveRequestedMsg struct{ task db.Task }

// internal/tui/task_detail.go — applyEdits() returns a Cmd:
func (d *taskDetail) applyEdits() (taskDetail, tea.Cmd) {
    // ... existing field update logic ...
    return *d, func() tea.Msg {
        return taskSaveRequestedMsg{task: d.task}
    }
}

// internal/tui/app.go — handler:
case taskSaveRequestedMsg:
    return a, a.saveTask(msg.task)
```

Remove `saveNeeded` field from `taskDetail` struct, remove the check from `app.go`'s main update loop.

#### 1c. Extract `applyMigration` Helper (`internal/db/sqlite.go`)

**Problem**: Each migration in `sqlite.go` (v1→v2, v2→v3, etc.) repeats the same pattern: begin tx, exec SQL, update schema_version, commit. This is ~10 lines of boilerplate repeated per migration.

**Fix**: Extract a private helper:

```go
// internal/db/sqlite.go
func applyMigration(ctx context.Context, tx *sql.Tx, version int, sql string) error {
    if _, err := tx.ExecContext(ctx, sql); err != nil {
        return fmt.Errorf("migration to v%d: %w", version, err)
    }
    _, err := tx.ExecContext(ctx,
        `UPDATE schema_version SET version = ?`, version)
    return err
}
```

Refactor existing migration blocks to call this helper. No behavior change — pure refactor.

#### 1d. Add `--search` to `task list` CLI (`internal/cli/task_cmd.go`)

**Problem**: The TUI has `/` fuzzy search, but `agentboard task list` has no search flag. Agents querying the board from scripts have to do client-side filtering.

**Implementation**:
```go
// internal/cli/task_cmd.go — add to listFlags:
var searchQuery string
taskListCmd.Flags().StringVar(&searchQuery, "search", "", "filter by title/description substring")

// In runTaskList():
tasks, err := svc.ListTasks(ctx)
if searchQuery != "" {
    tasks = filterTasksBySearch(tasks, searchQuery)
}

// private helper:
func filterTasksBySearch(tasks []db.Task, q string) []db.Task {
    q = strings.ToLower(q)
    var out []db.Task
    for _, t := range tasks {
        if strings.Contains(strings.ToLower(t.Title), q) ||
           strings.Contains(strings.ToLower(t.Description), q) {
            out = append(out, t)
        }
    }
    return out
}
```

This is client-side filtering (no DB change needed). Works with `--json` flag transparently.

---

### Phase 2: TUI Suggestion Polling + Status Bar Count

#### New Message Types (`internal/tui/messages.go`)

```go
type suggestionsLoadedMsg struct {
    items []db.Suggestion
}
type suggestionAcceptedMsg struct {
    taskID       string
    suggestionID string
}
type suggestionDismissedMsg struct {
    suggestionID string
}
```

#### Polling in `app.go`

Add a new ticker alongside the existing `enrichmentTicker`:

```go
// App struct — add:
pendingSuggestions []db.Suggestion
lastPendingCount   int

// New Cmd — analogous to checkForEnrichableNewTasks:
func (a *App) loadSuggestions() tea.Cmd {
    return func() tea.Msg {
        items, err := a.svc.ListPendingSuggestions(context.Background())
        if err != nil {
            return nil // non-fatal
        }
        return suggestionsLoadedMsg{items: items}
    }
}
```

Wire into the existing tick loop. When `suggestionsLoadedMsg` arrives:
- Update `a.pendingSuggestions`
- If `len(items) > a.lastPendingCount` → fire `a.notify(fmt.Sprintf("%d new proposal(s) pending — press s to review", delta))`
- Update `a.lastPendingCount`

#### Status Bar Count

The status bar is assembled in `internal/tui/board.go`'s `statusBar()` method (or `summaryBar()`). Add a pending count badge:

```go
// In board.go summaryBar() or passed via App.View():
if pendingCount > 0 {
    badge = suggestionBadgeStyle.Render(fmt.Sprintf("s: %d pending", pendingCount))
    // append badge to status bar line
}
```

Style: yellow/amber to stand out without being alarming. Use existing `lipgloss` style pattern from `styles.go`.

---

### Phase 3: Suggestion Overlay

#### New File: `internal/tui/suggestion_overlay.go`

Pattern: mirrors `task_detail.go` structure — a struct with `Init()`, `Update()`, `View()` methods.

```go
// internal/tui/suggestion_overlay.go

type suggestionOverlay struct {
    suggestions []db.Suggestion
    cursor      int
    width       int
    height      int
}

func newSuggestionOverlay(suggestions []db.Suggestion, w, h int) suggestionOverlay {
    return suggestionOverlay{suggestions: suggestions, width: w, height: h}
}

func (o suggestionOverlay) Update(msg tea.KeyMsg) (suggestionOverlay, tea.Cmd) {
    switch {
    case key.Matches(msg, keys.Up):
        if o.cursor > 0 { o.cursor-- }
    case key.Matches(msg, keys.Down):
        if o.cursor < len(o.suggestions)-1 { o.cursor++ }
    case key.Matches(msg, keys.Enter):
        if len(o.suggestions) > 0 {
            s := o.suggestions[o.cursor]
            return o, func() tea.Msg {
                return suggestionAcceptedMsg{suggestionID: s.ID}
            }
        }
    case key.Matches(msg, keys.D): // dismiss
        if len(o.suggestions) > 0 {
            s := o.suggestions[o.cursor]
            return o, func() tea.Msg {
                return suggestionDismissedMsg{suggestionID: s.ID}
            }
        }
    }
    return o, nil
}

func (o suggestionOverlay) View() string {
    // Render a bordered list of suggestions
    // Each item: [type badge] title/body preview
    // Selected item highlighted, shows full body below list
    // Footer: "Enter=accept  d=dismiss  Esc=close"
}
```

#### Overlay Routing in `app.go`

Add a new overlay constant and integrate into the routing:

```go
// Add overlay constant (alongside existing overlayDetail etc.):
const overlaySuggestions = "suggestions"

// Key binding — in updateBoard():
case key.Matches(msg, keys.S):
    if len(a.pendingSuggestions) > 0 {
        a.suggestionOverlay = newSuggestionOverlay(a.pendingSuggestions, a.width, a.height)
        a.overlay = overlaySuggestions
    }

// Overlay routing:
case overlaySuggestions:
    return a.updateSuggestionOverlay(msg)
```

#### Accept / Dismiss Handlers

```go
func (a *App) updateSuggestionOverlay(msg tea.KeyMsg) (*App, tea.Cmd) {
    // Check for Esc first:
    if key.Matches(msg, keys.Esc) {
        a.overlay = ""
        return a, nil
    }
    var cmd tea.Cmd
    a.suggestionOverlay, cmd = a.suggestionOverlay.Update(msg)
    return a, cmd
}

// Handle suggestionAcceptedMsg:
case suggestionAcceptedMsg:
    // 1. Call AcceptSuggestion
    // 2. Close overlay
    // 3. Reload tasks (new task was created)
    // 4. Reload suggestions
    return a, tea.Batch(
        a.acceptSuggestion(msg.suggestionID),
        a.loadSuggestions(),
    )

// acceptSuggestion Cmd:
func (a *App) acceptSuggestion(id string) tea.Cmd {
    return func() tea.Msg {
        task, err := a.svc.AcceptSuggestion(context.Background(), id)
        if err != nil {
            return errMsg{err}
        }
        return taskCreatedMsg{task: task}
    }
}
```

After acceptance, the new task has `enrichment_status=pending`. The existing `checkForEnrichableNewTasks` ticker picks it up on its next cycle — no additional wiring needed.

---

## Alternative Approaches Considered

| Approach | Decision |
|---|---|
| **Suggestion bar component** (collapsible panel below board) | Deferred — overlay is simpler, reuses existing patterns, validates the flow before building a dedicated component |
| **Push-based suggestion delivery** via WebSocket | Deferred — polling at 5s intervals is sufficient for v1 and avoids new protocol messages |
| **Trust levels** (auto-accept from specific agents) | Deferred to backlog — human approval required for v1 |
| **Transactional AcceptSuggestion** (explicit SQL tx) | Deferred — existing implementation is acceptable; failure mode is an orphaned task that can be manually deleted |
| **Suggestion preview card** (show proposed task fields before accepting) | Deferred — the overlay body section shows the proposal text; full preview card is polish |

---

## Acceptance Criteria

### Functional Requirements

- [x] `task list --search <query>` filters results by title/description substring (case-insensitive), works with `--json`
- [x] When a task is deleted, its ID is no longer retained in `enrichmentSeen` after the next task-load cycle
- [x] `saveNeeded` bool is removed from `taskDetail`; editing and saving a task from the detail overlay works correctly via message-passing
- [x] All existing migrations pass through `applyMigration` helper (no behavior change)
- [x] Status bar shows "s: N pending" badge when N > 0 pending suggestions exist
- [x] When new suggestions arrive (count increases), a notification fires: "N new proposal(s) pending — press s to review"
- [x] Pressing `s` on the board (when not in any overlay) opens the suggestion overlay
- [x] Suggestion overlay shows pending suggestions with title and body preview
- [x] `Enter` in overlay accepts the selected suggestion → task is created with `enrichment_status=pending` → TUI enrichment loop auto-enriches it
- [x] `d` in overlay dismisses the selected suggestion
- [x] `Esc` closes overlay without taking action
- [x] Overlay handles the empty state (no pending suggestions) gracefully — `s` key is a no-op when count is 0
- [x] After accept/dismiss, overlay refreshes the suggestion list

### Non-Functional Requirements

- [x] Suggestion polling interval: 5 seconds (same as other background tickers)
- [x] Overlay does not block TUI rendering or introduce noticeable lag
- [x] No new DB schema migrations required (uses existing `suggestions` table from v7)
- [x] All existing tests pass; new functionality has tests for `filterTasksBySearch` and `pruneEnrichmentSeen`

### Quality Gates

- [x] `go vet ./...` passes
- [x] `go test -race ./...` passes
- [x] No new linter warnings

---

## Dependencies & Prerequisites

| Dependency | Status |
|---|---|
| `db.CreateSuggestion`, `ListPendingSuggestions`, `GetSuggestion` | ✅ Done (`internal/db/suggestions.go`) |
| `board.AcceptSuggestion`, `board.DismissSuggestion`, `board.ListPendingSuggestions` | ✅ Done (`internal/board/local.go`) |
| `agentboard task propose` CLI command | ✅ Done (`internal/cli/task_cmd.go`) |
| Schema v7 with `suggestions` table and `enrichment_status` on tasks | ✅ Done (`internal/db/schema.go`) |
| TUI enrichment ticker (`checkForEnrichableNewTasks`) | ✅ Done (`internal/tui/app.go:1094`) |
| Notification overlay pattern | ✅ Done (`internal/tui/notification.go`) |

No external dependencies. No config changes.

---

## Risk Analysis

| Risk | Severity | Mitigation |
|------|----------|------------|
| Suggestion overlay key `s` conflicts with existing bindings | Low | `s` is not currently bound in board mode; verify in `keys.go` before implementing |
| Polling at 5s could miss rapid suggestion arrival | Low | Acceptable — this is a human-review flow, not real-time. Notification fires on next poll. |
| `AcceptSuggestion` creates task but enrichment fails to spawn | Low | Enrichment is non-blocking; task still exists and can be manually enriched later |
| `saveNeeded` refactor breaks task editing | Medium | Write a targeted test: edit a task in detail overlay, verify save completes. Run before merging. |
| Pruning `enrichmentSeen` mid-session interrupts active enrichment | Low | Only delete IDs whose tasks are gone; active enrichments have live task IDs |

---

## References

### Internal

- Brainstorm: `docs/brainstorms/2026-02-24-finish-intelligence-layer-brainstorm.md`
- `internal/board/local.go:143-168` — `AcceptSuggestion`, `DismissSuggestion`
- `internal/board/service.go:34-39` — Service interface (all 6 suggestion methods)
- `internal/db/suggestions.go` — All DB CRUD methods
- `internal/tui/app.go:1094-1150` — `checkForEnrichableNewTasks` (enrichment ticker pattern)
- `internal/tui/app.go:727-738` — Status bar assembly
- `internal/tui/task_detail.go:37` — `saveNeeded` bool
- `internal/tui/notification.go` — Notification pattern (`scheduleNotificationClear`)
- `internal/tui/task_detail.go` — Overlay component pattern (Init/Update/View)
- `internal/db/sqlite.go:126-207` — Migration chain (v1→v7)
- `internal/db/schema.go:3` — `schemaVersion = 7`
- `internal/cli/task_cmd.go:146,185,311-323` — `--no-enrich` flag

### Related PRs

- PR #17: `feat/worktree-massive-agent-drive-development-idea` — enrichment engine, suggestions DB/service/CLI
- PR #18: `feat/agent-driven-polish` — `status --json` enrichment fields, NULL task_id fix

---

## Implementation Order

1. **Phase 1 first** — Fixes are small, verifiable, safe to merge independently. Get them in so they don't conflict with overlay work.
2. **Phase 2 next** — Polling + status bar badge. Validates the service integration before the overlay.
3. **Phase 3 last** — Suggestion overlay. Depends on Phase 2's message types and polling Cmd.

Each phase can be reviewed independently. Recommend single PR with three commits if reviewer prefers, or three PRs if fine-grained review is desired.
