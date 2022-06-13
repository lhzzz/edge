package podmanager

import (
	"context"
	"os"

	"edge/internal/edgelet/podmanager/config"
	"edge/internal/edgelet/podmanager/dockercompose"
	"edge/internal/edgelet/podmanager/k8s"

	v1 "k8s.io/api/core/v1"
)

type PodManager interface {
	CreatePod(ctx context.Context, pod *v1.Pod) error
	UpdatePod(ctx context.Context, pod *v1.Pod) error
	DeletePod(ctx context.Context, pod *v1.Pod) error
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)
	GetPods(ctx context.Context) ([]*v1.Pod, error)
	GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error)
	GetContainerLogs(ctx context.Context)
}

func New(opts ...config.Option) PodManager {
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return k8s.NewPodManager(opts...)
	}
	opts = append(opts, config.WithProjectName("compose"))
	return dockercompose.NewPodManager(opts...)
}
