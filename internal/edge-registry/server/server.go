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
	return &EdgeRegistryServer{
		edgeConfig: option.NewDefaultOptions(),
		stopCh:     stopCh,
	}, nil
}

func (es *EdgeRegistryServer) Run() error {
	r := gin.Default()
	registry := r.Group("/edge/registry")
	{
		registry.POST("/node", CreateNode)
		registry.DELETE("/node", DeleteNode)
		registry.GET("/node/:nodeName", DescribeNode)
	}
	<-es.stopCh
	logrus.Info("Received a program exit signal")
	return nil
}
