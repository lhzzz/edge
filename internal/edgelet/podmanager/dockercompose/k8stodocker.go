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
	pod     *v1.Pod
	project string
}

type DockerComposeProject interface {
	Project() types.Project
	ListServices() []types.ServiceConfig
}

func NewPodProject(projectName string, pod *v1.Pod) DockerComposeProject {
	return &dockerComposeProject{
		pod:     pod,
		project: projectName,
	}
}

func makeContainerServiceName(podName, containerName string) string {
	return podName + "." + containerName
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

func (dcpp *dockerComposeProject) volumes() types.Volumes {
	volumns := types.Volumes{}
	for _, v := range dcpp.pod.Spec.Volumes {
		volumns[v.Name] = types.VolumeConfig{
			Name: v.Name,
			//TODO:? volume的转换
		}
	}
	return volumns
}

//TODO:健康检测的转换处理
func (dcpp *dockerComposeProject) toHealthCheck(container v1.Container) *types.HealthCheckConfig {

	return nil
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
	svrconf.Environment = types.MappingWithEquals{}
	for _, e := range container.Env {
		env := e
		svrconf.Environment[env.Name] = &env.Value
	}

	svrconf.HealthCheck = dcpp.toHealthCheck(container)
	svrconf.PullPolicy = types.PullPolicyIfNotPresent
	svrconf.Restart = types.RestartPolicyOnFailure + ":" + fmt.Sprint(restartTimes) //github.com/docker/compose/@v2.6.0/pkg/compose/create.go/getRestartPolicy
	svrconf.Scale = 1
	svrconf.Ports = []types.ServicePortConfig{}

	//TODO:port转换有问题
	for _, p := range container.Ports {
		svrconf.Ports = append(svrconf.Ports, types.ServicePortConfig{
			Protocol:  strings.ToLower(string(p.Protocol)),
			Published: fmt.Sprint(p.HostPort),
			Target:    uint32(p.ContainerPort),
		})
	}
	logrus.Info("ports:", svrconf.Ports)
	//TODO:volumn 的转换处理， configMap/secrets
	svrconf.Volumes = []types.ServiceVolumeConfig{}
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
	return types.Networks{}
}

func (dcpp *dockerComposeProject) configs() types.Configs {
	return types.Configs{}
}

func (dcpp *dockerComposeProject) Project() types.Project {
	project := types.Project{Name: dcpp.project}
	project.Services = dcpp.services()
	project.Volumes = dcpp.volumes()
	project.Networks = dcpp.networks()
	project.Configs = dcpp.configs()
	return project
}

//获取pod下的所有容器的service
func (dcpp *dockerComposeProject) ListServices() []types.ServiceConfig {
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
