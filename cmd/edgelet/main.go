package main

import (
	"edge/internal/edgelet"
	"flag"
)

var (
	listenPort = flag.String("address", ":10350", "edgelet listen address, default is ':10350'.")
)

var (
	buildVersion = "N/A"
)

func main() {
	flag.Parse()
	edgelet.Run(*listenPort, buildVersion)
}
