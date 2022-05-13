package main

import (
	"edge/internal/edge-registry/server"
	"edge/pkg/util"

	"github.com/sirupsen/logrus"
)

func main() {
	rs, err := server.CreateEdgeRegistry(util.SetupSignalHandler())
	if err != nil {
		logrus.Fatal("CreateEdgeRegistry failed,err=", err)
	}
	rs.Run(":80")
}
