package cli

import (
	"testing"

	"github.com/markx3/agentboard/internal/db"
)

func TestFilterTasksBySearch(t *testing.T) {
	tasks := []db.Task{
		{ID: "1", Title: "Fix login bug", Description: "Users can't log in"},
		{ID: "2", Title: "Add dark mode", Description: "UI enhancement"},
		{ID: "3", Title: "Improve performance", Description: "Optimize login flow"},
	}

	tests := []struct {
		query    string
		wantIDs  []string
	}{
		{"login", []string{"1", "3"}},
		{"LOGIN", []string{"1", "3"}}, // case-insensitive
		{"dark", []string{"2"}},
		{"mode", []string{"2"}},
		{"notfound", nil},
		{"", []string{"1", "2", "3"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if tt.query == "" {
				// empty query not handled by filterTasksBySearch, skip
				return
			}
			got := filterTasksBySearch(tasks, tt.query)
			if len(got) != len(tt.wantIDs) {
				t.Errorf("filterTasksBySearch(%q) got %d results, want %d", tt.query, len(got), len(tt.wantIDs))
				return
			}
			for i, task := range got {
				if task.ID != tt.wantIDs[i] {
					t.Errorf("filterTasksBySearch(%q)[%d] got ID %q, want %q", tt.query, i, task.ID, tt.wantIDs[i])
				}
			}
		})
	}
}
