package server

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/constant/manifests"
	"net/http"
	"sync"

	"edge/pkg/kubeclient"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	cs   *kubernetes.Clientset
	once sync.Once

	pbErrJsonFmt = &pb.Error{Code: pb.ErrorCode_PARAMETER_FAILED, Msg: "not a json fmt"}
	pbErrParam   = func(validateErr string) *pb.Error {
		return &pb.Error{Code: pb.ErrorCode_PARAMETER_FAILED, Msg: validateErr}
	}
	pbErrInternal = func(err error) *pb.Error { return &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()} }
)

func getK8sClient() *kubernetes.Clientset {
	once.Do(func() {
		clientset, err := kubeclient.GetClientSetInCluster()
		if err != nil {
			panic(err.Error())
		}
		cs = clientset
	})
	return cs
}

/*
1、创建一个deploy和svc给virtual-kubelet ? (svc能否只用一个)
2、创建一个ingress给这个svc
3、返回ingress创建的路由
*/
func createNode(c *gin.Context) {
	req := &pb.JoinRequest{}
	resp := &pb.JoinResponse{}
	if err := c.BindJSON(req); err != nil {
		resp.Error = pbErrJsonFmt
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	if req.NodeName == "" || req.Token == "" {
		resp.Error = pbErrParam("NodeName or Token is empty")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	logrus.Info("request:", req)
	namespace := req.NodeName
	option := map[string]string{
		"NodeName":      req.NodeName,
		"NodeNamespace": namespace,
	}

	node, _ := getK8sClient().CoreV1().Nodes().Get(context.TODO(), req.NodeName, v1.GetOptions{})
	if node != nil {
		logrus.Infof("Node %s Has Already Exist", req.NodeName)
		c.JSON(http.StatusOK, resp)
		return
	}

	if err := kubeclient.CreateResourceWithFile(getK8sClient(), manifests.VirtualKubeletYaml, option); err != nil {
		logrus.Error("CreateResourceWithFile failed,err=", err)
		resp.Error = pbErrInternal(err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	// pathType := netv1.PathTypePrefix
	// route := fmt.Sprintf(constant.EdgeIngressPrefixFormat, req.NodeName)
	// path := netv1.HTTPIngressPath{
	// 	Path:     route,
	// 	PathType: &pathType,
	// 	Backend: netv1.IngressBackend{
	// 		Service: &netv1.IngressServiceBackend{
	// 			Name: req.NodeName,
	// 			Port: netv1.ServiceBackendPort{Number: constant.VirtualKubeletDeafultPort},
	// 		},
	// 	},
	// }
	// if err := kubeclient.AppendPathToIngress(getK8sClient(), constant.EdgeNameSpace, constant.EdgeIngress, path); err != nil {
	// 	logrus.Error("create Ingress failed,err=", err)
	// 	resp.Error = pbErrInternal(err)
	// 	c.JSON(http.StatusInternalServerError, resp)
	// 	return
	// }
	c.JSON(http.StatusOK, resp)
}

func deleteNode(c *gin.Context) {
	resp := &pb.ResetResponse{}

	nodeName := c.Query("name")
	if nodeName == "" {
		resp.Error = pbErrParam("NodeName is empty")
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	namespace := nodeName
	option := map[string]string{
		"NodeName":      nodeName,
		"NodeNamespace": namespace,
	}
	if err := kubeclient.DeleteResourceWithFile(getK8sClient(), manifests.VirtualKubeletYaml, option); err != nil {
		logrus.Error("DeleteResourceWithFile failed,err=", err)
		resp.Error = pbErrInternal(err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}
	// route := fmt.Sprintf(constant.EdgeIngressPrefixFormat, nodeName)
	// if err := kubeclient.RemovePathToIngress(getK8sClient(), constant.EdgeNameSpace, constant.EdgeIngress, route); err != nil {
	// 	logrus.Error("remove path in Ingress failed,err=", err)
	// 	resp.Error = pbErrInternal(err)
	// 	c.JSON(http.StatusInternalServerError, resp)
	// 	return
	// }

	c.JSON(http.StatusOK, resp)
}

func describeNode(c *gin.Context) {

}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
