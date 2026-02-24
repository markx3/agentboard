---
status: pending
priority: p2
issue_id: "019"
tags: [code-review, simplicity, plan-consistency]
dependencies: []
---

# Remove Deferred Features from Plan Detail Specs

## Problem Statement

The plan says "defer X to post-v1" in research notes but then includes full specs for X in the detailed sections. This creates contradictions where the implementation sequence omits things that the phase descriptions include.

## Findings

- **Source**: code-simplicity-reviewer
- **Estimated savings**: ~300-335 LOC (20-25% reduction)

Specific contradictions:
1. **Suggestion bar**: Deferred to v1.1 in impl sequence but fully specced in Phase 3.2
2. **`enrichment_started_at` column**: Deferred timeout mechanism but column in migration
3. **3 suggestion types**: Only `proposal` needed but all 3 specced with typed constants
4. **Enricher interface composition**: Premature abstraction for 2 runners
5. **Service interface decomposition**: One implementation, theoretical benefit only
6. **`ListDependents` query**: No consumer in v1
7. **`--no-enrich` flag**: Syntactic sugar for `task update --enrichment-status skipped`
8. **`task enrich` command**: Same as `task update --enrichment-status pending`

## Proposed Solutions

### Option A: Clean up plan (Recommended, Effort: Small, Risk: Low)
Remove or mark as "v1.1" the items that are explicitly deferred. Keep them as brief notes for future reference rather than full implementation specs.

## Acceptance Criteria

- [ ] Plan implementation sequence and phase descriptions are consistent
- [ ] Deferred items clearly marked as "v1.1" or "future"
- [ ] No orphaned specs for features not in the implementation sequence
