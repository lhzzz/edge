package service

import (
	"bytes"
	"context"
	"edge/api/pb"
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
}

const (
	registryUrlFmt = "%s/edge/registry/node"
)

func NewEdgelet(cloudAddress string) *edgelet {
	return &edgelet{
		cloudAddress: cloudAddress,
		notify:       make(chan struct{}, 1),
	}
}

func (e *edgelet) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	logrus.Info("Join Request:", req)
	url := fmt.Sprintf(registryUrlFmt, e.cloudAddress)
	reqbyte, err := json.Marshal(req)
	if err != nil {
		logrus.Error("proto marshal failed,err=", err)
		return nil, err
	}
	resp, err := http.Post(url, "application/x-protobuf", bytes.NewBuffer(reqbyte))
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

	return nil, nil
}
