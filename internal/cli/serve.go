package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	boardpkg "github.com/markx3/agentboard/internal/board"
	"github.com/markx3/agentboard/internal/db"
	"github.com/markx3/agentboard/internal/peersync"
	"github.com/markx3/agentboard/internal/server"
	"github.com/markx3/agentboard/internal/tunnel"
)

var servePort int
var serveHost string
var serveTunnel bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start as a dedicated server (persistent mode, no TUI)",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 0, "port to listen on (0 for random)")
	serveCmd.Flags().StringVar(&serveHost, "bind", "127.0.0.1", "address to bind to")
	serveCmd.Flags().BoolVar(&serveTunnel, "tunnel", false, "expose server via ngrok tunnel (requires NGROK_AUTHTOKEN)")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Warn if --bind or --port are used alongside --tunnel
	if serveTunnel {
		if cmd.Flags().Changed("bind") || cmd.Flags().Changed("port") {
			fmt.Fprintf(os.Stderr, "Warning: --bind and --port are ignored when --tunnel is active\n")
		}
	}

	dbPath := filepath.Join(".agentboard", "board.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer database.Close()

	svc := boardpkg.NewLocalService(database)
	srv := server.New(svc, serveHost, servePort)

	if serveTunnel {
		return runServeTunnel(ctx, srv)
	}

	return runServeLocal(ctx, srv)
}

func runServeTunnel(ctx context.Context, srv *server.Server) error {
	fmt.Fprintf(os.Stderr, "Creating ngrok tunnel...\n")

	ln, err := tunnel.Listen(ctx)
	if err != nil {
		return err
	}

	tunnelURL := tunnel.URLFromListener(ln)
	srv.SetListener(ln)

	log.Printf("ngrok tunnel established: %s", tunnelURL)
	fmt.Fprintf(os.Stderr, "\nTunnel URL: %s\n", tunnelURL)
	fmt.Fprintf(os.Stderr, "Share this with peers: agentboard --connect %s\n\n", tunnelURL)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	<-ctx.Done()
	return <-errCh
}

func runServeLocal(ctx context.Context, srv *server.Server) error {
	go func() {
		<-ctx.Done()
		peersync.RemoveServerInfo()
	}()

	fmt.Fprintf(os.Stderr, "Starting agentboard server on %s:%d...\n", serveHost, servePort)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start(ctx)
	}()

	// Wait for the server to be ready (address assigned)
	var addr string
	for i := 0; i < 50; i++ {
		addr = srv.Addr()
		if addr != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if addr == "" {
		return fmt.Errorf("server did not become ready in time")
	}

	// Write server info for peer discovery
	if err := peersync.WriteServerInfo(addr); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not write server info: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Server running at %s\n", addr)
	<-ctx.Done()

	return <-errCh
}
