package cmd

import (
	"io"

	"github.com/spf13/cobra"
)

func NewResetCMD(out io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "edge reset the bind with cloud-cluster",
	}
}
