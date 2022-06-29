package server

import (
	"edge/pkg/common"
	"edge/pkg/kubeclient"
	"sync"

	"k8s.io/client-go/kubernetes"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

var (
	cs   *kubernetes.Clientset
	once sync.Once
)

func k8sClient() *kubernetes.Clientset {
	once.Do(func() {
		clientset, err := kubeclient.GetClientSetInCluster()
		if err != nil {
			panic(err.Error())
		}
		cs = clientset
	})
	return cs
}

func initResource() error {
	common.InitLogger()
	return nil
}
