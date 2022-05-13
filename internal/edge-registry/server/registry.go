package server

import (
	"edge/api/pb"
	"net/http"
	"sync"

	"edge/pkg/kubeclient"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	cs   *kubernetes.Clientset
	once sync.Once
)

const (
	edgeNodeIngress = "edgeNodeRegistry"
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
	if err := kubeclient.CreateNamespaceIfNotExist(cs, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: namespace},
	}); err != nil {
		logrus.Error("create Namespace failed,err=", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	deploy := newEdgeNodeDeployment(namespace, req.NodeName)
	if err := kubeclient.CreateOrUpdateDeployment(cs, deploy); err != nil {
		logrus.Error("create Deployment failed,err=", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	svc := newEdgeNodeService(namespace, req.NodeName)
	if err := kubeclient.CreateOrUpdateService(cs, svc); err != nil {
		logrus.Error("create Services failed,err=", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	pathType := netv1.PathTypePrefix
	path := netv1.HTTPIngressPath{
		Path:     "/",
		PathType: &pathType,
		Backend: netv1.IngressBackend{
			Service: &netv1.IngressServiceBackend{
				Name: svc.GetName(),
				Port: netv1.ServiceBackendPort{Number: 80},
			},
		},
	}
	if err := kubeclient.AppendPathToIngress(cs, "", edgeRegistryIngressName, path); err != nil {
		logrus.Error("create Ingress failed,err=", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	resp := &pb.JoinResponse{}
	c.JSON(http.StatusOK, &resp)
}

func deleteNode(c *gin.Context) {

}

func describeNode(c *gin.Context) {

}

func newEdgeNodeDeployment(namespace, nodeName string) *appsv1.Deployment {
	return &appsv1.Deployment{}
}

func newEdgeNodeService(namespace, nodeName string) *corev1.Service {
	return &corev1.Service{}
}
