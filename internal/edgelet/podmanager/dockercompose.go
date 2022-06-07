package podmanager

import (
	"context"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	v1 "k8s.io/api/core/v1"
)

type dcpPodManager struct {
	api api.Service
}

//Docker Compose版本必须要在V1.25.2以上
func newDockerComposeManager() *dcpPodManager {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		panic(err)
	}
	dockerCli.Initialize(flags.NewClientOptions())
	apiserver := compose.NewComposeService(dockerCli)
	return &dcpPodManager{
		api: apiserver,
	}
}

//将k8s的pod转换为docker compose中的
func (d *dcpPodManager) CreatePod(ctx context.Context, pod *v1.Pod) error {
	d.api.Create(ctx, &types.Project{Name: ""}, api.CreateOptions{})
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
