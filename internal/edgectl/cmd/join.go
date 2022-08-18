package cmd

import (
	"context"
	"edge/api/edge-proto/pb"
	"fmt"
	"io"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var (
	joinWorkerNodeDoneMsg = dedent.Dedent(`
		This node has joined the cluster:
		* Certificate signing request was sent to apiserver and a response was received.
		* The Edgectl was informed of the new secure connection details.

		
		`)
	joinLongDescription = dedent.Dedent(`
		When joining a cloud initialized cluster, we need to establish
		bidirectional trust. This is split into discovery (having the Node
		trust the Kubernetes Control Plane) and TLS bootstrap (having the
		Kubernetes Control Plane trust the Node).

		Often times the same token is used for both parts. In this case, the
		--token flag can be used instead of specifying each token individually.
		`)
)

type joinOptions struct {
	nodeName        string //node的名字
	registryAddress string //云端的地址
	stdout          io.Writer
	stderr          io.Writer
}

func NewJoinCMD(stdout, stderr io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	joinOptions := newJoinOptions()
	joinOptions.stdout = stdout
	joinOptions.stderr = stderr
	cmd := &cobra.Command{
		Use:   "join",
		Short: "edge join to cloud-cluster",
		Long:  joinLongDescription,
		RunE: func(cmd *cobra.Command, args []string) error {
			return joinRunner(cfg.EdgeletAddress, joinOptions)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cfg.EdgeletAddress == "" {
				return fmt.Errorf("edgelet address is empty")
			}
			if joinOptions.nodeName == "" {
				return fmt.Errorf("please enter node-name")
			}
			return nil
		},
	}
	addJoinFlags(cmd.Flags(), joinOptions)
	return cmd
}

func newJoinOptions() *joinOptions {
	return &joinOptions{}
}

// addJoinOtherFlags adds join flags that are not bound to a configuration file to the given flagset
func addJoinFlags(flagSet *pflag.FlagSet, joinOptions *joinOptions) {
	flagSet.StringVar(
		&joinOptions.nodeName, "node-name", "",
		"Specify the node name.",
	)
	flagSet.StringVar(
		&joinOptions.registryAddress, "registry-address", "",
		"Specify the cloud-cluster registry address.",
	)
}

func joinRunner(edgeletAddress string, opt *joinOptions) error {
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		fmt.Fprintf(opt.stderr, "connect edgeletAddress %s failed, err=%v\n", edgeletAddress, err)
		return nil
	}
	client := pb.NewEdgeadmClient(conn)
	resp, err := client.Join(context.Background(), &pb.JoinRequest{
		NodeName:     opt.nodeName,
		CloudAddress: opt.registryAddress,
	})
	if err != nil {
		fmt.Fprintf(opt.stderr, "Join failed, err=%v\n", err)
		return nil
	}
	if resp.Error != nil {
		fmt.Fprintf(opt.stderr, "Join failed, err=%v\n", resp.Error.Msg)
		return nil
	}
	fmt.Fprintln(opt.stdout, joinWorkerNodeDoneMsg)
	return nil
}
