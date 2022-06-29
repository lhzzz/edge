package service

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/pkg/protoerr"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func (e *edgelet) Join(ctx context.Context, req *pb.JoinRequest) (*pb.JoinResponse, error) {
	logrus.Info("Join Request:", req)
	resp := &pb.JoinResponse{}
	if req.CloudAddress == "" {
		resp.Error = protoerr.ParamErr("cloudaddress is empty")
		return resp, nil
	}
	e.addressMutex.Lock()
	e.cloudAddress = req.CloudAddress
	e.addressMutex.Unlock()

	// use restful
	/*
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
	*/
	conn, err := grpc.Dial(e.cloudAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("connect failed,cloudAddress:", e.cloudAddress, " err:", err)
		return nil, err
	}
	client := pb.NewEdgeRegistryServiceClient(conn)
	cnresp, err := client.CreateNode(context.Background(), &pb.CreateNodeRequest{NodeName: req.NodeName})
	if err != nil {
		return nil, err
	}
	resp.Error = cnresp.Error
	resp.Exist = cnresp.Exist
	return resp, nil
}

func (e *edgelet) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	logrus.Info("Reset Request:", req)
	resp := &pb.ResetResponse{}

	// use restful
	/*
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
	*/
	conn, err := grpc.Dial(e.cloudAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("connect failed,cloudAddress:", e.cloudAddress, " err:", err)
		return nil, err
	}
	client := pb.NewEdgeRegistryServiceClient(conn)
	delresp, err := client.DeleteNode(context.Background(), &pb.DeleteNodeRequest{NodeName: req.NodeName})
	if err != nil {
		return nil, err
	}
	resp.Error = delresp.Error
	return resp, nil
}
