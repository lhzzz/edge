package server

import (
	"edge/pkg/common"
	"edge/pkg/kubeclient"

	"edge/internal/constant/manifests"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

func initResource() error {
	common.InitLogger()

	if err := kubeclient.CreateResourceWithFile(getK8sClient(), manifests.EdgeIngressYaml, nil); err != nil {
		return err
	}
	return nil
}
