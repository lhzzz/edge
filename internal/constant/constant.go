package constant

const (
	EdgeNameSpace = "edge-cluster"
	CenterDomain  = "center.edge.com"
)

const (
	DockerCopyEdgeletCmd = "docker run --rm -v /root:/myapp %s cp /usr/local/bin/edgelet /myapp"
	UpdageEdgeletCmd     = "mv /root/edgelet /usr/bin/ && systemctl restart edgelet"

	DockerCopyEdgectlCmd = "docker run --rm -v /root:/myapp %s cp /usr/local/bin/edgectl /myapp"
	UpdateEdgectlCmd     = "mv /root/edgectl /usr/bin/"
)

const (
	EdgeletDefaultAddress = ":10350"
	EdgeletDurablePath    = "/data/edgelet/"
	EdgeletCfgPath        = "/data/edgelet/.conf"
)
