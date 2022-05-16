package constant

const (
	EdgeNameSpace           = "edge-cluster"
	EdgeIngress             = "edge-ingress"
	EdgeIngressPrefixFormat = "/edge/node/%s"

	VirtualKubeletDeafultPort = 80
)

var (
	VirtualKubeletLabel = map[string]string{"type": "virtual-kubelet"}
)
