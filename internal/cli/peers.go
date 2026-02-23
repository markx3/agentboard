package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var peersCmd = &cobra.Command{
	Use:   "peers",
	Short: "List connected peers",
	RunE:  runPeers,
}

func init() {
	rootCmd.AddCommand(peersCmd)
}

func runPeers(cmd *cobra.Command, args []string) error {
	fmt.Println("No peers connected (server not running)")
	// Will be wired to WebSocket client in phase 3
	return nil
}
