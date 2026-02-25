package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/markx3/agentboard/internal/auth"
	boardpkg "github.com/markx3/agentboard/internal/board"
	"github.com/markx3/agentboard/internal/db"
	"github.com/markx3/agentboard/internal/peersync"
	"github.com/markx3/agentboard/internal/tui"
)

var connectAddr string

// Version is set at build time via ldflags.
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "agentboard",
	Short:   "Collaborative agentic task management TUI",
	Long:    "A terminal-based collaborative Kanban board for managing agentic coding tasks across a team.",
	Version: Version,
	RunE:    runBoard,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&connectAddr, "connect", "", "connect to a remote server (e.g. https://abc.ngrok-free.app or 127.0.0.1:8080)")
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

	var opts []tui.AppOption
	var connector *peersync.Connector

	if connectAddr != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		token, err := auth.GetToken(ctx)
		if err != nil {
			return fmt.Errorf("getting auth token: %w", err)
		}

		connector = peersync.NewConnector(connectAddr, token)
		if err := connector.Connect(ctx); err != nil {
			return fmt.Errorf("connecting to server: %w", err)
		}
		defer connector.Close()

		opts = append(opts, tui.WithConnectAddr(connectAddr))
	}

	app := tui.NewApp(svc, opts...)
	_ = connector // connector messages will be handled in a future iteration

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return nil
}
