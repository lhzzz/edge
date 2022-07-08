package main

import (
	"edge/internal/constant"
	"edge/internal/edgelet"
	"flag"
)

var (
	listenAddr = flag.String("address", constant.EdgeletDefaultAddress, "edgelet listen address, default is "+constant.EdgeletDefaultAddress)
)

var (
	buildVersion = "N/A"
)

func main() {
	flag.Parse()
	edgelet.Run(*listenAddr, buildVersion)
}
