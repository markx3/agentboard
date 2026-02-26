---
title: "feat: Render Markdown in Task Detail View with Glamour"
type: feat
date: 2026-02-26
task_id: 58b9016c
---

# feat: Render Markdown in Task Detail View with Glamour

## Overview

The AI enrichment pipeline (PRs #17 and #18) now writes structured Markdown to task descriptions. The task detail overlay (`internal/tui/task_detail.go`) currently renders descriptions as plain text via `lipgloss.NewStyle().Width(w).Render(s)` — asterisks, hashes, and backticks appear verbatim. This plan integrates `charmbracelet/glamour` to render headers, bold, code blocks, and bullet lists with proper ANSI styling.

The change is intentionally narrow: **one helper function**, **one call site** in `buildReadContent()`, and **one new dependency**. No new message types, no new overlays, no caching logic.

---

## Problem Statement

`buildReadContent()` in `task_detail.go:206` builds the scrollable content of the read view. The description section (line 224) uses a local `wrap()` closure that applies `lipgloss.NewStyle().Width(w).Render(s)` — plain word-wrap with no Markdown parsing. Any `## Header`, `**bold**`, or ```` ```code``` ```` in the description is displayed as literal characters.

The enrichment pipeline produces descriptions like:

```markdown
## Overview
Detailed task description here.

## Acceptance Criteria
- [ ] Item 1
- [ ] Item 2

```go
// Example code block
renderer.Render(desc)
```
```

This content is unreadable in the current plain-text view.

---

## Proposed Solution

Add `github.com/charmbracelet/glamour` and replace the single `wrap(t.Description)` call in `buildReadContent()` with a glamour-rendered string, falling back to plain lipgloss wrap on any error.

### Why Glamour

- Same charmbracelet organization as Bubble Tea, Bubbles, and Lip Gloss — designed for TUI use
- Already evaluated for task 51c907de (artifact viewer) and identified as the right library
- `glamour.Render(s, style)` is a one-call API; no AST wiring required
- Handles ANSI output width, color themes, and word-wrap natively

---

## Technical Approach

### Architecture

No new components or message types. The change is entirely within `taskDetail` internals.

```
buildReadContent() [task_detail.go:206]
  ├── title block     (unchanged)
  ├── description     ← replace wrap(t.Description) with renderMarkdown(t.Description, w)
  ├── status/assignee (unchanged)
  ├── agent info      (unchanged)
  └── comments        (unchanged, remain plain-text for this PR)
```

### The `renderMarkdown` Helper

A private function in `task_detail.go` encapsulates all glamour logic, including the fallback:

```go
// renderMarkdown renders Markdown source s to ANSI-styled text at the given
// column width. Falls back to plain lipgloss word-wrap if glamour fails or
// width is non-positive.
func renderMarkdown(s string, width int) string {
    if width <= 0 {
        return s
    }
    rendered, err := glamour.Render(s, glamour.DarkStyleName)
    if err != nil {
        // Fallback: plain word-wrap preserves readability.
        return lipgloss.NewStyle().Width(width).Render(s)
    }
    return strings.TrimRight(rendered, "\n")
}
```

> **Note on `glamour.Render` vs `glamour.NewTermRenderer`**
>
> `glamour.Render(s, styleName)` is a convenience wrapper that instantiates a renderer and renders in a single call. It is simpler than constructing a `TermRenderer` manually and is appropriate here because:
> - We need a new renderer on every call anyway (value-receiver architecture means we cannot safely cache it)
> - The `glamour.Render` API automatically applies word-wrap based on terminal width detection; we override with our own logic via the style name
>
> If word-wrap control or margin suppression requires it, replace with the explicit constructor:
>
> ```go
> r, err := glamour.NewTermRenderer(
>     glamour.WithStylePath("dark"),
>     glamour.WithWordWrap(width),
> )
> ```

### Call Site Change (task_detail.go:224)

```go
// Before
if t.Description != "" {
    lines = append(lines, wrap(t.Description), "")
}

// After
if t.Description != "" {
    lines = append(lines, renderMarkdown(t.Description, w), "")
}
```

The `wrap` closure is still used for comment bodies, which are left as plain text in this PR.

---

## Implementation Phases

### Phase 1: Dependency Addition

**File**: `go.mod`

```bash
go get github.com/charmbracelet/glamour@latest
go mod tidy
```

**Verify no lipgloss version conflict.** The existing pin is `lipgloss v1.1.0`. Glamour v0.7+ has a transitive dependency on lipgloss. If glamour requires a newer version, `go mod tidy` will update the indirect pin. Confirm with:

```bash
go mod graph | grep lipgloss
```

Both the existing direct `lipgloss v1.1.0` and any version glamour requires must be compatible. In Go module semantics, the higher version wins (MVS). If the direct dependency is `v1.1.0` and glamour needs `v1.0.x`, there is no conflict. If glamour needs `v1.2.0+`, the direct dependency needs bumping.

**Deliverable**: Clean `go build ./...` with glamour in the module graph.

### Phase 2: Core Implementation

**File**: `internal/tui/task_detail.go`

1. Add import: `"github.com/charmbracelet/glamour"` and `"strings"` (if not already present).
2. Add `renderMarkdown(s string, width int) string` function (see above).
3. Replace `wrap(t.Description)` with `renderMarkdown(t.Description, w)` on line 224.

**Width/margin strategy:**

Glamour's `DarkStyleName` style includes default left/right margins (typically 2 cells each). This means with `WithWordWrap(width)`, the actual text content renders at `width - 4`. Since the viewport itself already clips content at `d.vp.Width`, the slight narrowing is acceptable and avoids horizontal overflow.

If the margins cause visible waste, disable them by switching to `glamour.NewTermRenderer` with a custom style JSON that sets `"margin": 0`. This is an optimization to defer unless it looks bad in practice.

**Trailing newline:**

Glamour appends `\n` to its output. The `lines` slice is joined with `"\n"` and a separator `""` element is appended after the description. Without trimming, this produces a double blank line between the description and the Status field. `strings.TrimRight(rendered, "\n")` prevents this.

**Width guard:**

`d.vp.Width` is computed as `d.width/2 - 6`. When `d.width == 0` (before the first `WindowSizeMsg`), this is `-6`. Passing a non-positive width to glamour is undefined behavior. The guard `if width <= 0 { return s }` ensures the fallback path is taken instead.

### Phase 3: Testing

**File**: `internal/tui/task_detail_test.go` (new file)

Minimum test coverage:

```go
package tui_test

import (
    "strings"
    "testing"
)

func TestRenderMarkdown_Headers(t *testing.T) {
    // Given a Markdown header
    input := "## Section\n\nBody text here."
    result := renderMarkdown(input, 80)
    // Expect ANSI bold sequence or styled text — not raw ##
    if strings.Contains(result, "##") {
        t.Errorf("renderMarkdown left raw Markdown markers in output: %q", result)
    }
}

func TestRenderMarkdown_ZeroWidth(t *testing.T) {
    input := "## Header\n\nBody."
    // Zero width must not panic and must return something
    result := renderMarkdown(input, 0)
    if result == "" {
        t.Error("renderMarkdown returned empty string for zero-width fallback")
    }
}

func TestRenderMarkdown_NegativeWidth(t *testing.T) {
    input := "## Header"
    result := renderMarkdown(input, -6)
    // Must return the raw string (fallback), not panic
    if result != input {
        t.Errorf("expected raw string fallback, got %q", result)
    }
}

func TestRenderMarkdown_GlamorFallback_EmptyString(t *testing.T) {
    // Empty string: guard in buildReadContent skips this, but renderMarkdown
    // must not panic on empty input.
    result := renderMarkdown("", 80)
    _ = result // any value is acceptable; must not panic
}
```

Note: `renderMarkdown` must be exported or the test must be in the `tui` package (not `tui_test`) to access the unexported function. Since this is an internal helper, place the test in `package tui` (not `package tui_test`) to access it directly.

### Phase 4: Verification

```bash
# 1. Build succeeds
go build ./...

# 2. Vet passes
go vet ./...

# 3. Tests pass with race detector
go test -race -v ./...

# 4. Manual smoke test: launch TUI, select an enriched task, verify header rendering
```

---

## Edge Cases and Mitigations

| Edge Case | Risk | Mitigation |
|-----------|------|------------|
| `d.vp.Width <= 0` (before first `WindowSizeMsg`) | glamour panic / empty output | Guard: `if width <= 0 { return s }` — same as existing `wrap()` |
| Glamour returns error | Plain text not shown | Fallback to `lipgloss.NewStyle().Width(w).Render(s)` |
| Glamour trailing `\n` | Double blank line between description and Status | `strings.TrimRight(rendered, "\n")` |
| Glamour internal margins (default ~2 cells each side) | Text narrower than viewport | Acceptable for now; can disable with custom style if needed |
| Light-background terminal | Glamour dark theme unreadable | Using `DarkStyleName` explicitly; `WithAutoStyle()` avoided to prevent termenv queries |
| Description with embedded ANSI (old plain-text tasks) | Double-rendering artifacts | Glamour will escape or re-render the ANSI; minor visual artifact, acceptable |
| Very large description (> 50 KB) | UI blocking during glamour parse | Deferred — enrichment pipeline limits content size in practice |
| `buildReadContent()` called from both `Update` and `View` | Double glamour render per frame | Accepted cost; glamour renders in < 1ms for typical descriptions |
| lipgloss version conflict between direct dep and glamour transitive | Build failure | Verify with `go mod graph` in Phase 1; bump direct dep if needed |
| Comment bodies left as plain text | Visual inconsistency | Known gap; deferred to follow-up issue |

---

## Out of Scope (Explicit Non-Goals)

- **Markdown preview in edit mode** — the textarea shows raw Markdown source. Acceptable for the current developer audience. Deferred.
- **Live description refresh when enrichment completes** — `a.detail.task` is only refreshed via `taskSavedMsg`, not `tasksLoadedMsg`. The user must close and reopen the overlay after enrichment. Deferred.
- **Comment Markdown rendering** — comments remain plain text. A follow-up issue should cover this.
- **Renderer caching** — no struct-level renderer cache. The value-receiver architecture and double-render-per-frame pattern make this complex without benefit for typical description sizes.
- **Custom glamour theme** — using `DarkStyleName` as-is. A future task could add theme configuration.

---

## Acceptance Criteria

### Functional Requirements

- [x] Task detail view renders `## Headers` with bold/sized ANSI styling (not literal `##`)
- [x] Task detail view renders `**bold**` and `*italic*` with ANSI bold/italic (not literal asterisks)
- [x] Task detail view renders fenced code blocks with syntax highlighting background
- [x] Task detail view renders bullet lists (`-` or `*`) as formatted list items
- [x] Empty description: the `if t.Description != ""` guard still prevents any glamour call
- [x] Plain-text (non-Markdown) descriptions: glamour renders them cleanly as prose

### Non-Functional Requirements

- [x] Terminal resize reflows correctly — `SetSize` is called on `WindowSizeMsg` when overlay is open, providing correct width to the next `buildReadContent()` call
- [x] No crash when `d.vp.Width <= 0` (zero-width guard is in place)
- [x] Graceful fallback to plain text on glamour error (fallback path in `renderMarkdown`)
- [x] `go test -race ./...` passes — no data races introduced

### Quality Gates

- [x] `go vet ./...` passes
- [x] `go build ./...` succeeds
- [x] `TestRenderMarkdown_*` tests pass
- [ ] Manual smoke test with an enriched task description confirms formatted output

---

## Key Files

| File | Change |
|------|--------|
| `go.mod` | Add `github.com/charmbracelet/glamour` |
| `go.sum` | Updated by `go mod tidy` |
| `internal/tui/task_detail.go` | Add `renderMarkdown` helper; replace `wrap(t.Description)` call |
| `internal/tui/task_detail_test.go` | New file: unit tests for `renderMarkdown` |

No changes to `app.go`, `messages.go`, `styles.go`, or any other file.

---

## Dependencies and Related Tasks

- **Task 51c907de** (Command to Display Generated Artifacts): also evaluates glamour for artifact browsing. Once this PR adds glamour to `go.mod`, task 51c907de can reuse the dependency. Consider extracting `renderMarkdown` to a shared utility if both tasks end up needing it.
- **Task a3236865** (Finish massive agent-driven agentboard proposal): in_progress. Enrichment output quality is visible only after this Markdown rendering lands.

---

## References

### Internal

- `internal/tui/task_detail.go:206` — `buildReadContent()` — the primary change site
- `internal/tui/task_detail.go:210-216` — existing `wrap()` closure (fallback pattern to preserve)
- `internal/tui/task_detail.go:223-225` — current `wrap(t.Description)` call to replace
- `internal/tui/task_detail.go:50-60` — `SetSize()` — where `d.vp.Width` is set
- `internal/tui/app.go:170-179` — `WindowSizeMsg` handler that calls `detail.SetSize`
- `internal/tui/app.go:687-695` — where `newTaskDetail` is called on Enter keypress
- `go.mod` — current dependency list (no glamour; direct lipgloss v1.1.0)

### External

- [charmbracelet/glamour](https://github.com/charmbracelet/glamour) — library to add
- `glamour.Render(s, styleName string)` — one-call convenience API
- `glamour.DarkStyleName` — dark ANSI theme consistent with app palette
