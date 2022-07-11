package service

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/podmanager"
	"edge/internal/edgelet/podmanager/config"
	"edge/pkg/errdefs"
	"edge/pkg/protoerr"
	"edge/pkg/util"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type edgelet struct {
	config             *EdgeletConfig
	configMutex        sync.Mutex
	pm                 podmanager.PodManager
	lastHeartbeatTime  metav1.Time
	lastTransitionTime metav1.Time
	localIPAddress     string
	kernalVersion      string
	OSIImage           string
	buildVersion       string
}

const (
	GiB                           = 1024 * 1024 * 1024
	memPressureThreshold  float64 = 90
	diskPressureThreshold float64 = 80
)

func NewEdgelet(version string) *edgelet {
	localaddress, _ := util.GetOutBoundIP()
	kernalversion, _ := host.KernelVersion()
	platform, _, _, _ := host.PlatformInformation()
	conf, err := initConfig()
	if err != nil {
		log.Panicf("init config %s, err=%v", configPath, err)
	}
	log.Info("config load success:", conf)
	return &edgelet{
		kernalVersion:  kernalversion,
		OSIImage:       platform,
		localIPAddress: localaddress,
		pm:             podmanager.New(config.WithIPAddress(localaddress)),
		config:         conf,
		buildVersion:   version,
	}
}

func (e *edgelet) Stop() {
	e.config.Save()
}

func (e *edgelet) CreateVolume(ctx context.Context, req *pb.CreateVolumeRequest) (*pb.CreateVolumeResponse, error) {
	log.Info("CreateVolume")
	resp := &pb.CreateVolumeResponse{}
	err := e.pm.CreateVolume(ctx, req)
	if err != nil {
		log.Error("CreateVolume failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) CreatePod(ctx context.Context, req *pb.CreatePodRequest) (*pb.CreatePodResponse, error) {
	log := log.WithField("pod", req.Pod.Name)
	resp := &pb.CreatePodResponse{}
	pod, err := e.pm.CreatePod(ctx, req.Pod)
	if err != nil {
		log.Error("CreatePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	resp.Pod = pod
	log.Info("CreatePod phase:", resp.Pod.Status.Phase, " reason:", resp.Pod.Status.Reason)
	return resp, nil
}

func (e *edgelet) UpdatePod(ctx context.Context, req *pb.UpdatePodRequest) (*pb.UpdatePodResponse, error) {
	log := log.WithField("pod", req.Pod.Name)
	resp := &pb.UpdatePodResponse{}
	pod, err := e.pm.UpdatePod(ctx, req.Pod)
	if err != nil {
		log.Error("UpdatePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	resp.Pod = pod
	log.Info("UpdatePod phase:", resp.Pod.Status.Phase, " reason:", resp.Pod.Status.Reason)
	return resp, nil
}

func (e *edgelet) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DeletePodResponse, error) {
	log.Info("DeletePod podName:", req.Pod.ObjectMeta.Name)
	resp := &pb.DeletePodResponse{}
	err := e.pm.DeletePod(ctx, req.Pod)
	if err != nil {
		log.Error("DeletePod failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) GetPod(ctx context.Context, req *pb.GetPodRequest) (*pb.GetPodResponse, error) {
	resp := &pb.GetPodResponse{}
	log.Info("GetPod ", req)
	pod, err := e.pm.GetPod(ctx, req.Namespace, req.Name)
	if err != nil {
		log.Error("GetPod failed, err=", err)
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
	log.Info("GetPods :", req)
	pods, err := e.pm.GetPods(ctx)
	if err != nil {
		log.Error("GetPods failed, err=", err)
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
		return resp, nil
	}
	resp.Pods = pods
	return resp, nil
}

func (e *edgelet) GetContainerLogStream(req *pb.GetContainerLogsRequest, stream pb.Edgelet_GetContainerLogStreamServer) error {
	log.Info("GetContainerLogStream :", req)

	ctx := stream.Context()
	ro, err := e.pm.GetContainerLogs(ctx, req.Namespace, req.Name, req.ContainerName, req.Opts)
	if err != nil {
		return err
	}
	defer ro.Close()

	isquit := false
	for !isquit {
		select {
		case <-ctx.Done():
			isquit = true
			log.Info("GetContainerLogStream ctx is done:", ctx.Err())
		default:
			data := make([]byte, 1024)
			n, er := ro.Read(data)
			if n > 0 {
				log.Info("send msg :", string(data[:n]))
				ew := stream.SendMsg(&pb.GetContainerLogsResponse{Log: data[:n]})
				if ew != nil {
					isquit = true
					log.Info("GetContainerLogStream Exit for ew=", ew)
					break
				}
			}
			if er != nil {
				if er == io.EOF {
					err = stream.SendMsg(&pb.GetContainerLogsResponse{Error: protoerr.StreamFinishErr("EOF")})
					if err != nil {
						log.Warn("GetContainerLogStream send EOF failed err=", err)
					}
				}
				isquit = true
				log.Info("GetContainerLogStream Exit for er=", er)
				break
			}
		}
	}
	log.Info("GetContainerLogStream is exit")
	return nil
}

func (e *edgelet) RunInContainer(ctx context.Context, req *pb.RunInContainerRequest) (*pb.RunInContainerResponse, error) {
	return &pb.RunInContainerResponse{}, nil
}

func (e *edgelet) GetStatsSummary(ctx context.Context, req *pb.GetStatsSummaryRequest) (*pb.GetStatsSummaryResponse, error) {
	return &pb.GetStatsSummaryResponse{}, nil
}

func (e *edgelet) DescribeNodeStatus(ctx context.Context, req *pb.DescribeNodeStatusRequest) (*pb.DescribeNodeStatusResponse, error) {
	log.Info("DescribeNodeStatus")
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
				ContainerRuntimeVersion: e.pm.ContainerRuntimeVersion(context.Background()),
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

	ms, err := mem.VirtualMemory()
	if err != nil {
		log.Error("fetch mermory failed,err=", err)
	} else {
		memCondition := v1.NodeCondition{
			Type:               v1.NodeMemoryPressure,
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  e.lastHeartbeatTime,
			LastTransitionTime: e.lastTransitionTime,
			Reason:             "KubeletHasSufficientMemory",
			Message:            "kubelet has sufficient memory available",
		}
		if ms.UsedPercent > memPressureThreshold {
			memCondition.Status = v1.ConditionTrue
			memCondition.Reason = "KubeletHasInsufficientMemory"
			memCondition.Message = "kubelet has insufficient memory available"
		}
		nodeConditions = append(nodeConditions, memCondition)
	}

	dsk, err := disk.Usage(e.config.DiskPath)
	if err != nil {
		log.Error("fetch dsk failed ,err=", err)
	} else {
		diskCondition := v1.NodeCondition{
			Type:               v1.NodeDiskPressure,
			Status:             v1.ConditionFalse,
			LastHeartbeatTime:  e.lastHeartbeatTime,
			LastTransitionTime: e.lastTransitionTime,
			Reason:             "KubeletHasNoDiskPressure",
			Message:            "kubelet has no disk pressure",
		}
		if dsk.UsedPercent > diskPressureThreshold {
			diskCondition.Status = v1.ConditionTrue
			diskCondition.Reason = "KubeletHasDiskPressure"
			diskCondition.Message = "kubelet has disk pressure"
		}
		nodeConditions = append(nodeConditions, diskCondition)
	}
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
