package server

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/pkg/protoerr"
	"net"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func (e *EdgeRegistryServer) RunGrpc(address string) {
	grpcServer := grpc.NewServer()
	//健康检测
	health := health.NewServer()
	health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	pb.RegisterEdgeRegistryServiceServer(grpcServer, e)

	listen, err := net.Listen("tcp", address)
	if err != nil {
		logrus.Fatal("failed to listen: ", err)
	}
	logrus.Info("EdgeRegistry listen success:", address)
	go func() {
		if err := grpcServer.Serve(listen); err != nil {
			logrus.Fatal("failed to serve:", err)
		}
	}()

	<-e.stopCh
	logrus.Info("Received a program exit signal")
	grpcServer.Stop()
}

func (e *EdgeRegistryServer) CreateNode(ctx context.Context, req *pb.CreateNodeRequest) (*pb.CreateNodeResponse, error) {
	resp := &pb.CreateNodeResponse{}
	if req.NodeName == "" {
		resp.Error = protoerr.ParamErr("NodeName is empty")
		return resp, nil
	}
	exist, err := createEdgeNode(ctx, req.NodeName)
	if err != nil {
		logrus.Error("createEdgeNode failed,err=", err)
		resp.Error = protoerr.InternalErr(err)
		return resp, nil
	}
	resp.Exist = exist
	return resp, nil
}

func (e *EdgeRegistryServer) DeleteNode(ctx context.Context, req *pb.DeleteNodeRequest) (*pb.DeleteNodeResponse, error) {
	resp := &pb.DeleteNodeResponse{}
	if req.NodeName == "" {
		resp.Error = protoerr.ParamErr("NodeName is empty")
		return resp, nil
	}
	if err := deleteEdgeNode(ctx, req.NodeName); err != nil {
		logrus.Error("deleteEdgeNode failed,err=", err)
		resp.Error = protoerr.InternalErr(err)
		return resp, nil
	}
	return resp, nil
}

func (e *EdgeRegistryServer) GetNode(ctx context.Context, req *pb.GetNodeRequest) (*pb.GetNodeResponse, error) {
	resp := &pb.GetNodeResponse{}
	if req.NodeName == "" {
		resp.Error = protoerr.ParamErr("NodeName is empty")
		return resp, nil
	}
	exist := existEdgeNode(ctx, req.NodeName)
	if !exist {
		resp.Error = protoerr.NotFoundErr("node not exist")
	}
	return resp, nil
}
