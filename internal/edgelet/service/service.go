package service

import (
	"bytes"
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/podmanager"
	"edge/internal/edgelet/podmanager/config"
	"edge/pkg/errdefs"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type edgelet struct {
	cloudAddress       string
	pm                 podmanager.PodManager
	lastHeartbeatTime  metav1.Time
	lastTransitionTime metav1.Time
	localIPAddress     string
	kernalVersion      string
	osiImage           string
}

const (
	GiB = 1024 * 1024 * 1024

	registryUrlFmt = "%s/edge/registry/node"
	logoutUrlFmt   = "%s/edge/registry/node?name=%s"
)

var (
	EOF = errors.New("EOF")
)

func NewEdgelet(cloudAddress string) *edgelet {
	localaddress, _ := getOutBoundIP()
	kernalversion, _ := host.KernelVersion()
	platform, _, _, _ := host.PlatformInformation()
	return &edgelet{
		cloudAddress:   cloudAddress,
		localIPAddress: localaddress,
		pm:             podmanager.New(config.WithProjectName("compose")),
		kernalVersion:  kernalversion,
		osiImage:       platform,
	}
}

func (e *edgelet) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	logrus.Info("Join Request:", req)
	url := fmt.Sprintf(registryUrlFmt, e.cloudAddress)
	reqbyte, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqbyte))
	if err != nil {
		logrus.Error("post failed,err=", err)
		return nil, err
	}
	defer resp.Body.Close()
	respbyte, _ := ioutil.ReadAll(resp.Body)
	logrus.Info("statusCode : ", resp.StatusCode, " respBody:", string(respbyte))
	respbody := pb.JoinResponse{}
	err = json.Unmarshal(respbyte, &respbody)
	if err != nil {
		logrus.Error("proto Unmarshal failed,err=", err)
		return nil, err
	}
	return &respbody, nil
}

func (e *edgelet) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	logrus.Info("Reset Request:", req)
	url := fmt.Sprintf(logoutUrlFmt, e.cloudAddress, req.NodeName)
	httpReq, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		logrus.Error("make delete request failed,err=", err)
		return nil, err
	}
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		logrus.Error("post failed,err=", err)
		return nil, err
	}
	defer httpResp.Body.Close()

	body, _ := ioutil.ReadAll(httpResp.Body)
	logrus.Info("statusCode : ", httpResp.StatusCode, " respBody:", string(body))
	resp := pb.ResetResponse{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		logrus.Error("json Unmarshal failed,err=", err)
		return nil, err
	}
	if resp.Error != nil {
		logrus.Error("response error,err=", fmt.Errorf(resp.Error.Msg))
		return nil, nil
	}
	return &resp, nil
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

func (e *edgelet) GetContainerLogs(req *pb.GetContainerLogsRequest, stream pb.Edgelet_GetContainerLogsServer) error {
	logrus.Info("GetContainerLogs :", req)

	ro, err := e.pm.GetContainerLogs(stream.Context(), req.Namespace, req.Name, req.ContainerName, req.Opts)
	if err != nil {
		return err
	}
	defer ro.Close()
	for {
		var data []byte
		n, er := ro.Read(data)
		if n > 0 {
			ew := stream.Send(&pb.GetContainerLogsResponse{Log: data})
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != EOF {
				err = er
			}
			break
		}
	}
	return nil
}

func (e *edgelet) RunInContainer(ctx context.Context, req *pb.RunInContainerRequest) (*pb.RunInContainerResponse, error) {
	return &pb.RunInContainerResponse{}, nil
}

func (e *edgelet) GetStatsSummary(ctx context.Context, req *pb.GetStatsSummaryRequest) (*pb.GetStatsSummaryResponse, error) {
	return &pb.GetStatsSummaryResponse{}, nil
}

func (e *edgelet) DescribeNodeStatus(ctx context.Context, req *pb.DescribeNodeStatusRequest) (*pb.DescribeNodeStatusResponse, error) {
	logrus.Info("DescribeNodeStatus")
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
				OSImage:                 e.osiImage,
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
		"pods":   resource.MustParse("30"),
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
		"pods":   resource.MustParse("30"), //TODO:这里要动态修改
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
			Type:    "InternalIP",
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
