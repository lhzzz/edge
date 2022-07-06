package cmd

import (
	"context"
	"edge/api/edge-proto/pb"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var (
	resetWorkerNodeDoneMsg = "This node has reset from the cluster"
)

type resetOptions struct {
	writer io.Writer
}

func NewResetCMD(stderr io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	resetOptions := newResetOptions()
	resetOptions.writer = stderr
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
	// flagSet.StringVar(
	// 	&ro.nodeName, "node-name", "",
	// 	"Specify the node name.",
	// )
}

func resetRunner(edgeletAddress string, opt *resetOptions) error {
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(opt.writer, "connect edgeletAddress %s failed, err=%v\n", edgeletAddress, err)
		return nil
	}
	client := pb.NewEdgeadmClient(conn)
	resp, err := client.Reset(context.Background(), &pb.ResetRequest{})
	if err != nil {
		fmt.Fprintln(opt.writer, err)
		return nil
	}
	if resp.Error != nil {
		fmt.Fprintln(opt.writer, resp.Error.Msg)
		return nil
	}
	fmt.Println(resetWorkerNodeDoneMsg)
	return nil
}
