package cmd

import (
	"context"
	"edge/api/edge-proto/pb"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

type resetOptions struct {
	nodeName string //node的名字
	writer   io.Writer
}

func NewResetCMD(out io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	resetOptions := newResetOptions()
	resetOptions.writer = out
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Performs a best effort revert of changes made to this host by 'edgectl join'",
		RunE: func(cmd *cobra.Command, args []string) error {
			return resetRunner(cfg.EdgeletAddress, resetOptions)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cfg.EdgeletAddress == "" {
				return fmt.Errorf("edgelet address is empty")
			}
			if resetOptions.nodeName == "" {
				return fmt.Errorf("please enter node-name")
			}
			return nil
		},
	}
	addResetFlags(cmd.Flags(), resetOptions)
	return cmd
}

func newResetOptions() *resetOptions {
	return &resetOptions{}
}

func addResetFlags(flagSet *pflag.FlagSet, ro *resetOptions) {
	flagSet.StringVar(
		&ro.nodeName, "node-name", "",
		"Specify the node name.",
	)
}

func resetRunner(edgeletAddress string, opt *resetOptions) error {
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure())
	if err != nil {
		logrus.Error("connect failed,edgeletAddress:", edgeletAddress, " err:", err)
		return err
	}
	client := pb.NewEdgeadmClient(conn)
	resp, err := client.Reset(context.Background(), &pb.ResetRequest{
		NodeName: opt.nodeName,
	})
	if err != nil {
		logrus.Error("Reset failed,err=", err)
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf(resp.Error.Msg)
	}
	return nil
}
