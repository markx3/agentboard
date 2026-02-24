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

type agentInfo struct {
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	Agent     string `json:"agent"`
	Status    string `json:"agent_status"`
	Column    string `json:"column"`
}

type enrichmentInfo struct {
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	Status    string `json:"enrichment_status"`
	Agent     string `json:"enrichment_agent,omitempty"`
}

type boardSummary struct {
	Columns            map[string]int   `json:"columns"`
	Total              int              `json:"total"`
	Agents             []agentInfo      `json:"agents,omitempty"`
	Enrichments        []enrichmentInfo `json:"enrichments,omitempty"`
	PendingSuggestions int              `json:"pending_suggestions"`
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
	var agents []agentInfo
	var enrichments []enrichmentInfo

	for _, t := range tasks {
		counts[string(t.Status)]++

		if t.AgentStatus == db.AgentActive {
			agents = append(agents, agentInfo{
				TaskID:    t.ID[:8],
				TaskTitle: t.Title,
				Agent:     t.AgentName,
				Status:    string(t.AgentStatus),
				Column:    string(t.Status),
			})
		}

		if t.EnrichmentStatus != "" && t.EnrichmentStatus != db.EnrichmentNone {
			enrichments = append(enrichments, enrichmentInfo{
				TaskID:    t.ID[:8],
				TaskTitle: t.Title,
				Status:    string(t.EnrichmentStatus),
				Agent:     t.EnrichmentAgentName,
			})
		}
	}

	pendingSuggestions := 0
	suggestions, sugErr := svc.ListPendingSuggestions(ctx)
	if sugErr == nil {
		pendingSuggestions = len(suggestions)
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
		Total:              len(tasks),
		Agents:             agents,
		Enrichments:        enrichments,
		PendingSuggestions: pendingSuggestions,
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

	if len(agents) > 0 {
		fmt.Printf("\nActive Agents:\n")
		for _, a := range agents {
			fmt.Printf("  %s: %s (%s) in %s\n", a.TaskID, a.Agent, a.Status, a.Column)
		}
	}

	if pendingSuggestions > 0 {
		fmt.Printf("\nPending Suggestions: %d\n", pendingSuggestions)
	}

	return nil
}
