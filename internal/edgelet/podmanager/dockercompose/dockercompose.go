/*
	Note: Docker Compose NEED A Project Name to specify the application Group !!!
*/

package dockercompose

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	pmconf "edge/internal/edgelet/podmanager/config"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	k8sNamespaceLabel = "k8s-namespace"
	always            = "always"
)

var (
	//errMissingLabel is returned when pod missing a service name
	errMissingMeta = errors.New("missing metaName")

	//health service dependency
	serviceHealthDependency = types.ServiceDependency{Condition: "service_healthy"}

	//default fields entry
	k8sManagerFieldsEntry = []metav1.ManagedFieldsEntry{
		{Manager: "kube-controller-manager", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1"},
		{Manager: "virtual-kubelet", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1"},
	}
)

type dcpPodManager struct {
	composeApi api.Service
	dockerCli  command.Cli
	project    string
}

//Docker Compose版本必须要在V2.0 以上
func NewPodManager(opts ...pmconf.Option) *dcpPodManager {
	conf := pmconf.DefaultConfig()
	for _, o := range opts {
		o.Apply(&conf)
	}
	if conf.Project == "" {
		panic("missing project name")
	}
	dockerCli, err := command.NewDockerCli()
	if err != nil {
		panic(err)
	}
	options := flags.NewClientOptions()
	options.ConfigDir = filepath.Dir(config.Dir())
	logrus.Info("config:", options.ConfigDir)
	dockerCli.Initialize(options)
	return &dcpPodManager{
		dockerCli:  dockerCli,
		composeApi: compose.NewComposeService(dockerCli),
		project:    conf.Project,
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
	podName := parseK8sPodName(pod)
	if podName == "" {
		return errMissingMeta
	}
	services := listPodContainerService(pod)
	return d.composeApi.Down(ctx, d.project, api.DownOptions{
		Project: &types.Project{
			Name:     d.project,
			Services: services,
		},
	})
}

func (d *dcpPodManager) GetPod(ctx context.Context, namespace, podname string) (*v1.Pod, error) {

	return nil, nil
}

func (d *dcpPodManager) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	cs := make(map[string]moby.Container)
	f := getDefaultFilters(d.project)
	//用docker-compose的api数据被转换，有效信息太少
	containers, err := d.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return nil, err
	}

	//duplicate
	for _, c := range containers {
		serivceName := c.Labels[api.ServiceLabel]
		podName, _ := parseContainerServiceName(serivceName)
		if _, ok := cs[podName]; !ok {
			cs[podName] = c
		}
	}

	ret := make([]*v1.Pod, len(cs))
	index := 0
	for _, c := range cs {
		ret[index] = containerToK8sPod(c)
		index++
	}
	return ret, nil
}

func (d *dcpPodManager) GetPodStatus(ctx context.Context, namespace, podName string) (*v1.PodStatus, error) {
	return nil, nil
}

func (d *dcpPodManager) GetContainerLogs(ctx context.Context) {

}

func (d *dcpPodManager) createOrUpdate(ctx context.Context, pod *v1.Pod) error {
	project := d.newDefaultDockerComposeProject()
	project.Services = k8sContainersToServices(pod, d.project)
	project.Volumes = k8sVolumeToVolume(pod.Spec.Volumes)
	err := d.composeApi.Up(ctx, &project, api.UpOptions{
		Create: api.CreateOptions{Inherit: true, Recreate: "force"},
		Start:  api.StartOptions{Project: &project},
	})
	return err
}

func (d *dcpPodManager) newDefaultDockerComposeProject() types.Project {
	return types.Project{Name: d.project}
}

//init-container依赖于上一个init-container的启动
//容器依赖于所有init-container的启动
func k8sContainersToServices(pod *v1.Pod, projectName string) types.Services {
	services := types.Services{}
	lastServiceName := ""
	initServiceNames := []string{}
	for i, ic := range pod.Spec.InitContainers {
		svrconf := k8sContainer2ServiceConfig(pod, ic, projectName)
		if i != 0 {
			svrconf.DependsOn = types.DependsOnConfig{
				lastServiceName: serviceHealthDependency,
			}
		}
		lastServiceName = svrconf.Name
		services = append(services, svrconf)
		initServiceNames = append(initServiceNames, svrconf.Name)
	}
	for _, c := range pod.Spec.Containers {
		svrconf := k8sContainer2ServiceConfig(pod, c, projectName)
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

func parseK8sPodName(pod *v1.Pod) (podName string) {
	podName = pod.ObjectMeta.Name
	return
}

func makeContainerServiceName(podName, containerName string) string {
	return podName + "." + containerName
}

func parseContainerServiceName(serviceName string) (podName, containerName string) {
	slice := strings.Split(serviceName, ".")
	if len(slice) < 2 {
		return "", ""
	}
	i := 1
	podName = slice[0]
	for ; i < len(slice)-1; i++ {
		podName += "." + slice[i]
	}
	containerName = slice[i]
	return
}

func newDefaultDockerComposeLabels(objectMeta metav1.ObjectMeta, project, service string) types.Labels {
	labels := types.Labels{}
	labels.Add(api.ProjectLabel, project)
	labels.Add(api.ServiceLabel, service)
	labels.Add(api.OneoffLabel, "False")
	labels.Add(k8sNamespaceLabel, objectMeta.Namespace)
	for k, v := range objectMeta.Labels {
		labels.Add(k, v)
	}
	return labels
}

//pod里面的容器转换成docker-compose的service
func k8sContainer2ServiceConfig(pod *v1.Pod, container v1.Container, project string) types.ServiceConfig {
	svrconf := types.ServiceConfig{}
	podName := parseK8sPodName(pod)
	svrconf.Name = makeContainerServiceName(podName, container.Name)
	svrconf.Command = append(container.Command, container.Args...)
	svrconf.Image = container.Image
	svrconf.Labels = newDefaultDockerComposeLabels(pod.ObjectMeta, project, svrconf.Name)
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
	//TODO:port转换有问题
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

//获取pod下的所有容器的service
func listPodContainerService(pod *v1.Pod) []types.ServiceConfig {
	var rets []types.ServiceConfig
	podName := parseK8sPodName(pod)
	for _, ic := range pod.Spec.InitContainers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podName, ic.Name)})
	}
	for _, c := range pod.Spec.Containers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podName, c.Name)})
	}
	return rets
}

func projectFilter(projectName string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", api.ProjectLabel, projectName))
}

func serviceFilter(serviceName string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", api.ServiceLabel, serviceName))
}

func getDefaultFilters(projectName string, selectedServices ...string) []filters.KeyValuePair {
	f := []filters.KeyValuePair{projectFilter(projectName)}
	if len(selectedServices) == 1 {
		f = append(f, serviceFilter(selectedServices[0]))
	}
	return f
}

func containerToK8sPod(containers ...moby.Container) *v1.Pod {
	if len(containers) == 0 {
		return nil
	}
	pod := &v1.Pod{}
	c := containers[0]
	serivceName := c.Labels[api.ServiceLabel]
	podName, _ := parseContainerServiceName(serivceName)
	pod.ObjectMeta = metav1.ObjectMeta{
		Name:      podName,
		Namespace: c.Labels[k8sNamespaceLabel],
		Labels:    dropFilterLabel(c.Labels),
	}
	return nil
}

func dropFilterLabel(labels types.Labels) types.Labels {
	copylabel := types.Labels{}
	for k, v := range labels {
		copylabel[k] = v
	}
	delete(copylabel, k8sNamespaceLabel)
	delete(copylabel, api.ServiceLabel)
	delete(copylabel, api.ProjectLabel)
	delete(copylabel, api.ConfigHashLabel)
	delete(copylabel, api.ContainerNumberLabel)
	delete(copylabel, api.VolumeLabel)
	delete(copylabel, api.NetworkLabel)
	delete(copylabel, api.WorkingDirLabel)
	delete(copylabel, api.ConfigFilesLabel)
	delete(copylabel, api.EnvironmentFileLabel)
	delete(copylabel, api.OneoffLabel)
	delete(copylabel, api.SlugLabel)
	delete(copylabel, api.ImageDigestLabel)
	delete(copylabel, api.DependenciesLabel)
	delete(copylabel, api.VersionLabel)
	return copylabel
}
