package cmd

import (
	"context"
	"edge/api/edge-proto/pb"
	"encoding/json"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type versionOptions struct {
	nodeName string
	version  string
}

type version struct {
	EdgectlVersion string `json:"EdgectlVersion"`
	EdgeletVersion string `json:"EdgeletVersion"`
}

func NewVersionCMD(stderr io.Writer, version string, cfg *EdgeCtlConfig) *cobra.Command {
	vo := newVersionOptions()
	vo.version = version
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the version of the edge component",
		Long:  `Show the version of the edge component`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return versionRunner(stderr, cfg.EdgeletAddress, vo)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cfg.EdgeletAddress == "" {
				return fmt.Errorf("edgelet address is empty")
			}
			return nil
		},
	}
	addVersionFlags(cmd.Flags(), vo)
	return cmd
}

func newVersionOptions() *versionOptions {
	return &versionOptions{}
}

func addVersionFlags(flagSet *pflag.FlagSet, vo *versionOptions) {
	flagSet.StringVar(&vo.nodeName, "nodeName", "", "If you wanna get edge version on cloud, you should specify nodeName.")
}

func versionRunner(stderr io.Writer, edgeletAddress string, vo *versionOptions) error {
	ver := version{}
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure())
	if err != nil {
		logrus.Error("connect failed,edgeletAddress:", edgeletAddress, " err:", err)
		return err
	}
	client := pb.NewEdgeadmClient(conn)
	ctx := metadata.AppendToOutgoingContext(context.Background(), "node", vo.nodeName)
	resp, err := client.ListVersion(ctx, &pb.ListVersionRequest{})
	if err != nil {
		fmt.Fprintln(stderr, "listVersion failed,err=", err)
		goto END
	}
	if resp.Error != nil {
		fmt.Fprintln(stderr, "listVersion failed,err=", resp.Error.Msg)
		goto END
	}
	ver.EdgeletVersion = resp.EdgeletVersion
END:
	ver.EdgectlVersion = vo.version
	data, _ := json.Marshal(&ver)
	fmt.Println(string(data))
	return nil
}
