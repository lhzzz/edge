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
	if req.CloudAddress == "" && e.config.CloudAddress == "" {
		resp.Error = protoerr.ParamErr("cloudaddress is empty")
		return resp, nil
	}
	e.addressMutex.Lock()
	if req.CloudAddress != "" {
		e.config.CloudAddress = req.CloudAddress
	}
	e.addressMutex.Unlock()
	conn, err := grpc.Dial(e.config.CloudAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("grpc.Dial %s failed, err=%v:", e.config.CloudAddress, err)
		return nil, err
	}
	client := pb.NewEdgeRegistryServiceClient(conn)
	cnresp, err := client.CreateNode(context.Background(), &pb.CreateNodeRequest{NodeName: req.NodeName})
	if err != nil {
		logrus.Error("CreateNode in Join failed,err=", err)
		return nil, err
	}
	resp.Error = cnresp.Error
	resp.Exist = cnresp.Exist
	return resp, nil
}

func (e *edgelet) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	logrus.Info("Reset Request:", req)
	resp := &pb.ResetResponse{}

	conn, err := grpc.Dial(e.config.CloudAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("connect failed,cloudAddress:", e.config.CloudAddress, " err:", err)
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
