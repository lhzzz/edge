package main

import (
	app "edge/internal/edgectl"
	goflag "flag"
	"os"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

var (
	buildVersion = "N/A"
)

func main() {
	defer app.Quit()

	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	cmd := app.NewEdgeCtlCommand(os.Stdin, os.Stdout, os.Stderr, buildVersion)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
