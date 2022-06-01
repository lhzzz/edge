package podmanager

import "os"

type PodManager interface {
	CreatePod()
	UpdatePod()
	DeletePod()
	GetPods()
	GetPodStatus()
	GetContainerLogs()
}

func NewPodManager() PodManager {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return &k8sPodManager{}
	}
	return &dcpPodManager{}
}
