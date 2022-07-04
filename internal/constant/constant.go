package constant

const (
	EdgeNameSpace           = "edge-cluster"
	EdgeIngress             = "edge-ingress"
	EdgeIngressPrefixFormat = "/edge/node/%s"

	VirtualKubeletDeafultPort = 80
)

const (
	DockerCopyEdgeletCmd = "docker run --rm -v /root:/myapp %s cp /usr/local/bin/edgelet /myapp"
	UpdageEdgeletCmd     = "mv /root/edgelet /usr/bin/ && systemctl restart edgelet"

	DockerCopyEdgectlCmd = "docker run --rm -v /root:/myapp %s cp /usr/local/bin/edgectl /myapp"
	UpdateEdgectlCmd     = "mv /root/edgectl /usr/bin/"
)
