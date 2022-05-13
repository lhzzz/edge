package server

import (
	"edge/pkg/kubeclient"

	"edge/internal/constant/manifests"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

func initResource() error {
	if err := kubeclient.CreateResourceWithFile(getK8sClient(), manifests.EdgeIngressYaml, nil); err != nil {
		return err
	}
	return nil
}
