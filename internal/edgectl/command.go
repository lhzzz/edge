package edgectl

import (
	"edge/internal/edgectl/cmd"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
)

var (
	edgectlConf = cmd.EdgeCtlConfig{
		EdgeletAddress: ":10350",
	}
)

func NewEdgeCtlCommand(in io.Reader, out, err io.Writer) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "edgectl COMMAND [arg...]",
		Short: "edgectl use to connect cloud-cluster",
		Long:  "The edgectl is the command-line tool to control edgelet which is connect with cloud-cluster",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
			}
		},
	}

	globalFlagSet(nil)
	cmds.ResetFlags()

	cmds.AddCommand(cmd.NewJoinCMD(os.Stdout, &edgectlConf))
	cmds.AddCommand(cmd.NewResetCMD(os.Stdout, &edgectlConf))
	cmds.AddCommand(cmd.NewInitCmd())
	//cmds.SetUsageFunc(edgeCtlUsageFunc)
	//cmds.SetHelpFunc(edgeCtlHelpFunc)
	return cmds
}

func edgeCtlUsageFunc(cmd *cobra.Command) error {
	usageFmt := "Usage:\n  %s\n"
	fmt.Fprintf(cmd.OutOrStderr(), usageFmt, cmd.UseLine())
	return nil
}

func edgeCtlHelpFunc(cmd *cobra.Command, s []string) {
	usageFmt := "Usage:\n  %s\n"
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n"+usageFmt, cmd.Long, cmd.UseLine())
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		fmt.Fprintf(cmd.OutOrStdout(), "FLAG: --%s=%q\n", flag.Name, flag.Value)
	})
}

func globalFlagSet(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	flagset.StringVar(&edgectlConf.EdgeletAddress, "edgelet-address", ":10350", "connect edgelet to communicate cloud-cluster")
	pflag.CommandLine.AddGoFlagSet(flagset)
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	flag.Parse()
}
