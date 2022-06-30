/*
	Note: Docker Compose NEED A Project Name to specify the application Group !!!
*/

package dockercompose

import (
	"context"
	"edge/api/edge-proto/pb"
	pmconf "edge/internal/edgelet/podmanager/config"
	"edge/pkg/errdefs"
	"edge/pkg/util"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/compose/v2/pkg/compose"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	k8sNamespaceLabel = "k8s-namespace"
	k8sPodInfoLabel   = "k8s-podinfo"
	k8sPodNameLabel   = "k8s-podname"
	k8sInitContainer  = "k8s-initContainer"
	always            = "always"

	restartTimes = 5
)

type containerReason string

const (
	initErrorReason        containerReason = "Init:Error"
	completedReason        containerReason = "Completed"
	crashLoopBackOffReason containerReason = "CrashLoopBackOff"
	errorReason            containerReason = "Error"
	recreateReason         containerReason = "ReCreated"
	pendingReason          containerReason = "Pending"
)

var (
	//errMissingLabel is returned when pod missing a service name
	errMissingMeta = errors.New("missing metaName")

	//health service dependency
	serviceHealthDependency = types.ServiceDependency{Condition: types.ServiceConditionHealthy}

	//init-container dependency
	serviceCompeleteDependency = types.ServiceDependency{Condition: types.ServiceConditionCompletedSuccessfully}

	//Pending Container
	pendingContainerState = v1.ContainerState{Waiting: &v1.ContainerStateWaiting{Reason: string(pendingReason)}}

	//default fields entry
	k8sManagerFieldsEntry = []metav1.ManagedFieldsEntry{
		{Manager: "kube-controller-manager", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1"},
		{Manager: "virtual-kubelet", Operation: metav1.ManagedFieldsOperationUpdate, APIVersion: "v1"},
	}
)

type dcpPodManager struct {
	pmconf.Config
	composeApi api.Service
	dockerCli  command.Cli
	podEvents  map[string]struct{}
	eventMutex sync.RWMutex
	podMutex   sync.Mutex
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
	dockerCli.Initialize(options)
	composeAPI := compose.NewComposeService(dockerCli)
	dcp := &dcpPodManager{
		dockerCli:  dockerCli,
		composeApi: composeAPI,
		Config:     conf,
		podEvents:  map[string]struct{}{},
	}
	go dcp.handleEvent(func(event api.Event) error {
		dcp.eventMutex.Lock()
		podName, _ := parseContainerServiceName(event.Service)
		dcp.podEvents[podName] = struct{}{}
		dcp.eventMutex.Unlock()
		return nil
	})
	return dcp
}

func (d *dcpPodManager) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) error {
	for _, v := range req.Vols {
		switch vol := v.Volumn.(type) {
		case *pb.EdgeVolume_EmptyDir:
			path := filepath.Join(d.EmptyDirRoot(), v.Name)
			if util.IsFileExist(path) {
				continue
			}
			err := os.MkdirAll(path, 0755)
			if err != nil {
				return fmt.Errorf("mkdir emptydir failed,err=%v", err)
			}
		case *pb.EdgeVolume_HostPath:
			path := vol.HostPath.Path
			if util.IsFileExist(path) {
				continue
			}
			hostType := v1.HostPathType(vol.HostPath.HostType)
			if hostType == v1.HostPathDirectory || hostType == v1.HostPathDirectoryOrCreate {
				err := os.MkdirAll(path, 0755)
				if err != nil {
					return fmt.Errorf("mkdir hostPath failed,err=%v", err)
				}
			} else if hostType == v1.HostPathFile || hostType == v1.HostPathFileOrCreate {
				f, err := os.Create(path)
				if err != nil {
					return fmt.Errorf("mkdir hostPath failed,err=%v", err)
				}
				f.Close()
			}
		case *pb.EdgeVolume_ConfigMap:
			dirpath := filepath.Join(d.ConfigMapRoot(), vol.ConfigMap.Namespace, v.Name)
			if err := os.MkdirAll(dirpath, 0755); err != nil {
				return fmt.Errorf("mkdir configPath %s failed, err=%v", dirpath, err)
			}
			for name, data := range vol.ConfigMap.Items {
				path := filepath.Join(dirpath, name)
				if util.IsFileExist(path) {
					continue
				}
				f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
				if err != nil {
					return fmt.Errorf("create configmap failed,err=%v", err)
				}
				_, err = f.WriteString(data)
				if err != nil {
					return fmt.Errorf("write configmap %s failed,err=%v", name, err)
				}
				f.Close()
			}
		case *pb.EdgeVolume_Secret:
			dirpath := filepath.Join(d.SecretRoot(), vol.Secret.Namespace, v.Name)
			if err := os.MkdirAll(dirpath, 0755); err != nil {
				return fmt.Errorf("mkdir secretPath %s failed, err=%v", dirpath, err)
			}
			for name, data := range vol.Secret.Items {
				path := filepath.Join(dirpath, name)
				if util.IsFileExist(path) {
					continue
				}
				f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0755)
				if err != nil {
					return fmt.Errorf("create secret failed,err=%v", err)
				}
				_, err = f.Write(data)
				if err != nil {
					return fmt.Errorf("write secret %s failed,err=%v", name, err)
				}
				f.Close()
			}
		}
	}
	return nil
}

//将k8s的pod转换为docker compose中的
func (d *dcpPodManager) CreatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	return d.createOrUpdate(ctx, pod)
}

func (d *dcpPodManager) UpdatePod(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	return d.createOrUpdate(ctx, pod)
}

func (d *dcpPodManager) DeletePod(ctx context.Context, pod *v1.Pod) error {
	pp := NewPodProject(d.Config, pod)
	services := pp.ServiceNames()
	return d.composeApi.Down(ctx, d.Project, api.DownOptions{
		Project: &types.Project{
			Name:     d.Project,
			Services: services,
		},
	})
}

func (d *dcpPodManager) GetPod(ctx context.Context, namespace, podName string) (*v1.Pod, error) {
	f := getDefaultFilters(d.Project)
	if len(namespace) > 0 {
		f = append(f, namespaceFilter(namespace))
	}
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

	inspects := make([]moby.ContainerJSON, len(containers))
	for i, c := range containers {
		inspect, err := d.dockerCli.Client().ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		inspects[i] = inspect
	}

	return d.mobyContainersToK8sPod(inspects...), nil
}

func (d *dcpPodManager) GetPods(ctx context.Context) ([]*v1.Pod, error) {
	podContainers := make(map[string][]moby.ContainerJSON)
	f := getDefaultFilters(d.Project)
	//用docker-compose的api数据被转换，有效信息太少
	containers, err := d.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return nil, err
	}
	//将一个pod下的container分组
	for _, c := range containers {
		serivceName := c.Labels[api.ServiceLabel]
		podName, _ := parseContainerServiceName(serivceName)
		inspect, err := d.dockerCli.Client().ContainerInspect(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		podContainers[podName] = append(podContainers[podName], inspect)
	}
	pods := make([]*v1.Pod, len(podContainers))
	index := 0
	for _, cs := range podContainers {
		pods[index] = d.mobyContainersToK8sPod(cs...)
		index++
	}
	return pods, nil
}

func (d *dcpPodManager) GetContainerLogs(ctx context.Context, namespace, podname, containerName string, opts *pb.ContainerLogOptions) (io.ReadCloser, error) {
	f := getDefaultFilters(d.Project)
	f = append(f, serviceFilter(makeContainerServiceName(podname, containerName)))
	mcs, err := d.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return nil, err
	}
	if len(mcs) == 0 {
		return nil, errdefs.NotFoundf("%s/%s-%s not found", namespace, podname, containerName)
	}

	mopts := moby.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      opts.SinceTime,
		Timestamps: opts.Timestamps,
		Details:    true,
	}
	if opts.Tail > 0 {
		mopts.Tail = fmt.Sprint(opts.Tail)
	}
	return d.dockerCli.Client().ContainerLogs(ctx, mcs[0].ID, mopts)
}

func (d *dcpPodManager) DescribePodsStatus(ctx context.Context) ([]*v1.Pod, error) {
	var podNames []string
	var pods []*v1.Pod
	d.eventMutex.Lock()
	for podName := range d.podEvents {
		podNames = append(podNames, podName)
		delete(d.podEvents, podName)
	}
	d.eventMutex.Unlock()

	for _, podName := range podNames {
		pod, err := d.GetPod(ctx, "", podName)
		if err != nil {
			continue
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

func (d *dcpPodManager) handleEvent(consumer func(event api.Event) error) {
	eventCh, errCh := d.dockerCli.Client().Events(context.Background(), moby.EventsOptions{
		Filters: filters.NewArgs(projectFilter(d.Project)),
	})
	for {
		select {
		case event := <-eventCh:
			if event.Type != events.ContainerEventType {
				continue
			}
			service := event.Actor.Attributes[api.ServiceLabel]
			attributes := map[string]string{}
			for k, v := range event.Actor.Attributes {
				if strings.HasPrefix(k, "com.docker.compose.") {
					continue
				}
				attributes[k] = v
			}
			timestamp := time.Unix(event.Time, 0)
			if event.TimeNano != 0 {
				timestamp = time.Unix(0, event.TimeNano)
			}
			err := consumer(api.Event{
				Timestamp:  timestamp,
				Service:    service,
				Container:  event.ID,
				Status:     event.Status,
				Attributes: attributes,
			})
			if err != nil {
				logrus.Error("handleEvent consumer failed ,err=", err)
			}
		case err := <-errCh:
			logrus.Error("handleEvent receive err ,err=", err)
		}
	}
}

func (d *dcpPodManager) createOrUpdate(ctx context.Context, pod *v1.Pod) (*v1.Pod, error) {
	logrus.Info("podIp:", pod.Status.PodIP, pod.Status.PodIPs)
	d.podMutex.Lock()
	defer d.podMutex.Unlock()
	project := NewPodProject(d.Config, pod).Project()
	err := d.composeApi.Up(ctx, &project, api.UpOptions{
		Create: api.CreateOptions{
			Inherit:              true,
			Recreate:             api.RecreateDiverged,
			RecreateDependencies: api.RecreateDiverged,
			IgnoreOrphans:        true,
		},
		Start: api.StartOptions{Project: &project},
	})
	if err != nil {
		return pod, err
	}
	pod, err = d.GetPod(ctx, pod.Namespace, pod.Name)
	if err != nil {
		return pod, err
	}
	return pod, nil
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

func makeNetworkName(projectName string) (networkField, networkName string) {
	return "default", projectName + "_default"
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
func (d *dcpPodManager) mobyContainersToK8sPod(containers ...moby.ContainerJSON) *v1.Pod {
	if len(containers) == 0 {
		return nil
	}
	pod := v1.Pod{}
	podinfo := containers[0].Config.Labels[k8sPodInfoLabel]
	err := json.Unmarshal([]byte(podinfo), &pod)
	if err != nil {
		logrus.Error("json unmarshal container pod label failed,err=", err)
		return nil
	}
	pod.Status.Phase = v1.PodRunning
	pod.Status.Reason = ""
	pod.Status.PodIP = d.IPAddress
	pod.Status.HostIP = d.IPAddress
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

	initContainers := make(map[string]moby.ContainerJSON)
	runContainers := make(map[string]moby.ContainerJSON)
	for _, c := range containers {
		serviceName := c.Config.Labels[api.ServiceLabel]
		_, podContainerName := parseContainerServiceName(serviceName)
		_, isInit := c.Config.Labels[k8sInitContainer]
		if isInit {
			initContainers[podContainerName] = c
		} else {
			runContainers[podContainerName] = c
		}
	}

	//spec container
	var initStatus, statuses []v1.ContainerStatus
	for _, ic := range pod.Spec.InitContainers {
		mobyContainer, ok := initContainers[ic.Name]
		if ok {
			containerStatus := mobyContainerToK8sContainerState(ic.Name, mobyContainer, true)
			if !containerStatus.Ready {
				pod.Status.Conditions[0].Status = v1.ConditionFalse
				pod.Status.Conditions[1].Status = v1.ConditionFalse
			}
			initStatus = append(initStatus, containerStatus)
		}
	}
	pod.Status.InitContainerStatuses = initStatus

	for _, c := range pod.Spec.Containers {
		mobyContainer, ok := runContainers[c.Name]
		if ok {
			containerStatus := mobyContainerToK8sContainerState(c.Name, mobyContainer, false)
			if !containerStatus.Ready {
				pod.Status.Conditions[1].Status = v1.ConditionFalse
			}
			statuses = append(statuses, containerStatus)
		}
	}
	pod.Status.ContainerStatuses = statuses
	return &pod
}

func mobyContainerToK8sContainerState(podContainerName string, container moby.ContainerJSON, isInit bool) v1.ContainerStatus {
	ret := v1.ContainerStatus{}
	ret.Name = podContainerName
	ret.Image = container.Image
	ret.RestartCount = int32(container.RestartCount)
	ret.Ready = false
	createTime, _ := time.Parse(time.RFC3339Nano, container.Created)
	createAt := metav1.NewTime(createTime)
	if container.State.Running && !container.State.Restarting {
		ret.Ready = true
		ret.State.Running = &v1.ContainerStateRunning{
			StartedAt: createAt,
		}
		return ret
	}

	startTime, _ := time.Parse(time.RFC3339Nano, container.State.StartedAt)
	endtime, _ := time.Parse(time.RFC3339Nano, container.State.FinishedAt)
	if isInit {
		ret.State.Terminated = &v1.ContainerStateTerminated{
			ExitCode:   int32(container.State.ExitCode),
			StartedAt:  metav1.NewTime(startTime),
			FinishedAt: metav1.NewTime(endtime),
		}
		if container.State.ExitCode == 0 {
			ret.Ready = true
			ret.State.Terminated.Reason = string(completedReason)
		} else {
			ret.State.Terminated.Reason = string(initErrorReason)
			ret.State.Terminated.Message = container.State.Error
		}
		return ret
	}

	terminate := &v1.ContainerStateTerminated{
		ExitCode:   int32(container.State.ExitCode),
		Reason:     string(errorReason),
		StartedAt:  metav1.NewTime(startTime),
		FinishedAt: metav1.NewTime(endtime),
	}
	if container.State.ExitCode == 0 {
		terminate.Reason = string(completedReason)
	}

	if ret.RestartCount >= 3 {
		ret.State.Waiting = &v1.ContainerStateWaiting{
			Reason: string(crashLoopBackOffReason),
		}
		ret.LastTerminationState.Terminated = terminate
	} else {
		ret.State.Terminated = terminate
	}
	return ret
}
