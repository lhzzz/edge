package service

import (
	"bytes"
	"context"
	"edge/api/edge-proto/pb"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

var (
	paramError = &pb.Error{Code: pb.ErrorCode_PARAMETER_FAILED, Msg: "Param Error"}
)

func (e *edgelet) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	logrus.Info("Join Request:", req)
	if req.CloudAddress == "" {
		return &pb.JoinResponse{Error: paramError}, nil
	}
	e.addressMutex.Lock()
	e.cloudAddress = req.CloudAddress
	e.addressMutex.Unlock()
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
