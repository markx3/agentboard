package tui

import (
	"testing"

	"github.com/markx3/agentboard/internal/db"
)

func TestPruneEnrichmentSeen(t *testing.T) {
	seen := map[string]db.EnrichmentStatus{
		"aaa": db.EnrichmentPending,
		"bbb": db.EnrichmentDone,
		"ccc": db.EnrichmentError,
	}

	// Only task "aaa" is still alive
	live := []db.Task{
		{ID: "aaa"},
	}

	pruneEnrichmentSeen(seen, live)

	if _, ok := seen["aaa"]; !ok {
		t.Error("expected 'aaa' to remain in enrichmentSeen")
	}
	if _, ok := seen["bbb"]; ok {
		t.Error("expected 'bbb' to be pruned from enrichmentSeen")
	}
	if _, ok := seen["ccc"]; ok {
		t.Error("expected 'ccc' to be pruned from enrichmentSeen")
	}
}

func TestPruneEnrichmentSeenEmpty(t *testing.T) {
	seen := map[string]db.EnrichmentStatus{
		"xxx": db.EnrichmentPending,
	}

	pruneEnrichmentSeen(seen, []db.Task{})

	if len(seen) != 0 {
		t.Errorf("expected empty map after pruning, got %d entries", len(seen))
	}
}

func TestPruneEnrichmentSeenNoOp(t *testing.T) {
	seen := map[string]db.EnrichmentStatus{
		"id1": db.EnrichmentDone,
		"id2": db.EnrichmentDone,
	}

	tasks := []db.Task{{ID: "id1"}, {ID: "id2"}}
	pruneEnrichmentSeen(seen, tasks)

	if len(seen) != 2 {
		t.Errorf("expected 2 entries (no pruning needed), got %d", len(seen))
	}
}
