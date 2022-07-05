package cmd

import "edge/internal/constant"

type EdgeCtlConfig struct {
	EdgeletAddress string
	ConfigPath     string //配置路径
}

func NewEdgeCtlConfig() EdgeCtlConfig {
	return EdgeCtlConfig{
		EdgeletAddress: constant.EdgeletDefaultAddress,
		ConfigPath:     constant.EdgectlConfPath,
	}
}
