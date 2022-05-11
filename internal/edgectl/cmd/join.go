package cmd

import (
	"github.com/spf13/cobra"
)

func NewJoinCMD() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "join",
		Short: "edge join to cloud-cluster",
		Run: func(cmd *cobra.Command, args []string) {

		},
	}
	return cmd
}
