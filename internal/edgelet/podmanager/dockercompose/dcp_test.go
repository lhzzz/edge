package dockercompose

import (
	"context"
	"edge/internal/edgelet/podmanager/config"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	moby "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/sanathkr/go-yaml"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_remove(t *testing.T) {
	dcp := NewPodManager()
	// err := dcp.api.Remove(context.TODO(), "compose", api.RemoveOptions{
	// 	Services: []string{"ubuntu-bygolang"},
	// 	Force:    true,
	// 	//DryRun:   true,
	// })
	// t.Log("remove err:", err)
	dcp.composeApi.Down(context.TODO(), "compose", api.DownOptions{
		Project: &types.Project{
			Name: "compose",
			Services: types.Services{types.ServiceConfig{
				Name: "ubuntu-bygolang",
			}},
		},
	})
}

func Test_createAndRun(t *testing.T) {
	dcp := NewPodManager()
	label := types.Labels{}
	label.Add(api.ServiceLabel, "ubuntu-bygolang")
	label.Add(api.ProjectLabel, "compose")
	label.Add(api.OneoffLabel, "False")
	// label.Add(api.WorkingDirLabel, "/mnt/c/Users/LinHao/go/test/compose")
	// label.Add(api.ConfigFilesLabel, "docker-compose.yml")
	label2 := types.Labels{}
	label2.Add(api.ServiceLabel, "ubuntu-bygolang")
	label2.Add(api.ProjectLabel, "compose")
	label2.Add(api.OneoffLabel, "False")

	project := types.Project{
		Name: "compose",
		Services: types.Services{
			types.ServiceConfig{
				Name:          "ubuntu-bygolang",
				Command:       types.ShellCommand{"sleep", "10d"},
				Image:         "ubuntu:latest",
				ContainerName: "ubuntu-bygolang-1-grpc",
				CustomLabels:  types.Labels{},
				Labels:        label,
				Scale:         1,
				Restart:       "always",
			},
			types.ServiceConfig{
				Name:          "ubuntu-bygolang",
				Command:       types.ShellCommand{"sleep", "10d"},
				Image:         "ubuntu:latest",
				CustomLabels:  types.Labels{},
				ContainerName: "ubuntu-bygolang-2-proxy",
				Labels:        label2,
				Scale:         1,
				Restart:       "always",
			},
		},
	}

	err := dcp.composeApi.Up(context.TODO(), &project, api.UpOptions{
		Create: api.CreateOptions{
			Inherit:  true,
			Recreate: "force",
		},
		Start: api.StartOptions{Project: &project},
	})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log("up success")
}

func Test_ps(t *testing.T) {
	dcp := NewPodManager(config.WithProjectName("compose"))
	sum, err := dcp.composeApi.Ps(context.TODO(), "compose", api.PsOptions{All: true})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(sum)
}

func Test_convert(t *testing.T) {
	dcp := NewPodManager()

	label := types.Labels{}
	label.Add(api.ServiceLabel, "ubuntu-bygolang")
	label.Add(api.ProjectLabel, "compose")
	label.Add(api.OneoffLabel, "False")
	label.Add(api.WorkingDirLabel, "/mnt/c/Users/LinHao/go/test/compose")
	label.Add(api.ConfigFilesLabel, "docker-compose.yml")
	pro := &types.Project{
		Name: "compose",
		Services: types.Services{
			types.ServiceConfig{
				Name:         "ubuntu-bygolang",
				Command:      types.ShellCommand{"sleep", "10d"},
				Image:        "ubuntu:latest",
				CustomLabels: types.Labels{},
				Labels:       label,
				Scale:        1,
				Restart:      "always",
			},
		},
	}
	data, err := dcp.composeApi.Convert(context.Background(), pro, api.ConvertOptions{Format: "yaml"})
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
}

func Test_yaml(t *testing.T) {
	f, err := os.Open("./testdata/compose.yml")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()

	data, _ := ioutil.ReadAll(f)
	pj := types.Project{}
	err = yaml.Unmarshal(data, &pj)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(pj)
}

var pod = &v1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:            "nginx-deployment-2xclp",
		GenerateName:    "nginx-deployment-",
		Namespace:       "default",
		UID:             "d01f7541-c361-4394-a97a-d9514a4b9f16",
		ResourceVersion: "17777990",
		//CreationTimestamp: metav1.Time{time.Parse()},
		Labels: map[string]string{
			"k8s-app": "nginx",
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: "apps/v1",
				Kind:       "DaemonSet",
				Name:       "nginx-deployment",
			},
		},
		ManagedFields: []metav1.ManagedFieldsEntry{
			{
				Manager:    "kube-controller-manager",
				Operation:  metav1.ManagedFieldsOperationUpdate,
				APIVersion: "v1",
			},
			{
				Manager:    "virtual-kubelet",
				Operation:  metav1.ManagedFieldsOperationUpdate,
				APIVersion: "v1",
			},
		},
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:    "u1",
				Image:   "ubuntu:latest",
				Command: []string{"sleep", "10d"},
				Ports: []v1.ContainerPort{
					{
						ContainerPort: 80,
						Protocol:      v1.ProtocolTCP,
						HostPort:      8080,
					},
				},
				Env: []v1.EnvVar{
					{
						Name:  "KUBERNETES_SERVICE_PORT_HTTPS",
						Value: "443",
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "kube-api-access-55wsb",
						ReadOnly:  true,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					},
				},
				ImagePullPolicy: v1.PullIfNotPresent,
			},
			{
				Name:    "u2",
				Image:   "ubuntu:latest",
				Command: []string{"sleep", "10d"},
				Ports: []v1.ContainerPort{
					{
						ContainerPort: 443,
						Protocol:      v1.ProtocolTCP,
						HostPort:      8443,
					},
				},
				Env: []v1.EnvVar{
					{
						Name:  "KUBERNETES_SERVICE_PORT_HTTPS",
						Value: "443",
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      "kube-api-access-55wsb",
						ReadOnly:  true,
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					},
				},
				ImagePullPolicy: v1.PullIfNotPresent,
			},
		},
		RestartPolicy:      v1.RestartPolicyAlways,
		DNSPolicy:          v1.DNSClusterFirst,
		NodeSelector:       map[string]string{"type": "virtual-kubelet"},
		ServiceAccountName: "default",
		NodeName:           "vk1",
	},
	Status: v1.PodStatus{
		Phase:  v1.PodRunning,
		HostIP: "1.2.3.4",
		PodIP:  "5.6.7.8",
	},
}

func Test_createPod(t *testing.T) {
	dcp := NewPodManager(config.WithProjectName("compose"))

	_, err := dcp.CreatePod(context.Background(), pod)
	if err != nil {
		t.Error(err)
	}
}

func Test_deletePod(t *testing.T) {
	dcp := NewPodManager()

	err := dcp.DeletePod(context.Background(), pod)
	if err != nil {
		t.Error(err)
	}
}

func Test_listPod(t *testing.T) {
	dir := NewPodManager(config.WithProjectName("compose"))
	pods, err := dir.GetPod(context.Background(), "default", "nginx-deployment-2xclp")

	t.Log(pods, err)
}

func getOutBoundIP() (ip string, err error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		fmt.Println(err)
		return
	}
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	ip = strings.Split(localAddr.String(), ":")[0]
	return
}

func Test_getLocalIPAddress(t *testing.T) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println(ipnet.IP.String())
			}
		}
	}
}

func Test_getip(t *testing.T) {
	ip, _ := getOutBoundIP()
	t.Log(ip)
}

type LogConsumer struct {
	containerName string
}

func NewLogConsumer() *LogConsumer {
	return &LogConsumer{}
}

func (lg *LogConsumer) Log(service, container, message string) {
	logrus.Infof("%s/%s:%s", service, container, message)
}

func (lg *LogConsumer) Status(container, msg string) {
	logrus.Infof("%s:%s", container, msg)
}

func (lg *LogConsumer) Register(container string) {
	lg.containerName = container
}

func Test_log(t *testing.T) {
	dcp := NewPodManager(config.WithProjectName("edge"))
	ctx := context.Background()

	// err := dcp.composeApi.Logs(ctx, dcp.project, NewLogConsumer(), api.LogOptions{
	// 	Services:   []string{"iperf-exporter-8vc4m.iperf3"},
	// 	Tail:       "1000",
	// 	Timestamps: false,
	// })
	// if err != nil {
	// 	t.Log(err)
	// 	return
	// }
	// return

	podname := "iperf-exporter-8vc4m"
	containerName := "exporter"
	f := getDefaultFilters(dcp.Project)
	f = append(f, serviceFilter(makeContainerServiceName(podname, containerName)))
	mcs, err := dcp.dockerCli.Client().ContainerList(ctx, moby.ContainerListOptions{
		Filters: filters.NewArgs(f...),
		All:     true,
	})
	if err != nil {
		return
	}
	if len(mcs) == 0 {
		return
	}
	ro, err := dcp.dockerCli.Client().ContainerLogs(ctx, mcs[0].ID, moby.ContainerLogsOptions{
		//Since:      "2022-06-20 06:00:00",
		Timestamps: false,
		Follow:     false,
		//Tail:       "1000",
		ShowStdout: true,
		Details:    true,
		ShowStderr: true,
	})
	if err != nil {
		t.Log(err)
		return
	}
	defer ro.Close()

	// data := make([]byte, 1024)
	// for {
	// 	n, err := ro.Read(data)
	// 	if n == 0 {
	// 		break
	// 	}
	// 	logrus.Infof("Read %d bytes", n)
	// 	logrus.Info(string(data[:n]))
	// 	if err != nil {
	// 		t.Log(err)
	// 		break
	// 	}
	// }
	data, err := ioutil.ReadAll(ro)
	t.Log(string(data))
}

func Test_load(t *testing.T) {
	path := "/mnt/c/Users/LinHao/go/test/hummingbird/"
	opts, err := cli.NewProjectOptions([]string{path + "docker-compose.yaml"}, cli.WithName("hummingbird"))
	if err != nil {
		t.Error(err)
		return
	}

	project, err := cli.ProjectFromOptions(opts)
	if err != nil {
		t.Error(err)
		return
	}

	for _, s := range project.Services {
		if s.Name == "vearch-master" {
			t.Log(s)
		}
	}

	// data, _ := json.MarshalIndent(project, "", "\t")
	// t.Log(string(data))

	// ym, _ := yaml.Marshal(project)
	// t.Log(string(ym))

	// f, err := os.OpenFile(path+"rewrite.yaml", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 666)
	// if err != nil {
	// 	t.Error(err)
	// 	return
	// }
	// defer f.Close()

	// n, err := f.Write(ym)
	// t.Log(n, err)
}

func Test_readlog(t *testing.T) {
	file := "/var/lib/docker/containers/2c7e3fe780e9c9a3078fdf0b0a8d16ab99dff97134cd21a97fd8c26d4aa95ed3/2c7e3fe780e9c9a3078fdf0b0a8d16ab99dff97134cd21a97fd8c26d4aa95ed3-json.log"
	f, err := os.OpenFile(file, os.O_RDONLY, 0666)
	if err != nil {
		t.Log(err)
		return
	}
	defer f.Close()

}
