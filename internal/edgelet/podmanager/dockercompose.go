package podmanager

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type dcpPodManager struct {
}

func newDockerComposeManager() *dcpPodManager {
	return &dcpPodManager{}
}

func (d *dcpPodManager) CreatePod(ctx context.Context, pod *v1.Pod) error {
	return nil
}

func (d *dcpPodManager) UpdatePod(ctx context.Context) {

}

func (d *dcpPodManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	return nil
}

func (d *dcpPodManager) GetPods(ctx context.Context) {

}

func (d *dcpPodManager) GetPodStatus(ctx context.Context) {

}

func (d *dcpPodManager) GetContainerLogs(ctx context.Context) {

}
