package server

import (
	"edge/internal/edge-registry/option"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type EdgeRegistryServer struct {
	edgeConfig *option.EdgeRegistryOptions
	stopCh     <-chan struct{}
}

func CreateEdgeRegistry(stopCh <-chan struct{}) (*EdgeRegistryServer, error) {
	if err := initResource(); err != nil {
		return nil, err
	}

	return &EdgeRegistryServer{
		edgeConfig: option.NewDefaultOptions(),
		stopCh:     stopCh,
	}, nil
}

func (es *EdgeRegistryServer) Run() {
	r := gin.Default()
	registry := r.Group("/edge/registry")
	{
		registry.POST("/node", createNode)
		registry.DELETE("/node", deleteNode)
		registry.GET("/node/:nodeName", describeNode)
	}
	<-es.stopCh
	logrus.Info("Received a program exit signal")
	return
}
