package server

import (
	"edge/api/edge-proto/pb"
	"edge/internal/edge-registry/option"
	"edge/pkg/protoerr"
	"net/http"

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

func (es *EdgeRegistryServer) Run(address string) {
	r := gin.Default()
	registry := r.Group("/edge/registry")
	{
		registry.POST("/node", createNode)
		registry.DELETE("/node", deleteNode)
		registry.GET("/node/:nodeName", describeNode)
		registry.GET("/ping", healthCheck)
	}

	go r.Run(address)

	<-es.stopCh
	logrus.Info("Received a program exit signal")
}

/*
1、创建一个deploy和svc给virtual-kubelet ? (svc能否只用一个)
*/
func createNode(c *gin.Context) {
	req := &pb.JoinRequest{}
	resp := &pb.JoinResponse{}
	if err := c.BindJSON(req); err != nil {
		resp.Error = protoerr.ParamErr("not a json fmt")
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	if req.NodeName == "" {
		resp.Error = protoerr.ParamErr("NodeName or Token is empty")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	logrus.Info("request:", req)
	exist, err := createEdgeNode(c.Request.Context(), req.NodeName)
	if err != nil {
		logrus.Error("createEdgeNode failed,err=", err)
		resp.Error = protoerr.InternalErr(err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	resp.Exist = exist
	c.JSON(http.StatusOK, resp)
}

func deleteNode(c *gin.Context) {
	resp := &pb.ResetResponse{}

	nodeName := c.Query("name")
	if nodeName == "" {
		resp.Error = protoerr.ParamErr("NodeName is empty")
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	if err := deleteEdgeNode(c.Request.Context(), nodeName); err != nil {
		logrus.Error("deleteEdgeNode failed,err=", err)
		resp.Error = protoerr.InternalErr(err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func describeNode(c *gin.Context) {

}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
