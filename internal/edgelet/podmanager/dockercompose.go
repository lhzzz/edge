/*
	Note: Docker Compose NEED A Project Name to specify the application Group !!!
	So the k8s pod transfer to docker-compose container , must carry project and service message .
*/

package podmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

const (
	k8sProjectLabel = "k8s-project"
	k8sServiceLabel = "k8s-app"
	always          = "always"
)

var (
	//errMissingLabel is returned when pod missing a project or service label
	errMissingLabel = errors.New("missing label")
)

var (
	serviceHealthDependency = types.ServiceDependency{Condition: "service_healthy"}
)

type dcpPodManager struct {
	api api.Service
}

//Docker Compose版本必须要在V2.0 以上
func newDockerComposeManager() *dcpPodManager {
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		panic(err)
	}
	options := flags.NewClientOptions()
	options.ConfigDir = "/home/linhao/"
	_ = config.Dir()
	logrus.Info("config:", options.ConfigDir)
	dockerCli.Initialize(options)
	apiserver := compose.NewComposeService(dockerCli)
	return &dcpPodManager{
		api: apiserver,
	}
}

//将k8s的pod转换为docker compose中的
func (d *dcpPodManager) CreatePod(ctx context.Context, pod *v1.Pod) error {
	return d.createOrUpdate(ctx, pod)
}

func (d *dcpPodManager) UpdatePod(ctx context.Context, pod *v1.Pod) error {
	return d.createOrUpdate(ctx, pod)
}

func (d *dcpPodManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	projectName, serviceName := parsePodProjectAndService(pod)
	if projectName == "" || serviceName == "" {
		return errMissingLabel
	}
	services := parsePodContainerService(pod)
	return d.api.Down(ctx, projectName, api.DownOptions{
		Project: &types.Project{
			Name:     projectName,
			Services: services,
		},
	})
}

func (d *dcpPodManager) GetPods(ctx context.Context) {

}

func (d *dcpPodManager) GetPodStatus(ctx context.Context) {

}

func (d *dcpPodManager) GetContainerLogs(ctx context.Context) {

}

//从上层传来的pod中解析出project和pod的service
func parsePodProjectAndService(pod *v1.Pod) (project, service string) {
	project = pod.Labels[k8sProjectLabel]
	if len(pod.OwnerReferences) > 0 {
		service = pod.OwnerReferences[0].Name
	}
	if service == "" {
		service = pod.Labels[k8sServiceLabel]
	}
	return
}

func makeContainerServiceName(podServiceName, containerName string) string {
	return podServiceName + "." + containerName
}

//获取pod下的所有容器的service
func parsePodContainerService(pod *v1.Pod) []types.ServiceConfig {
	var rets []types.ServiceConfig
	_, podServiceName := parsePodProjectAndService(pod)
	for _, ic := range pod.Spec.InitContainers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podServiceName, ic.Name)})
	}
	for _, c := range pod.Spec.Containers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podServiceName, c.Name)})
	}
	return rets
}

func newDefaultDockerComposeLabels(project, service string) types.Labels {
	labels := types.Labels{}
	labels.Add(api.ProjectLabel, project)
	labels.Add(api.ServiceLabel, service)
	labels.Add(api.OneoffLabel, "False")
	return labels
}

func newDefaultDockerComposeProject(project string) types.Project {
	return types.Project{Name: project}
}

func k8sContainer2ServiceConfig(container v1.Container, project, service string) types.ServiceConfig {
	svrconf := types.ServiceConfig{}
	svrconf.Name = makeContainerServiceName(service, container.Name)
	svrconf.Command = append(container.Command, container.Args...)
	svrconf.Image = container.Image
	svrconf.Labels = newDefaultDockerComposeLabels(project, svrconf.Name)
	svrconf.CustomLabels = types.Labels{}
	svrconf.Environment = types.MappingWithEquals{}
	for _, e := range container.Env {
		env := e
		svrconf.Environment[env.Name] = &env.Value
	}
	//TODO:健康检测的转换处理
	svrconf.HealthCheck = &types.HealthCheckConfig{}
	svrconf.PullPolicy = "always"
	svrconf.Restart = "always"
	svrconf.Scale = 1
	svrconf.Ports = []types.ServicePortConfig{}
	for _, p := range container.Ports {
		svrconf.Ports = append(svrconf.Ports, types.ServicePortConfig{
			Protocol:  string(p.Protocol),
			Published: fmt.Sprint(p.HostPort),
			Target:    uint32(p.ContainerPort),
			HostIP:    p.HostIP,
		})
	}
	//TODO:volumn 的转换处理， configMap/secrets
	svrconf.Volumes = []types.ServiceVolumeConfig{}
	logrus.Info(svrconf)
	return svrconf
}

//init-container依赖于上一个init-container的启动
//容器依赖于所有init-container的启动
func k8sContainersToServices(initContainers, containers []v1.Container, projectName, serviceName string) types.Services {
	services := types.Services{}
	lastServiceName := ""
	initServiceNames := []string{}
	for i, ic := range initContainers {
		svrconf := k8sContainer2ServiceConfig(ic, projectName, serviceName)
		if i != 0 {
			svrconf.DependsOn = types.DependsOnConfig{
				lastServiceName: serviceHealthDependency,
			}
		}
		lastServiceName = svrconf.Name
		services = append(services, svrconf)
		initServiceNames = append(initServiceNames, svrconf.Name)
	}
	for _, c := range containers {
		svrconf := k8sContainer2ServiceConfig(c, projectName, serviceName)
		svrconf.DependsOn = types.DependsOnConfig{}
		for _, isn := range initServiceNames {
			svrconf.DependsOn[isn] = serviceHealthDependency
		}
		services = append(services, svrconf)
	}
	return services
}

func k8sVolumeToVolume(vols []v1.Volume) types.Volumes {
	volumns := types.Volumes{}
	for _, v := range vols {
		volumns[v.Name] = types.VolumeConfig{
			Name: v.Name,
			//TODO:? volume的转换
		}
	}
	return volumns
}

func (d *dcpPodManager) createOrUpdate(ctx context.Context, pod *v1.Pod) error {
	projectName, serviceName := parsePodProjectAndService(pod)
	if projectName == "" || serviceName == "" {
		return errMissingLabel
	}
	project := newDefaultDockerComposeProject(projectName)
	project.Services = k8sContainersToServices(pod.Spec.InitContainers, pod.Spec.Containers, projectName, serviceName)
	project.Volumes = k8sVolumeToVolume(pod.Spec.Volumes)
	err := d.api.Up(ctx, &project, api.UpOptions{
		Create: api.CreateOptions{Inherit: true, Recreate: "force"},
		Start:  api.StartOptions{Project: &project},
	})
	return err
}
