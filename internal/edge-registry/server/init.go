package server

import (
	"edge/pkg/common"
)

const (
	edgeRegistryIngressName = "edgeRegistry"
)

func initResource() error {
	common.InitLogger()
	return nil
}
