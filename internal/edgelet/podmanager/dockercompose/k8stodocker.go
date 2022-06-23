package dockercompose

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
)

type dockerComposeProject struct {
	pod         *v1.Pod
	project     string
	projectPath string
}

type DockerComposeProject interface {
	Project() types.Project
	ServiceNames() []types.ServiceConfig
}

func NewPodProject(projectName, projectPath string, pod *v1.Pod) DockerComposeProject {
	return &dockerComposeProject{
		pod:         pod,
		project:     projectName,
		projectPath: projectPath,
	}
}

func (dcpp *dockerComposeProject) newDockerComposeLabels(service string, isInit bool) types.Labels {
	labels := types.Labels{}
	labels.Add(api.ProjectLabel, dcpp.project)
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

//TODO:volumn 的sourcePath转换处理,configMap/secrets
func (dcpp *dockerComposeProject) genSourcePath(mountVolumeName string) string {
	var index int = 0
	for i, vo := range dcpp.pod.Spec.Volumes {
		if vo.Name == mountVolumeName {
			index = i
			break
		}
	}
	vo := dcpp.pod.Spec.Volumes[index]
	if vo.HostPath != nil {
		return vo.HostPath.Path
	}
	if vo.ConfigMap != nil {
		logrus.Warn("Not support ConfigMap")
		return ""
	}
	if vo.Secret != nil {
		logrus.Warn("Not support Secret")
		return ""
	}
	//now not support configmap/secrets
	return ""
}

//TODO:volumn 的转换处理,configMap/secrets
func (dcpp *dockerComposeProject) volumes() types.Volumes {
	volumns := types.Volumes{}
	for _, v := range dcpp.pod.Spec.Volumes {
		volumns[v.Name] = types.VolumeConfig{
			Name: v.Name,
		}
	}
	return volumns
}

//TODO:健康检测的转换处理
func (dcpp *dockerComposeProject) toHealthCheck(container v1.Container) *types.HealthCheckConfig {

	return nil
}

//status-host ip / podip 这些env的处理
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
		source := dcpp.genSourcePath(v.Name)
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

//TODO:port转换有问题
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
	logrus.Info("ports:", ports)
	return ports
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
	svrconf.Restart = types.RestartPolicyOnFailure + ":" + fmt.Sprint(restartTimes) //github.com/docker/compose/@v2.6.0/pkg/compose/create.go/getRestartPolicy
	svrconf.Scale = 1
	svrconf.Ports = dcpp.toPort(container)
	//svrconf.Networks = map[string]*types.ServiceNetworkConfig{dcpp.project: nil}
	svrconf.Volumes = dcpp.toVolumes(container)
	return svrconf
}

//init-container依赖于上一个init-container的启动
//容器依赖于所有init-container的启动
func (dcpp *dockerComposeProject) services() types.Services {
	services := types.Services{}
	lastServiceName := ""
	initServiceNames := []string{}
	for i, ic := range dcpp.pod.Spec.InitContainers {
		svrconf := dcpp.toService(ic, true)
		if i != 0 {
			svrconf.DependsOn = types.DependsOnConfig{
				lastServiceName: serviceHealthDependency,
			}
		}
		lastServiceName = svrconf.Name
		services = append(services, svrconf)
		initServiceNames = append(initServiceNames, svrconf.Name)
	}
	for _, c := range dcpp.pod.Spec.Containers {
		svrconf := dcpp.toService(c, false)
		svrconf.DependsOn = types.DependsOnConfig{}
		for _, isn := range initServiceNames {
			svrconf.DependsOn[isn] = serviceHealthDependency
		}
		services = append(services, svrconf)
	}
	return services
}

func (dcpp *dockerComposeProject) networks() types.Networks {
	networks := types.Networks{}
	networks[dcpp.project] = types.NetworkConfig{
		Name: dcpp.project,
	}
	return networks
}

func (dcpp *dockerComposeProject) configs() types.Configs {
	return types.Configs{}
}

func (dcpp *dockerComposeProject) Project() types.Project {
	project := types.Project{Name: dcpp.project}
	project.Services = dcpp.services()
	//project.Volumes = dcpp.volumes()
	//project.Networks = dcpp.networks()
	//project.Configs = dcpp.configs()
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
