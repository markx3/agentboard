package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show board summary",
	RunE:  runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(statusCmd)
}

type boardSummary struct {
	Columns map[string]int `json:"columns"`
	Total   int            `json:"total"`
}

func runStatus(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}

	counts := make(map[string]int)
	for _, t := range tasks {
		counts[string(t.Status)]++
	}

	summary := boardSummary{
		Columns: map[string]int{
			string(db.StatusBacklog):    counts[string(db.StatusBacklog)],
			string(db.StatusBrainstorm): counts[string(db.StatusBrainstorm)],
			string(db.StatusPlanning):   counts[string(db.StatusPlanning)],
			string(db.StatusInProgress): counts[string(db.StatusInProgress)],
			string(db.StatusReview):     counts[string(db.StatusReview)],
			string(db.StatusDone):       counts[string(db.StatusDone)],
		},
		Total: len(tasks),
	}

	if statusJSON {
		return json.NewEncoder(os.Stdout).Encode(summary)
	}

	fmt.Printf("Agentboard Status\n")
	fmt.Printf("─────────────────\n")
	fmt.Printf("Backlog:     %d\n", summary.Columns[string(db.StatusBacklog)])
	fmt.Printf("Brainstorm:  %d\n", summary.Columns[string(db.StatusBrainstorm)])
	fmt.Printf("Planning:    %d\n", summary.Columns[string(db.StatusPlanning)])
	fmt.Printf("In Progress: %d\n", summary.Columns[string(db.StatusInProgress)])
	fmt.Printf("Review:      %d\n", summary.Columns[string(db.StatusReview)])
	fmt.Printf("Done:        %d\n", summary.Columns[string(db.StatusDone)])
	fmt.Printf("─────────────────\n")
	fmt.Printf("Total:       %d\n", summary.Total)
	return nil
}
