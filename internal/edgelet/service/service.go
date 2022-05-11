package service

import (
	"bytes"
	"context"
	"edge/internal/edgelet/pb"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

type edgelet struct {
	cloudAddress string
	vkUrl        string
	mutex        sync.RWMutex
	notify       chan struct{}
}

func NewEdgelet(cloudAddress string) *edgelet {
	return &edgelet{
		cloudAddress: cloudAddress,
		notify:       make(chan struct{}, 1),
	}
}

func (e *edgelet) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	url := fmt.Sprintf("%s/edgenode", e.cloudAddress)
	reqbyte, err := proto.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(reqbyte))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respbyte, _ := ioutil.ReadAll(resp.Body)
	respbody := pb.JoinResponse{}
	err = proto.Unmarshal(respbyte, &respbody)
	if err != nil {
		return nil, err
	}
	logrus.Info(&respbody)
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
	return nil, nil
}

func (e *edgelet) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {

	return nil, nil
}
