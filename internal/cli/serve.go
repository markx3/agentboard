package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	boardpkg "github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
	"github.com/marcosfelipeeipper/agentboard/internal/peersync"
	"github.com/marcosfelipeeipper/agentboard/internal/server"
)

var servePort int
var serveHost string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start as a dedicated server (persistent mode, no TUI)",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "port to listen on (0 for random)")
	serveCmd.Flags().StringVar(&serveHost, "bind", "127.0.0.1", "address to bind to")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dbPath := filepath.Join(".agentboard", "board.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	svc := boardpkg.NewLocalService(database)
	srv := server.New(svc, serveHost, servePort)

	go func() {
		<-ctx.Done()
		peersync.RemoveServerInfo()
	}()

	fmt.Fprintf(os.Stderr, "Starting agentboard server on %s:%d...\n", serveHost, servePort)

	if err := srv.Start(ctx); err != nil {
		return err
	}

	// Write server info for peer discovery
	if err := peersync.WriteServerInfo(srv.Addr()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write server info: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Server running at %s\n", srv.Addr())
	<-ctx.Done()
	return nil
}
