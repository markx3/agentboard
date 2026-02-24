---
status: pending
priority: p3
issue_id: "021"
tags: [code-review, enrichment, agent-native]
dependencies: []
---

# Write Concrete Enrichment Prompt Template

## Problem Statement

The plan describes the enrichment prompt in prose (8 steps at section 2.3) but does not provide the actual prompt text. The existing `buildClaudeSystemPrompt` at `internal/agent/claude.go:38-77` is concrete Go code. The enrichment prompt needs the same level of detail.

## Findings

- **Source**: agent-native-reviewer
- **Missing from spec**: SQLite retry with jitter instructions, failure handling, dependency heuristics, comment format, idempotency check (avoid duplicate enrichment)

## Proposed Solutions

### Option A: Write full prompt in plan (Effort: Medium, Risk: Low)
Include the actual `buildEnrichmentSystemPrompt()` and `buildEnrichmentPrompt()` function bodies in the plan, with:
- Retry logic for CLI commands
- Instructions for determining dependencies
- Comment body format
- Exit behavior on success/failure

## Acceptance Criteria

- [ ] Enrichment prompt template is a concrete Go string, not prose description
- [ ] Includes CLI retry instructions
- [ ] Includes dependency identification heuristics
- [ ] Includes comment format specification
