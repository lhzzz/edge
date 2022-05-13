package server

import (
	"edge/api/pb"
	"edge/internal/constant"
	"edge/internal/constant/manifests"
	"fmt"
	"net/http"
	"sync"

	"edge/pkg/kubeclient"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	netv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	cs   *kubernetes.Clientset
	once sync.Once
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
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}
	if req.NodeName == "" || req.Token == "" {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	logrus.Info("request:", req)
	namespace := req.NodeName
	option := map[string]string{
		"NodeName":      req.NodeName,
		"NodeNamespace": namespace,
	}
	if err := kubeclient.CreateResourceWithFile(getK8sClient(), manifests.VirtualKubeletYaml, option); err != nil {
		logrus.Error("CreateResourceWithFile failed,err=", err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	pathType := netv1.PathTypePrefix
	route := fmt.Sprintf(constant.EdgeIngressPrefixFormat, req.NodeName)
	path := netv1.HTTPIngressPath{
		Path:     route,
		PathType: &pathType,
		Backend: netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: req.NodeName,
				Port: netv1.ServiceBackendPort{Number: constant.VirtualKubeletDeafultPort},
			},
		},
	}
	if err := kubeclient.AppendPathToIngress(getK8sClient(), "", constant.EdgeIngress, path); err != nil {
		logrus.Error("create Ingress failed,err=", err)
		c.JSON(http.StatusInternalServerError, resp)
		return
	}

	resp.VkUrl = route
	c.JSON(http.StatusOK, resp)
}

func deleteNode(c *gin.Context) {

}

func describeNode(c *gin.Context) {

}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
