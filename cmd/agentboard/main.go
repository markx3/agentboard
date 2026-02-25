package main

import (
	"os"

	"github.com/markx3/agentboard/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
