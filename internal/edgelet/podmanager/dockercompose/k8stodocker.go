package dockercompose

import (
	"edge/internal/edgelet/podmanager/config"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	v1 "k8s.io/api/core/v1"
)

const (
	k8sappLabel = "k8s-app"
	appLabel    = "app"
)

const (
	networkModeHost        = "host"
	networkModeBridge      = "bridge"
	networkModeServiceRely = "service:"
)

type dockerComposeProject struct {
	pod    *v1.Pod
	config config.Config
}

type DockerComposeProject interface {
	Project() types.Project
	ServiceNames() []types.ServiceConfig
}

func NewPodProject(conf config.Config, pod *v1.Pod) DockerComposeProject {
	return &dockerComposeProject{
		pod:    pod,
		config: conf,
	}
}

func (dcpp *dockerComposeProject) newDockerComposeLabels(service string, isInit bool) types.Labels {
	labels := types.Labels{}
	labels.Add(api.ProjectLabel, dcpp.config.Project)
	labels.Add(api.ServiceLabel, service)
	labels.Add(api.OneoffLabel, "False")
	labels.Add(k8sNamespaceLabel, dcpp.pod.ObjectMeta.Namespace)
	labels.Add(k8sPodNameLabel, dcpp.pod.ObjectMeta.Name)
	jbyte, _ := json.Marshal(dcpp.pod)
	labels.Add(k8sPodInfoLabel, string(jbyte))
	if isInit {
		labels.Add(k8sInitContainer, "true")
	}
	return labels
}

//volume 的sourcePath转换处理,configMap/secrets
func (dcpp *dockerComposeProject) genSourcePath(mount v1.VolumeMount) string {
	var index int = 0
	for i, vo := range dcpp.pod.Spec.Volumes {
		if vo.Name == mount.Name {
			index = i
			break
		}
	}
	vo := dcpp.pod.Spec.Volumes[index]
	path := ""
	if vo.HostPath != nil {
		path = vo.HostPath.Path
	}
	if vo.ConfigMap != nil {
		path = filepath.Join(dcpp.config.ConfigMapRoot(), dcpp.pod.Namespace, vo.Name)
	}
	if vo.Secret != nil {
		path = filepath.Join(dcpp.config.SecretRoot(), dcpp.pod.Namespace, vo.Name)
	}
	if vo.EmptyDir != nil {
		path = filepath.Join(dcpp.config.EmptyDirRoot(), vo.Name)
	}
	if mount.SubPath != "" {
		path = filepath.Join(path, mount.SubPath)
	}
	return path
}

//TODO:健康检测的转换处理
func (dcpp *dockerComposeProject) toHealthCheck(container v1.Container) *types.HealthCheckConfig {

	return nil
}

//env的处理
func (dcpp *dockerComposeProject) toEnv(container v1.Container) types.MappingWithEquals {
	envs := types.MappingWithEquals{}
	for _, e := range container.Env {
		env := e
		envs[env.Name] = &env.Value
	}
	return envs
}

func (dcpp *dockerComposeProject) toVolumes(container v1.Container) []types.ServiceVolumeConfig {
	vs := []types.ServiceVolumeConfig{}
	for _, v := range container.VolumeMounts {
		source := dcpp.genSourcePath(v)
		if source == "" {
			continue
		}
		volume := types.ServiceVolumeConfig{
			Type:   types.VolumeTypeBind,
			Source: source,
			Target: v.MountPath,
			Bind: &types.ServiceVolumeBind{
				CreateHostPath: true,
			},
		}
		vs = append(vs, volume)
	}
	return vs
}

func (dcpp *dockerComposeProject) toPort(container v1.Container) []types.ServicePortConfig {
	var ports []types.ServicePortConfig
	for _, p := range container.Ports {
		ports = append(ports, types.ServicePortConfig{
			Mode:      "ingress",
			Protocol:  strings.ToLower(string(p.Protocol)),
			Published: fmt.Sprint(p.HostPort),
			Target:    uint32(p.ContainerPort),
		})
	}
	return ports
}

func (dcpp *dockerComposeProject) toNetworkMode(container v1.Container) string {
	if dcpp.pod.Spec.HostNetwork {
		return networkModeHost
	}
	//多容器Pod网络处理，都依赖于第一个容器的网络
	if len(dcpp.pod.Spec.Containers) > 0 {
		if dcpp.pod.Spec.Containers[0].Name != container.Name {
			return networkModeServiceRely + makeContainerServiceName(dcpp.pod.Name, dcpp.pod.Spec.Containers[0].Name)
		}
	}
	return ""
}

func (dcpp *dockerComposeProject) toServiceNetworks(isInit bool) map[string]*types.ServiceNetworkConfig {
	aliasNames := make([]string, 0)
	serviceName := ""
	if !isInit {
		serviceName = dcpp.pod.Labels[k8sappLabel]
		if serviceName == "" {
			serviceName = dcpp.pod.Labels[appLabel]
		}
	}
	if serviceName == "" {
		if len(dcpp.pod.OwnerReferences) > 0 {
			serviceName = dcpp.pod.OwnerReferences[0].Name
		}
	}
	if serviceName != "" {
		aliasNames = append(aliasNames, serviceName)
	}
	netfield, _ := makeNetworkName(dcpp.config.Project)
	if !dcpp.pod.Spec.HostNetwork {
		return map[string]*types.ServiceNetworkConfig{netfield: {Aliases: aliasNames}}
	}
	return nil
}

func (dcpp *dockerComposeProject) toPrivileged(container v1.Container) bool {
	if container.SecurityContext != nil {
		if container.SecurityContext.Privileged != nil {
			return *container.SecurityContext.Privileged
		}
	}
	return false
}

//pod里面的容器转换成docker-compose的service
func (dcpp *dockerComposeProject) toService(container v1.Container, isInit bool) types.ServiceConfig {
	svrconf := types.ServiceConfig{}
	podName := dcpp.pod.Name
	svrconf.Name = makeContainerServiceName(podName, container.Name)
	svrconf.Command = append(container.Command, container.Args...)
	svrconf.Image = container.Image
	svrconf.Labels = dcpp.newDockerComposeLabels(svrconf.Name, isInit)
	svrconf.CustomLabels = types.Labels{}
	svrconf.Environment = dcpp.toEnv(container)
	svrconf.HealthCheck = dcpp.toHealthCheck(container)
	svrconf.PullPolicy = types.PullPolicyIfNotPresent
	svrconf.Restart = types.RestartPolicyAlways //types.RestartPolicyOnFailure+ ":" + fmt.Sprint(restartTimes) //github.com/docker/compose/@v2.6.0/pkg/compose/create.go/getRestartPolicy
	svrconf.Scale = 1
	svrconf.Ports = dcpp.toPort(container)
	svrconf.Networks = dcpp.toServiceNetworks(isInit)
	svrconf.NetworkMode = dcpp.toNetworkMode(container)
	svrconf.Volumes = dcpp.toVolumes(container)
	svrconf.Privileged = dcpp.toPrivileged(container)
	svrconf.Tty = true
	return svrconf
}

//init-container依赖于上一个init-container的启动
//容器依赖于所有init-container的启动
func (dcpp *dockerComposeProject) services() types.Services {
	services := types.Services{}
	lastServiceName := ""
	initServiceNames := make([]string, len(dcpp.pod.Spec.InitContainers))
	for i, ic := range dcpp.pod.Spec.InitContainers {
		svrconf := dcpp.toService(ic, true)
		if i != 0 {
			svrconf.DependsOn = types.DependsOnConfig{
				lastServiceName: serviceCompeleteDependency,
			}
		}
		lastServiceName = svrconf.Name
		services = append(services, svrconf)
		initServiceNames[i] = svrconf.Name
	}
	for _, c := range dcpp.pod.Spec.Containers {
		svrconf := dcpp.toService(c, false)
		svrconf.DependsOn = types.DependsOnConfig{}
		for _, isn := range initServiceNames {
			svrconf.DependsOn[isn] = serviceCompeleteDependency
		}
		services = append(services, svrconf)
	}
	return services
}

func (dcpp *dockerComposeProject) Project() types.Project {
	project := types.Project{Name: dcpp.config.Project}
	project.Services = dcpp.services()
	networkField, networkName := makeNetworkName(project.Name)
	project.Networks = types.Networks{networkField: types.NetworkConfig{Name: networkName}}
	return project
}

//获取pod下的所有容器的service
func (dcpp *dockerComposeProject) ServiceNames() []types.ServiceConfig {
	var rets []types.ServiceConfig
	podName := dcpp.pod.Name
	for _, ic := range dcpp.pod.Spec.InitContainers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podName, ic.Name)})
	}
	for _, c := range dcpp.pod.Spec.Containers {
		rets = append(rets, types.ServiceConfig{Name: makeContainerServiceName(podName, c.Name)})
	}
	return rets
}
