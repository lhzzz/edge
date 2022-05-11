package edgectl

import (
	"edge/internal/edgectl/cmd"
	"flag"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

var (
	edgectlConf = cmd.EdgeCtlConfig{
		EdgeletAddress: ":50051",
	}
)

func NewEdgeCtlCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "edgectl COMMAND [arg...]",
		Short: "edgectl use to connect cloud-cluster",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	globalFlagSet(nil)
	cmds.ResetFlags()

	cmds.AddCommand(cmd.NewJoinCMD())
	cmds.AddCommand(cmd.NewResetCMD())
	return cmds
}

func globalFlagSet(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	flagset.StringVar(&edgectlConf.EdgeletAddress, "edgelet-address", ":50051", "connect edgelet to communicate cloud-cluster")
	pflag.CommandLine.AddGoFlagSet(flagset)
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	flag.Parse()
}
