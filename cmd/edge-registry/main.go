package main

import (
	"edge/internal/edge-registry/server"
	"edge/pkg/util"

	"github.com/sirupsen/logrus"
)

func main() {
	er, err := server.CreateEdgeRegistry(util.SetupSignalHandler())
	if err != nil {
		logrus.Fatal("CreateEdgeRegistry failed,err=", err)
	}
	//er.Run(":80")
	er.RunGrpc(":80")
}
