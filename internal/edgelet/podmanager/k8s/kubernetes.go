package k8s

import (
	"context"
	"edge/internal/edgelet/podmanager/config"

	v1 "k8s.io/api/core/v1"
)

type k8sPodManager struct {
}

func NewPodManager(opts ...config.Option) *k8sPodManager {
	return &k8sPodManager{}
}

func (k *k8sPodManager) CreatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	return nil, nil
}

func (k *k8sPodManager) UpdatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	return nil, nil
}

func (k *k8sPodManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	return nil
}

func (k *k8sPodManager) GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error) {
	return nil, nil
}

func (k *k8sPodManager) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	return nil, nil
}

func (k *k8sPodManager) GetPodStatus(ctx context.Context, namespace, name string) (*v1.PodStatus, error) {
	return nil, nil
}

func (k *k8sPodManager) GetContainerLogs(ctx context.Context) {

}
