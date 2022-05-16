package cmd

import (
	"context"
	"edge/api/pb"
	"fmt"
	"io"

	"github.com/lithammer/dedent"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var (
	joinWorkerNodeDoneMsg = dedent.Dedent(`
		This node has joined the cluster:
		* Certificate signing request was sent to apiserver and a response was received.
		* The Edgectl was informed of the new secure connection details.

		Run 'edgectl get nodes' on the control-plane to see this node join the cluster.

		`)
	joinLongDescription = dedent.Dedent(`
		When joining a cloud initialized cluster, we need to establish
		bidirectional trust. This is split into discovery (having the Node
		trust the Kubernetes Control Plane) and TLS bootstrap (having the
		Kubernetes Control Plane trust the Node).

		There are 2 main schemes for discovery. The first is to use a shared
		token along with the IP address of the API server. The second is to
		provide a file - a subset of the standard kubeconfig file. This file
		can be a local file or downloaded via an HTTPS URL. The forms are
		edgeadm join --discovery-token abcdef.1234567890abcdef 1.2.3.4:6443,
		edgeadm join --discovery-file path/to/file.conf, or edgeadm join
		--discovery-file https://url/file.conf. Only one form can be used. If
		the discovery information is loaded from a URL, HTTPS must be used.
		Also, in that case the host installed CA bundle is used to verify
		the connection.

		If you use a shared token for discovery, you should also pass the
		--discovery-token-ca-cert-hash flag to validate the public key of the
		root certificate authority (CA) presented by the Kubernetes Control Plane.
		The value of this flag is specified as "<hash-type>:<hex-encoded-value>",
		where the supported hash type is "sha256". The hash is calculated over
		the bytes of the Subject Public Key Info (SPKI) object (as in RFC7469).
		This value is available in the output of "edgeadm init" or can be
		calculated using standard tools. The --discovery-token-ca-cert-hash flag
		may be repeated multiple times to allow more than one public key.

		If you cannot know the CA public key hash ahead of time, you can pass
		the --discovery-token-unsafe-skip-ca-verification flag to disable this
		verification. This weakens the edgeadm security model since other nodes
		can potentially impersonate the Kubernetes Control Plane.

		The TLS bootstrap mechanism is also driven via a shared token. This is
		used to temporarily authenticate with the Kubernetes Control Plane to submit a
		certificate signing request (CSR) for a locally created key pair. By
		default, edgeadm will set up the Kubernetes Control Plane to automatically
		approve these signing requests. This token is passed in with the
		--tls-bootstrap-token abcdef.1234567890abcdef flag.

		Often times the same token is used for both parts. In this case, the
		--token flag can be used instead of specifying each token individually.
		`)
)

type joinOptions struct {
	nodeName     string //node的名字
	token        string //cloud分配的token
	cloudAddress string //云端的地址
	writer       io.Writer
}

func NewJoinCMD(out io.Writer, cfg *EdgeCtlConfig) *cobra.Command {
	joinOptions := newJoinOptions()
	joinOptions.writer = out
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
			if joinOptions.cloudAddress == "" {
				return fmt.Errorf("please enter cloudAddress")
			}
			if joinOptions.nodeName == "" {
				return fmt.Errorf("please enter node-name")
			}
			if joinOptions.token == "" {
				return fmt.Errorf("please enter token")
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
		&joinOptions.token, "token", "",
		"Use this token for both discovery-token and tls-bootstrap-token when those values are not provided.",
	)
	flagSet.StringVar(
		&joinOptions.nodeName, "node-name", "",
		"Specify the node name.",
	)
	flagSet.StringVar(
		&joinOptions.cloudAddress, "cloud-address", "",
		"Specify the cloud-cluster address.",
	)
}

func joinRunner(edgeletAddress string, opt *joinOptions) error {
	conn, err := grpc.Dial(edgeletAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("connect failed,edgeletAddress:", edgeletAddress, " err:", err)
		return err
	}
	client := pb.NewEdgeletClient(conn)
	resp, err := client.Join(context.Background(), &pb.JoinRequest{
		NodeName:     opt.nodeName,
		Token:        opt.token,
		CloudAddress: opt.cloudAddress,
	})
	if err != nil {
		logrus.Error("Join failed,err=", err)
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf(resp.Error.Msg)
	}
	fmt.Fprint(opt.writer, joinWorkerNodeDoneMsg)
	return nil
}
