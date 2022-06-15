/*
	Note: Docker Compose NEED A Project Name to specify the application Group !!!
*/

package dockercompose

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	pmconf "edge/internal/edgelet/podmanager/config"
	"edge/pkg/errdefs"

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
	k8sPodInfoLabel   = "k8s-podinfo"
	k8sPodNameLabel   = "k8s-podname"
	always            = "always"
)

type containerState string

const (
	pausedState     containerState = "paused"
	restartingState containerState = "restarting"
	removingState   containerState = "removing"
	runningState    containerState = "running"
	deadState       containerState = "dead"
	createdState    containerState = "created"
	exitedState     containerState = "exited"
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
		panic("missing project name: docker-compose init must specify a project")
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
func (d *dcpPodManager) CreatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	return d.createOrUpdate(ctx, pod)
}

func (d *dcpPodManager) UpdatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
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

func (d *dcpPodManager) GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	f := getDefaultFilters(d.project)
	f = append(f, namespaceFilter(namespace))
	f = append(f, podnameFilter(podName))
	containers, err := d.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, errdefs.NotFoundf("%s/%s not found", namespace, podName)
	}
	return containerToK8sPod(containers...), nil
}

func (d *dcpPodManager) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	podContainers := make(map[string][]moby.Container)
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
		podContainers[podName] = append(podContainers[podName], c)
	}
	ret := make([]*v1.Pod, len(podContainers))
	index := 0
	for _, cs := range podContainers {
		ret[index] = containerToK8sPod(cs...)
		index++
	}
	return ret, nil
}

//获取pod的status
func (d *dcpPodManager) GetPodStatus(ctx context.Context, namespace, podName string) (*v1.PodStatus, error) {
	f := getDefaultFilters(d.project)
	f = append(f, namespaceFilter(namespace))
	f = append(f, podnameFilter(podName))
	containers, err := d.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, errdefs.NotFoundf("%s/%s not found", namespace, podName)
	}
	pod := containerToK8sPod(containers...)
	return &pod.Status, nil
}

func (d *dcpPodManager) GetContainerLogs(ctx context.Context) {

}

func (d *dcpPodManager) createOrUpdate(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	project := d.newDefaultDockerComposeProject()
	project.Services = k8sContainersToServices(pod, d.project)
	project.Volumes = k8sVolumeToVolume(pod.Spec.Volumes)
	err := d.composeApi.Up(ctx, &project, api.UpOptions{
		Create: api.CreateOptions{Inherit: true, Recreate: "force"},
		Start:  api.StartOptions{Project: &project},
	})
	if err != nil {
		logrus.Info("createOrUpdate Pod failed,err=", err)
		return pod, err
	}
	pod, err = d.GetPod(ctx, pod.Namespace, pod.Name)
	if err != nil {
		logrus.Info("GetPod in createOrUpdate failed,err=", err)
		return pod, err
	}
	return pod, nil
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

func newDefaultDockerComposeLabels(pod *v1.Pod, project, service string) types.Labels {
	labels := types.Labels{}
	labels.Add(api.ProjectLabel, project)
	labels.Add(api.ServiceLabel, service)
	labels.Add(api.OneoffLabel, "False")
	labels.Add(k8sNamespaceLabel, pod.ObjectMeta.Namespace)
	labels.Add(k8sPodNameLabel, pod.ObjectMeta.Name)
	jbyte, _ := json.Marshal(pod)
	labels.Add(k8sPodInfoLabel, string(jbyte))
	return labels
}

//pod里面的容器转换成docker-compose的service
func k8sContainer2ServiceConfig(pod *v1.Pod, container v1.Container, project string) types.ServiceConfig {
	svrconf := types.ServiceConfig{}
	podName := parseK8sPodName(pod)
	svrconf.Name = makeContainerServiceName(podName, container.Name)
	svrconf.Command = append(container.Command, container.Args...)
	svrconf.Image = container.Image
	svrconf.Labels = newDefaultDockerComposeLabels(pod, project, svrconf.Name)
	svrconf.CustomLabels = types.Labels{}
	svrconf.Environment = types.MappingWithEquals{}
	for _, e := range container.Env {
		env := e
		svrconf.Environment[env.Name] = &env.Value
	}
	//TODO:健康检测的转换处理
	svrconf.HealthCheck = &types.HealthCheckConfig{}
	svrconf.PullPolicy = types.PullPolicyIfNotPresent
	svrconf.Restart = types.RestartPolicyOnFailure + ":3" //github.com/docker/compose/@v2.6.0/pkg/compose/create.go/getRestartPolicy
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
	//logrus.Info(svrconf)
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

func namespaceFilter(namespace string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", k8sNamespaceLabel, namespace))
}

func podnameFilter(podname string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("%s=%s", k8sPodNameLabel, podname))
}

func getDefaultFilters(projectName string, selectedServices ...string) []filters.KeyValuePair {
	f := []filters.KeyValuePair{projectFilter(projectName)}
	if len(selectedServices) == 1 {
		f = append(f, serviceFilter(selectedServices[0]))
	}
	return f
}

//重点
func containerToK8sPod(containers ...moby.Container) *v1.Pod {
	if len(containers) == 0 {
		return nil
	}
	pod := v1.Pod{}
	podinfo := containers[0].Labels[k8sPodInfoLabel]
	err := json.Unmarshal([]byte(podinfo), &pod)
	if err != nil {
		logrus.Error("json unmarshal container pod label failed,err=", err)
		return nil
	}
	pod.Status.Reset()
	pod.Status.Phase = v1.PodRunning
	pod.Status.Reason = ""
	pod.Status.Conditions = []v1.PodCondition{
		{
			Type:   v1.PodInitialized,
			Status: v1.ConditionTrue,
		},
		{
			Type:   v1.PodReady,
			Status: v1.ConditionTrue,
		},
		{
			Type:   v1.PodScheduled,
			Status: v1.ConditionTrue,
		},
	}

	dockerContainers := make(map[string]moby.Container)
	for _, c := range containers {
		logrus.Infof("podName:%v container:%v state:%v status:%v", pod.Name, c.Names, c.State, c.Status)
		if c.State != string(runningState) {
			pod.Status.Phase = v1.PodUnknown
			pod.Status.Reason = c.Status
		}
		serviceName := c.Labels[api.ServiceLabel]
		_, podContainerName := parseContainerServiceName(serviceName)
		dockerContainers[podContainerName] = c
	}
	//spec container
	for _, c := range pod.Spec.Containers {
		mobyContainer := dockerContainers[c.Name]
		containerStatus := v1.ContainerStatus{
			Name:         c.Name,
			Image:        c.Image,
			State:        containerStateToK8sContainerState(mobyContainer),
			Ready:        true,
			RestartCount: 0,
		}
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, containerStatus)
	}
	logrus.Infof("podname:%v status:%v", pod.Name, pod.Status.Phase)
	return &pod
}

func containerStateToK8sContainerState(container moby.Container) v1.ContainerState {
	ret := v1.ContainerState{}
	cs := containerState(container.State)
	now := metav1.NewTime(time.Now())
	if cs == runningState || cs == createdState {
		ret.Running = &v1.ContainerStateRunning{
			StartedAt: now,
		}
		return ret
	}
	if cs == removingState || cs == deadState || cs == exitedState {
		ret.Terminated = &v1.ContainerStateTerminated{
			Message: container.Status,
		}
		return ret
	}
	if cs == pausedState {
		ret.Waiting = &v1.ContainerStateWaiting{
			Message: container.Status,
		}
		return ret
	}
	return ret
}
