package service

import (
	"context"
	"edge/api/edge-proto/pb"
	"edge/internal/constant"
	"edge/pkg/protoerr"
	"edge/pkg/util"
	"fmt"

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

func (e *edgelet) Upgrade(ctx context.Context, req *pb.UpgradeRequest) (*pb.UpgradeResponse, error) {
	logrus.Info("Upgrade request:", req)
	resp := &pb.UpgradeResponse{}

	logrus.Infof("updating component %s ...", req.Component)

	if req.Component == pb.EdgeComponent_COMPONENT_EDGELET {
		pberr := e.upgradeEdgelet(ctx, req.Image, req.ShellCmds)
		if pberr != nil {
			resp.Error = pberr
		}
		return resp, nil
	}

	if req.Component == pb.EdgeComponent_COMPONENT_EDGECTL {
		pberr := e.upgradeEdgelet(ctx, req.Image, req.ShellCmds)
		if pberr != nil {
			resp.Error = pberr
		}
		return resp, nil
	}

	logrus.Info("Unknow component:", req.Component)

	err := util.RunLinuxCommands(true, req.ShellCmds...)
	if err != nil {
		resp.Error = protoerr.InternalErr(err)
	}
	return resp, nil
}

func (e *edgelet) upgradeEdgelet(ctx context.Context, image string, shellcmds []string) *pb.Error {
	if len(shellcmds) == 0 {
		if len(image) == 0 {
			return protoerr.ParamErr("image is empty")
		}
		dockerRunCmd := fmt.Sprintf(constant.DockerCopyEdgeletCmd, image)
		err := util.RunLinuxCommands(true, dockerRunCmd, constant.UpdageEdgeletCmd)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	} else {
		err := util.RunLinuxCommands(true, shellcmds...)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	}
	return nil
}

func (e *edgelet) upgradeEdgectl(ctx context.Context, image string, shellcmds []string) *pb.Error {
	if len(shellcmds) == 0 {
		if len(image) == 0 {
			return protoerr.ParamErr("image is empty")
		}
		dockerRunCmd := fmt.Sprintf(constant.DockerCopyEdgectlCmd, image)
		err := util.RunLinuxCommands(true, dockerRunCmd, constant.UpdateEdgectlCmd)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	} else {
		err := util.RunLinuxCommands(true, shellcmds...)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	}
	return nil
}
