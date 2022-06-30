package service

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/podmanager"
	"edge/internal/edgelet/podmanager/config"
	"edge/pkg/errdefs"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"strings"
	"sync"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type edgelet struct {
	config             *EdgeletConfig
	addressMutex       sync.Mutex
	pm                 podmanager.PodManager
	lastHeartbeatTime  metav1.Time
	lastTransitionTime metav1.Time
	localIPAddress     string
	kernalVersion      string
	OSIImage           string
}

const (
	GiB = 1024 * 1024 * 1024
)

var (
	pberrEOF = &pb.Error{Code: pb.ErrorCode_SERVICE_STREAM_CALL_FINISH}
)

func NewEdgelet() *edgelet {
	localaddress, _ := getOutBoundIP()
	kernalversion, _ := host.KernelVersion()
	platform, _, _, _ := host.PlatformInformation()
	conf, err := initConfig()
	if err != nil {
		logrus.Panicf("init config %s, err=%v", configPath, err)
	}
	logrus.Info("config load success:", conf)
	return &edgelet{
		kernalVersion:  kernalversion,
		OSIImage:       platform,
		localIPAddress: localaddress,
		pm:             podmanager.New(config.WithIPAddress(localaddress)),
		config:         conf,
	}
}

func (e *edgelet) Stop() {
	e.config.Save()
}

func (e *edgelet) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	logrus.Info("CreateVolume")
	resp := &pb.CreateVolumeResponse{}
	err := e.pm.CreateVolume(ctx, req)
	if err != nil {
		logrus.Error("CreateVolume failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) CreatePod(ctx context.Context, req *pb.CreatePodRequest) (*pb.CreatePodResponse, error) {
	log := logrus.WithField("pod", req.Pod.Name)
	resp := &pb.CreatePodResponse{}
	pod, err := e.pm.CreatePod(ctx, req.Pod)
	if err != nil {
		logrus.Error("CreatePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	resp.Pod = pod
	log.Info("CreatePod phase:", resp.Pod.Status.Phase, " reason:", resp.Pod.Status.Reason)
	return resp, nil
}

func (e *edgelet) UpdatePod(ctx context.Context, req *pb.UpdatePodRequest) (*pb.UpdatePodResponse, error) {
	log := logrus.WithField("pod", req.Pod.Name)
	resp := &pb.UpdatePodResponse{}
	pod, err := e.pm.UpdatePod(ctx, req.Pod)
	if err != nil {
		logrus.Error("UpdatePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	resp.Pod = pod
	log.Info("UpdatePod phase:", resp.Pod.Status.Phase, " reason:", resp.Pod.Status.Reason)
	return resp, nil
}

func (e *edgelet) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DeletePodResponse, error) {
	logrus.Info("DeletePod podName:", req.Pod.ObjectMeta.Name)
	resp := &pb.DeletePodResponse{}
	err := e.pm.DeletePod(ctx, req.Pod)
	if err != nil {
		logrus.Error("DeletePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) GetPod(ctx context.Context, req *pb.GetPodRequest) (*pb.GetPodResponse, error) {
	resp := &pb.GetPodResponse{}
	logrus.Info("GetPod ", req)
	pod, err := e.pm.GetPod(ctx, req.Namespace, req.Name)
	if err != nil {
		logrus.Error("GetPod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		if errdefs.IsNotFound(err) {
			resp.Error.Code = pb.ErrorCode_NO_RESULT
		}
		return resp, nil
	}
	resp.Pod = pod
	return resp, nil
}

func (e *edgelet) GetPods(ctx context.Context, req *pb.GetPodsRequest) (*pb.GetPodsResponse, error) {
	resp := &pb.GetPodsResponse{}
	logrus.Info("GetPods :", req)
	pods, err := e.pm.GetPods(ctx)
	if err != nil {
		logrus.Error("GetPods failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		return resp, nil
	}
	resp.Pods = pods
	return resp, nil
}

// func (e *edgelet) GetContainerLogs(req *pb.GetContainerLogsRequest, stream pb.Edgelet_GetContainerLogsServer) error {
// 	logrus.Info("GetContainerLogs :", req)

// 	ctx := stream.Context()
// 	ro, err := e.pm.GetContainerLogs(ctx, req.Namespace, req.Name, req.ContainerName, req.Opts)
// 	if err != nil {
// 		return err
// 	}
// 	defer ro.Close()

// 	isquit := false
// 	for !isquit {
// 		select {
// 		case <-ctx.Done():
// 			isquit = true
// 			logrus.Info("GetContainerLogs ctx is done:", ctx.Err())
// 		default:
// 			data := make([]byte, 1024)
// 			n, er := ro.Read(data)
// 			if n > 0 {
// 				logrus.Info("send msg :", string(data[:n]))
// 				ew := stream.SendMsg(&pb.GetContainerLogsResponse{Log: data[:n]})
// 				if ew != nil {
// 					isquit = true
// 					logrus.Info("GetContainerLogs Exit for ew=", ew)
// 					break
// 				}
// 			}
// 			if er != nil {
// 				if er == io.EOF {
// 					err = stream.SendMsg(&pb.GetContainerLogsResponse{Error: pberrEOF})
// 					if err != nil {
// 						logrus.Warn("GetContainerLogs send EOF failed err=", err)
// 					}
// 				}
// 				isquit = true
// 				logrus.Info("GetContainerLogs Exit for er=", er)
// 				break
// 			}
// 		}
// 	}
// 	logrus.Info("GetContainerLogs is exit")
// 	return nil
// }

func (e *edgelet) GetContainerLogs(ctx context.Context, req *pb.GetContainerLogsRequest) (*pb.GetContainerLogsResponse, error) {
	logrus.Info("GetContainerLogs :", req)
	resp := &pb.GetContainerLogsResponse{}

	ro, err := e.pm.GetContainerLogs(ctx, req.Namespace, req.Name, req.ContainerName, req.Opts)
	if err != nil {
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		return resp, err
	}
	data, err := ioutil.ReadAll(ro)
	if err != nil {
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		return resp, err
	}
	resp.Log = data
	return resp, nil
}

func (e *edgelet) RunInContainer(ctx context.Context, req *pb.RunInContainerRequest) (*pb.RunInContainerResponse, error) {
	return &pb.RunInContainerResponse{}, nil
}

func (e *edgelet) GetStatsSummary(ctx context.Context, req *pb.GetStatsSummaryRequest) (*pb.GetStatsSummaryResponse, error) {
	return &pb.GetStatsSummaryResponse{}, nil
}

func (e *edgelet) DescribeNodeStatus(ctx context.Context, req *pb.DescribeNodeStatusRequest) (*pb.DescribeNodeStatusResponse, error) {
	logrus.Debug("DescribeNodeStatus")
	resp := &pb.DescribeNodeStatusResponse{}
	changePods, err := e.pm.DescribePodsStatus(ctx)
	if err != nil {
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		return resp, nil
	}
	resp.ChangePods = changePods
	resp.Node = e.configNode()
	e.lastHeartbeatTime = metav1.Now()
	e.lastTransitionTime = metav1.Now()
	return resp, nil
}

func (e *edgelet) configNode() *v1.Node {
	ms, _ := mem.VirtualMemory()
	node := &v1.Node{
		Status: v1.NodeStatus{
			Phase:       v1.NodeRunning,
			Capacity:    e.capacity(ms),
			Allocatable: e.allocatable(ms),
			Conditions:  e.nodeConditions(),
			Addresses:   e.nodeAddresses(),
			NodeInfo: v1.NodeSystemInfo{
				OperatingSystem:         e.operatingSystem(),
				Architecture:            e.architecture(),
				KernelVersion:           e.kernalVersion,
				OSImage:                 e.OSIImage,
				ContainerRuntimeVersion: "",
			},
		},
	}
	return node
}

// Capacity returns a resource list containing the capacity limits.
func (e *edgelet) capacity(minfo *mem.VirtualMemoryStat) v1.ResourceList {
	var total uint64 = 100
	if minfo != nil {
		total = minfo.Total / GiB
	}
	return v1.ResourceList{
		"cpu":    resource.MustParse("100"),
		"memory": resource.MustParse(fmt.Sprintf("%dGi", total)),
		"pods":   resource.MustParse("110"),
	}
}

func (e *edgelet) allocatable(minfo *mem.VirtualMemoryStat) v1.ResourceList {
	var usage uint64 = 0
	if minfo != nil {
		usage = minfo.Free / GiB
	}
	return v1.ResourceList{
		"cpu":    resource.MustParse("100"),
		"memory": resource.MustParse(fmt.Sprintf("%dGi", usage)),
		"pods":   resource.MustParse("110"), //TODO:这里要动态修改
	}
}

// NodeConditions returns a list of conditions (Ready, OutOfDisk, etc), for updates to the node status
// within Kubernetes.
func (e *edgelet) nodeConditions() []v1.NodeCondition {
	nodeConditions := []v1.NodeCondition{}
	//ready

	nodeConditions = append(nodeConditions, v1.NodeCondition{
		Type:               "Ready",
		Status:             v1.ConditionTrue,
		LastHeartbeatTime:  e.lastHeartbeatTime,
		LastTransitionTime: e.lastTransitionTime,
		Reason:             "EdgeletReady",
		Message:            "Edgelet is ready.",
	})

	//disk
	//memory
	//network

	// conds := []v1.NodeCondition{
	// 	{
	// 		Type:               "OutOfDisk",
	// 		Status:             v1.ConditionTrue,
	// 		LastHeartbeatTime:  e.lastHeartbeatTime,
	// 		LastTransitionTime: e.lastTransitionTime,
	// 		Reason:             "KubeletHasSufficientDisk",
	// 		Message:            "kubelet has sufficient disk space available",
	// 	},
	// 	{
	// 		Type:               "MemoryPressure",
	// 		Status:             v1.ConditionFalse,
	// 		LastHeartbeatTime:  e.lastHeartbeatTime,
	// 		LastTransitionTime: e.lastTransitionTime,
	// 		Reason:             "KubeletHasSufficientMemory",
	// 		Message:            "kubelet has sufficient memory available",
	// 	},
	// 	{
	// 		Type:               "DiskPressure",
	// 		Status:             v1.ConditionFalse,
	// 		LastHeartbeatTime:  e.lastHeartbeatTime,
	// 		LastTransitionTime: e.lastTransitionTime,
	// 		Reason:             "KubeletHasNoDiskPressure",
	// 		Message:            "kubelet has no disk pressure",
	// 	},
	// 	{
	// 		Type:               "NetworkUnavailable",
	// 		Status:             v1.ConditionFalse,
	// 		LastHeartbeatTime:  e.lastHeartbeatTime,
	// 		LastTransitionTime: e.lastTransitionTime,
	// 		Reason:             "RouteCreated",
	// 		Message:            "RouteController created a route",
	// 	},
	// }
	// nodeConditions = append(nodeConditions, conds...)
	return nodeConditions
}

// NodeAddresses returns a list of addresses for the node status
// within Kubernetes.
func (e *edgelet) nodeAddresses() []v1.NodeAddress {
	return []v1.NodeAddress{
		{
			Type:    v1.NodeExternalIP,
			Address: e.localIPAddress,
		},
	}
}

func (e *edgelet) operatingSystem() string {
	return runtime.GOOS
}

func (e *edgelet) architecture() string {
	return runtime.GOARCH
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
