package main

import (
	app "edge/internal/edgectl"
	"os"
)

func main() {
	cmd := app.NewEdgeCtlCommand(os.Stdin, os.Stdout, os.Stderr)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	return
}
