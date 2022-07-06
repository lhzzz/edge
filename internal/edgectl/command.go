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
	edgectlConf *cmd.EdgeCtlConfig
)

func NewEdgeCtlCommand(in io.Reader, stdout, stderr io.Writer, version string) *cobra.Command {
	cmds := &cobra.Command{
		Use:   "edgectl COMMAND [arg...]",
		Short: "edgectl use to connect cloud-cluster",
		Long:  "The edgectl is the command-line tool to control edgelet which is connect with cloud-cluster",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	conf, err := cmd.NewEdgeCtlConfig()
	if err != nil {
		panic(err)
	}
	edgectlConf = conf
	globalFlagSet(nil)
	cmds.ResetFlags()
	cmds.AddCommand(cmd.NewJoinCMD(stdout, stderr, edgectlConf))
	cmds.AddCommand(cmd.NewResetCMD(stderr, edgectlConf))
	cmds.AddCommand(cmd.NewUpgradeCMD(stdout, stderr, edgectlConf))
	cmds.AddCommand(cmd.NewInitCmd(edgectlConf))
	cmds.AddCommand(cmd.NewVersionCMD(stderr, version, edgectlConf))
	return cmds
}

func Quit() {
	edgectlConf.Save()
}

func globalFlagSet(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	flagset.StringVar(&edgectlConf.EdgeletAddress, "edgelet-address", edgectlConf.EdgeletAddress, "connect edgelet to communicate cloud-cluster")
	pflag.CommandLine.AddGoFlagSet(flagset)
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	flag.Parse()
}
