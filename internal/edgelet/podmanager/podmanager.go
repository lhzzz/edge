package podmanager

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type PodManager interface {
	CreatePod(ctx context.Context, pod *v1.Pod) error
	UpdatePod(ctx context.Context, pod *v1.Pod) error
	DeletePod(ctx context.Context, pod *v1.Pod) error
	GetPods(ctx context.Context)
	GetPodStatus(ctx context.Context)
	GetContainerLogs(ctx context.Context)
}

func NewPodManager() PodManager {
	// if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
	// 	return newKubernetesManager()
	// }
	return newDockerComposeManager()
}
