package server

import (
	"edge/pkg/kubeclient"

	netv1 "k8s.io/api/networking/v1"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

func InitResource() error {
	ing := newEdgeRegistryIngress()
	if err := kubeclient.CreateIngressIfNotExist(cs, ing); err != nil {
		return err
	}
	return nil
}

func newEdgeRegistryIngress() *netv1.Ingress {
	return &netv1.Ingress{}
}
