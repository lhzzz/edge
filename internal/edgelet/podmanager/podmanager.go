package podmanager

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/podmanager/config"
	"edge/internal/edgelet/podmanager/dockercompose"
	"io"

	v1 "k8s.io/api/core/v1"
)

type PodManager interface {
	CreatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error)
	UpdatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error)
	DeletePod(ctx context.Context, pod *v1.Pod) error
	GetPod(ctx context.Context, namespace, name string) (*v1.Pod, error)
	GetPods(ctx context.Context) ([]*v1.Pod, error)
	GetContainerLogs(ctx context.Context, namespace, podname, containerName string, opts *pb.ContainerLogOptions) (io.ReadCloser, error)
	DescribePodsStatus(ctx context.Context) ([]*v1.Pod, error)
	CreateVolume(ctx context.Context, volume *pb.CreateVolumeRequest) error
	ContainerRuntimeVersion(ctx context.Context) string
}

func New(opts ...config.Option) PodManager {
	return dockercompose.NewPodManager(opts...)
}
