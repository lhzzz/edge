package k8s

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type k8sPodManager struct {
}

func NewPodManager(opts ...ConfigOption) *k8sPodManager {
	return &k8sPodManager{}
}

func (k *k8sPodManager) CreatePod(ctx context.Context, pod *v1.Pod) error {
	return nil
}

func (k *k8sPodManager) UpdatePod(ctx context.Context) {

}

func (k *k8sPodManager) DeletePod(ctx context.Context) {

}

func (k *k8sPodManager) GetPods(ctx context.Context) {

}

func (k *k8sPodManager) GetPodStatus(ctx context.Context) {

}

func (k *k8sPodManager) GetContainerLogs(ctx context.Context) {

}
