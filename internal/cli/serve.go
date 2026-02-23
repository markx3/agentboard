package cli

import (
	"fmt"

	"github.com/spf13/cobra"
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
	fmt.Printf("Starting agentboard server on %s:%d...\n", serveHost, servePort)
	// Will be wired to WebSocket server in phase 3
	return nil
}
