package service

import (
	"bytes"
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/podmanager"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

type edgelet struct {
	cloudAddress string
	vkUrl        string
	mutex        sync.RWMutex
	notify       chan struct{}
	pm           podmanager.PodManager
}

const (
	registryUrlFmt = "%s/edge/registry/node"
	logoutUrlFmt   = "%s/edge/registry/node?name=%s"
)

func NewEdgelet(cloudAddress string) *edgelet {
	return &edgelet{
		cloudAddress: cloudAddress,
		notify:       make(chan struct{}, 1),
		pm:           podmanager.NewPodManager(),
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
	isNeedNotify := false
	if len(respbody.VkUrl) > 0 {
		e.mutex.Lock()
		if respbody.VkUrl != e.vkUrl {
			e.vkUrl = respbody.VkUrl
			isNeedNotify = true
		}
		e.mutex.Unlock()
	}
	if isNeedNotify {
		e.notify <- struct{}{}
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
	resp := &pb.CreatePodResponse{}
	err := e.pm.CreatePod(ctx, req.Pod)
	if err != nil {
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) UpdatePod(ctx context.Context, req *pb.UpdatePodRequest) (*pb.UpdatePodResponse, error) {
	return nil, nil
}

func (e *edgelet) DeletePod(ctx context.Context, req *pb.DeletePodRequest) (*pb.DeletePodResponse, error) {
	resp := &pb.DeletePodResponse{}
	err := e.pm.DeletePod(ctx, req.Pod)
	if err != nil {
		resp.Error = &pb.Error{Code: pb.ErrorCode_INTERNAL_ERROR, Msg: err.Error()}
	}
	return resp, nil
}

func (e *edgelet) GetPod(ctx context.Context, req *pb.GetPodRequest) (*pb.GetPodResponse, error) {
	return nil, nil
}

func (e *edgelet) GetPods(ctx context.Context, req *pb.GetPodsRequest) (*pb.GetPodsResponse, error) {
	return nil, nil
}

func (e *edgelet) GetPodStatus(ctx context.Context, req *pb.GetPodStatusRequest) (*pb.GetPodStatusResponse, error) {
	return nil, nil
}

func (e *edgelet) GetContainerLogs(req *pb.GetContainerLogsRequest, stream pb.Edgelet_GetContainerLogsServer) error {
	return nil
}

func (e *edgelet) RunInContainer(ctx context.Context, req *pb.RunInContainerRequest) (*pb.RunInContainerResponse, error) {
	return nil, nil
}

func (e *edgelet) GetStatsSummary(ctx context.Context, req *pb.GetStatsSummaryRequest) (*pb.GetStatsSummaryResponse, error) {
	return nil, nil
}
