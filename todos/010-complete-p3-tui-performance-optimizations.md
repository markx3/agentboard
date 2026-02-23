---
status: pending
priority: p3
issue_id: "010"
tags: [code-review, performance, tui]
dependencies: []
---

# TUI Performance Optimizations (Style Allocation, Slice Pre-alloc)

## Problem Statement

Minor performance improvements: lipgloss styles allocated on every render frame (~16 heap allocs/frame), task slices grow via append without pre-allocation, and Sequencer uses mutex where atomic would suffice.

## Findings

- **Performance Review** OPT-6: column.View() allocates styles per render
- **Performance Review** OPT-7: ListTasks slice growth pattern
- **Performance Review** OPT-5: Sequencer mutex → atomic.Int64

### Locations:
- `internal/tui/column.go:70-87` - lipgloss.NewStyle() in View()
- `internal/tui/task_detail.go:34` - same pattern
- `internal/tui/task_form.go:78,96` - same pattern
- `internal/db/tasks.go:82` - `var tasks []Task` without capacity
- `internal/server/sequencer.go` - mutex for single int64

## Proposed Solutions

### Solution A: Hoist styles + pre-alloc + atomic (Recommended)
1. Move lipgloss styles to package-level vars in styles.go
2. Pre-allocate task slices: `make([]Task, 0, 64)`
3. Replace Sequencer mutex with `atomic.Int64`
- **Effort**: Small
- **Risk**: Low

## Acceptance Criteria

- [ ] No lipgloss.NewStyle() calls inside View() methods
- [ ] Task slices pre-allocated
- [ ] Sequencer uses atomic.Int64
- [ ] All tests pass

## Work Log

- 2026-02-23: Created from code review synthesis
