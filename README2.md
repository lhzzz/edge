# Virtual-kubelet 

- 基于开源项目 [https://github.com/virtual-kubelet/virtual-kubelet](https://github.com/virtual-kubelet/virtual-kubelet) ： 大体功能就是通过实现kubelet的行为（例如接收Pod调度、与apiserver交互、提供restful接口等），并提供了一系列接口来让应用层实现，从而达到虚拟节点的作用，适用于边缘计算、IOT等场景(因为k8s对于边缘端来说过于厚重）
- 实现了基于自定义proto协议的接口，当云端部署了一个deploy时，会将pod调度给虚拟节点，然后虚拟节点会调用我们实现的接口，将Pod元数据传给我们的边缘机器，从而实现资源的部署
- 限制：云端要求为k8s集群，边缘端可自实现其容器编排的方式（只要实现了proto协议即可）

> 注：virtual-kubelet 在另一个仓库，不在本代码仓库中

### 主要实现的接口

```go
type PodLifecycleHandler interface {
	// CreatePod takes a Kubernetes Pod and deploys it within the provider.
	CreatePod(ctx context.Context, pod *corev1.Pod) error

	// UpdatePod takes a Kubernetes Pod and updates it within the provider.
	UpdatePod(ctx context.Context, pod *corev1.Pod) error

	// DeletePod takes a Kubernetes Pod and deletes it from the provider. Once a pod is deleted, the provider is
	// expected to call the NotifyPods callback with a terminal pod status where all the containers are in a terminal
	// state, as well as the pod. DeletePod may be called multiple times for the same pod.
	DeletePod(ctx context.Context, pod *corev1.Pod) error

	// GetPod retrieves a pod by name from the provider (can be cached).
	// The Pod returned is expected to be immutable, and may be accessed
	// concurrently outside of the calling goroutine. Therefore it is recommended
	// to return a version after DeepCopy.
	GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error)

	// GetPodStatus retrieves the status of a pod by name from the provider.
	// The PodStatus returned is expected to be immutable, and may be accessed
	// concurrently outside of the calling goroutine. Therefore it is recommended
	// to return a version after DeepCopy.
	GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error)

	// GetPods retrieves a list of all pods running on the provider (can be cached).
	// The Pods returned are expected to be immutable, and may be accessed
	// concurrently outside of the calling goroutine. Therefore it is recommended
	// to return a version after DeepCopy.
	GetPods(context.Context) ([]*corev1.Pod, error)
    
    //Fill some runtime message for a pod which will be created
	SetPodRuntimeInfo(ctx context.Context, pod *corev1.Pod) error
}

// PodNotifier is used as an extension to PodLifecycleHandler to support async updates of pod statuses.
type PodNotifier interface {
	// NotifyPods instructs the notifier to call the passed in function when
	// the pod status changes. It should be called when a pod's status changes.
	//
	// The provided pointer to a Pod is guaranteed to be used in a read-only
	// fashion. The provided pod's PodStatus should be up to date when
	// this function is called.
	//
	// NotifyPods must not block the caller since it is only used to register the callback.
	// The callback passed into `NotifyPods` may block when called.
	NotifyPods(context.Context, func(*corev1.Pod))
}
```

```go
// NodeProvider is the interface used for registering a node and updating its
// status in Kubernetes.
//
// Note: Implementers can choose to manage a node themselves, in which case
// it is not needed to provide an implementation for this interface.
type NodeProvider interface { // nolint:golint
	// Ping checks if the node is still active.
	// This is intended to be lightweight as it will be called periodically as a
	// heartbeat to keep the node marked as ready in Kubernetes.
	Ping(context.Context) error

	// NotifyNodeStatus is used to asynchronously monitor the node.
	// The passed in callback should be called any time there is a change to the
	// node's status.
	// This will generally trigger a call to the Kubernetes API server to update
	// the status.
	//
	// NotifyNodeStatus should not block callers.
	NotifyNodeStatus(ctx context.Context, cb func(*corev1.Node))
}
```

# edge-registry

- 部署在云端k8s集群，主要负责 virtual-kubelet的创建/销毁/查询 

# edgelet

- 边缘端的核心实现，即云边proto协议的实现服务
- 目前主要支持docker-compose的容器编排，即将云端的pod资源转换为docker compose中的service，支持的功能主要有：
  - 集群方面：
    - `Join`：加入到k8s集群，加入后容器由k8s master节点分配
    - `Reset`：退出k8s集群，容器维持原状运行
    - `Upgrade`：升级组件，例如edgelet/edgectl
    - `Version`：查看当前edgectl和edgelet的版本信息
  - 容器方面：
    - `CreatePod`：创建k8s pod到边缘端，对应到docker-compose中的service
    - `UpdatePod`：更新k8s pod到边缘端
    - `DeletePod`：删除k8s pod
    - `GetPod`：获取k8s pod
    - `GetPods`：批量获取k8s pod
    - `GetContainerLogs`： 获取k8s pod中容器的日志信息，不支持-f
    - `DescribeNodeStatus`： 获取边缘节点的状态信息，可传递节点的ip、内核信息、容器运行时版本、内存占用情况、cpu占用情况、指定目录的磁盘占用情况
    - `CreateVolume`：创建pod需要的挂载资源，例如configMap/secret
- 资源转换：
  - 命名 ：
    - 一个service对应一个pod中的一个容器，即多容器Pod对应多个service。因为docker-compose 中service命名不能重复，为了保证每一个pod容器的唯一性，命名以 `podname.containerName`方式命名
  - 容器网络：
    - 设置service的 `Networks`来注册容器网络，容器之间可以通过serviceName来实现访问，但由于上述说到是通过`podname.containerName`的方式命名service，与k8s的service命名不同，因此需要通过alias进行取别名，通过提取pod中label为`k8s-app`或`app`的value来别名service

```go
// ServiceNetworkConfig is the network configuration for a service
type ServiceNetworkConfig struct {
	Priority    int      `yaml:",omitempty" json:"priotirt,omitempty"`
	Aliases     []string `yaml:",omitempty" json:"aliases,omitempty"`
	Ipv4Address string   `mapstructure:"ipv4_address" yaml:"ipv4_address,omitempty" json:"ipv4_address,omitempty"`
	Ipv6Address string   `mapstructure:"ipv6_address" yaml:"ipv6_address,omitempty" json:"ipv6_address,omitempty"`

	Extensions map[string]interface{} `yaml:",inline" json:"-"`
}
```

   - 服务依赖： 
     - 容器依赖： 需要考虑到initContainer，Pod中initContainer按顺序执行后，才能运行container，因此docker-compose中需要进行service依赖，`service dependency`的建立：container依赖于最后一个initContainer的执行完成，最后一个initContainer依赖于前一个initContainer的执行
     - 网络依赖： 同一Pod下的多个容器之前网络是共享的，即可以通过访问127.0.0.1来访问其他container，通过设置docker-compose中service的`networkMode`来实现网络共享
   - 挂载资源
     - hostPath：直接进行边缘端的宿主机上的路径挂载 , Pod下发前会检查是否有这个路径，如果宿主机没有就会去创建，Pod下发后直接进行挂载即可
     - configMap/secret ：Pod下发前会检查是否有configMap和secret，如果有则会去获取configMap和secret中的数据，然后下发给边缘端，以文件的方式存储在指定的持久化路径，路径树

```shell
/data
  |--- /edgelet
         |--- /project
                 |--- /vol
                        |--- /configmap
                        |       |--- /namespace
                        |                |--- /user-config
                        |--- /secret
                                |--- /namespace
                                         |--- /user-secret
```

   - 环境变量
     - 用户自定义的环境变量，直接下发到容器即可
     - k8s中的runtime环境变量，例如：status.hostIP、status.PodIP之类的，通过使用node的IP来解决
     - 注：k8s自动添加的环境变量，例如其他服务的IP地址，也会下发到容器（因为无法与普通环境变量区分)
   - label
     - docker-compose通过label来过滤符合条件的容器，同时我们要满足将容器转换成Pod的需求，因此需要加上一些label来过滤：
       - `k8s-podName`：点查Pod时需要的标签，查询一个pod由哪些容器组成
       - `k8s-namespace`：过滤一个namespace下的pod
       - `k8s-podinfo`：由于还需要将docker-compose中的service转换成Pod返回给云端，因此Pod中有些数据对应到service是没有的，需要将其存储下来，后续好返回，例如namespace、ownerReference等，将这些数据序列化后放到容器的label中进行存储
     - 还有一些是docker-compose自带给容器的标签，挑几个重要的说明一下：
       - `com.docker.compose.project` 一个docker-compose.yaml生成的所有服务都为一个project，可以理解为一个集群就是一个project，命令行的docker-compose的project为其目录所在名，edgelet的project默认命名为`edge`
       - `com.docker.compose.service` docker-compose的service标签，值为创建时的service的name
- 状态转换：
  - 容器状态如何转换成Pod的状态
    - 考虑init-container的状态，是否执行结束，出错则报init:Error；正常则为complete
    - container的状态，restart三次，且还是报错的pod变为crashloopbackoff，低于三次为error
    - container全部都正常运行的为Running
- node信息：
  - node运行条件：
    - memory：内存占用情况，内存占用超过90%，将不能分配pod
    - disk：磁盘压力，磁盘占用超过80%，将不能分配pod，监控的磁盘默认路径为`/`
  - 心跳：目前每隔1分钟由virtual-kubelet 给edgelet发一次心跳，心跳返回当前node的状态，以及1分钟内有状态改变的pod（例如crash/running等）
- 持久化
  - edgelet的相关配置信息需要持久化到本地，以达到重启后配置不丢失，默认存储路径为`/data/edgelet/.conf/config.json`,目前配置的信息有：

```json
{
  "registryAddress":"10.0.0.122:31173", //注册到云端的服务地址，即edge-registry的地址
  "diskPath":"/",	//磁盘监测的目录
  "nodeName":"mw-123123124" //join后会将node-Name存储起来，后续reset会自动使用该nodeName
}
```

- 启动方式
  - docker容器方式启动，由于volume的需要，可能会在宿主机的任意位置进行创建目录，因此需要挂载根目录，另外还需要调用docker api接口，又需要将docker.sock进行挂载，同时对于升级操作来说，需要依靠另一个进程来控制，因此这种方式未采用
  - 使用`systemd`作为后台进程的方式启动，升级操作时通过替换可执行文件，然后`systemctl restart edgelet`即可实现升级

# edgectl

- 和edgelet进行交互的命令行工具，目前主要支持的命令有：

```shell
Usage:
  edgectl COMMAND [arg...] [flags]
  edgectl [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  join        edge join to cloud-cluster
  reset       Performs a best effort revert of changes made to this host by 'edgectl join'
  upgrade     upgrade a component on edge
  version     Show the version of the edge component

Flags:
      --edgelet-address string   connect edgelet to communicate cloud-cluster (default "10.0.0.122:10350")
  -h, --help                     help for edgectl

Additional help topics:
  edgectl            

Use "edgectl [command] --help" for more information about a command.
```

- 配置路径，edgectl的配置存储路径为`~/edgectl/conf`，edgectl会记录上一次连接的edgelet的address并存储，后续可以不用输入
- upgrade/version可以支持从云端对边缘端的edgelet进行升级/获取版本
- 升级命令详述

```shell
Usage:
  edgectl upgrade [flags]

Flags:
      --component string   Specify the component to upgrade
  -h, --help               help for upgrade
      --image string       Specify the image to upgrade component.
      --nodeName string    Specify the node name to upgrade the edgeNode component.

Global Flags:
      --edgelet-address string   connect edgelet to communicate cloud-cluster (default "10.0.0.122:10350")
```

   - component：组件命令，目前有edgelet/edgectl
   - image：升级的镜像
   - nodeName：如果需要升级边缘端的edgelet，需要填写nodeName