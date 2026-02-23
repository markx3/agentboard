package cli

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	boardpkg "github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
	"github.com/marcosfelipeeipper/agentboard/internal/tui"
)

var connectAddr string

var rootCmd = &cobra.Command{
	Use:   "agentboard",
	Short: "Collaborative agentic task management TUI",
	Long:  "A terminal-based collaborative Kanban board for managing agentic coding tasks across a team.",
	RunE:  runBoard,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&connectAddr, "connect", "", "connect to a specific server (host:port)")
}

func Execute() error {
	return rootCmd.Execute()
}

func runBoard(cmd *cobra.Command, args []string) error {
	dbPath := filepath.Join(".agentboard", "board.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	svc := boardpkg.NewLocalService(database)
	app := tui.NewApp(svc)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}
