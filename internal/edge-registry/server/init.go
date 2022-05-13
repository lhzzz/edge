package server

import (
	"edge/pkg/kubeclient"

	"edge/internal/constant/manifests"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

func initResource() error {
	if err := kubeclient.CreateResourceWithFile(cs, manifests.EdgeIngressYaml, nil); err != nil {
		return err
	}
	return nil
}
