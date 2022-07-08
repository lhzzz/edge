package cmd

import (
	"context"
	"edge/api/edge-proto/pb"
	"fmt"
	"io"
	"strings"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var (
	edgeletUpgradeDescription = dedent.Dedent(`
	when upgrade edgelet match error 'signal: terminated', it might be edgelet is upgrading itself.
	can use 'edgectl version' to check version later
	`)
)

type upgradeOptions struct {
	nodeName  string   //node的名字
	component string   //组件的名字
	image     string   //镜像的名字
	shellCmds []string //自定义命令
	writer    io.Writer
	stderr    io.Writer
}

func NewUpgradeCMD(out, stderr io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	upgradeOptions := newUpgradeOptions()
	upgradeOptions.writer = out
	upgradeOptions.stderr = stderr
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "upgrade a component on edge",
		RunE: func(cmd *cobra.Command, args []string) error {
			return upgradeRunner(cfg.EdgeletAddress, upgradeOptions)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cfg.EdgeletAddress == "" {
				return fmt.Errorf("edgelet address is empty")
			}
			if upgradeOptions.component == "" {
				return fmt.Errorf("please enter component")
			}
			if upgradeOptions.image == "" {
				return fmt.Errorf("upgrade need specify image ")
			}
			return nil
		},
	}
	addUpgradeFlags(cmd.Flags(), upgradeOptions)
	return cmd
}

func newUpgradeOptions() *upgradeOptions {
	return &upgradeOptions{}
}

func addUpgradeFlags(flagSet *pflag.FlagSet, uo *upgradeOptions) {
	flagSet.StringVar(&uo.nodeName, "nodeName", "", "Specify the node name to upgrade the edgeNode component.")
	flagSet.StringVar(&uo.component, "component", "", "Specify the component to upgrade")
	flagSet.StringVar(&uo.image, "image", "", "Specify the image to upgrade component.")
	flagSet.StringArrayVar(&uo.shellCmds, "cmd", nil, "Customize the upgrade shell command.")
	flagSet.MarkHidden("cmd")
}

func upgradeRunner(edgeletAddress string, opt *upgradeOptions) error {
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(opt.stderr, "connect edgeletAddress %s failed, err=%v\n", edgeletAddress, err)
		return nil
	}
	client := pb.NewEdgeadmClient(conn)

	component := pb.EdgeComponent_UNKNOW
	if value, ok := pb.EdgeComponent_value[strings.ToUpper(opt.component)]; ok {
		component = pb.EdgeComponent(value)
	}
	ctx := metadata.AppendToOutgoingContext(context.Background(), "node", opt.nodeName)
	req := &pb.UpgradeRequest{
		Component: component,
		Image:     opt.image,
		ShellCmds: opt.shellCmds,
	}
	resp, err := client.Upgrade(ctx, req)
	if err != nil {
		fmt.Fprintf(opt.stderr, "upgrade %s match err=%v\n", opt.component, err)
		return nil
	}
	if resp.Error != nil {
		fmt.Fprintf(opt.stderr, "upgrade %s failed, err=%v\n", opt.component, resp.Error.Msg)
		if component == pb.EdgeComponent_EDGELET {
			fmt.Println(edgeletUpgradeDescription)
		}
		return nil
	}
	fmt.Printf("Upgrade Component %s Success !\n", opt.component)
	return nil
}
