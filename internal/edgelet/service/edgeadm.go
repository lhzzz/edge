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
	if req.CloudAddress == "" && e.config.RegistryAddress == "" {
		resp.Error = protoerr.ParamErr("cloudaddress is empty")
		return resp, nil
	}

	if req.NodeName == "" {
		resp.Error = protoerr.ParamErr("nodeName is empty")
		return resp, nil
	}

	e.configMutex.Lock()
	if req.CloudAddress != "" {
		e.config.RegistryAddress = req.CloudAddress
	}
	e.configMutex.Unlock()
	conn, err := grpc.Dial(e.config.RegistryAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Errorf("grpc.Dial %s failed, err=%v:", e.config.RegistryAddress, err)
		return nil, err
	}
	client := pb.NewEdgeRegistryServiceClient(conn)
	if e.config.NodeName != "" && e.config.NodeName != req.NodeName {
		getrsp, err := client.GetNode(ctx, &pb.GetNodeRequest{NodeName: e.config.NodeName})
		if err != nil {
			return resp, err
		}
		if getrsp.Error != nil {
			if !protoerr.IsNotFoundErr(getrsp.Error) {
				resp.Error = getrsp.Error
				return resp, nil
			}
		} else {
			msg := fmt.Sprintf("The last nodeName %s has already exist in cluster, Please reset it before new join.", e.config.NodeName)
			resp.Error = protoerr.ParamErr(msg)
			return resp, nil
		}
	}

	cnresp, err := client.CreateNode(ctx, &pb.CreateNodeRequest{NodeName: req.NodeName})
	if err != nil {
		logrus.Error("CreateNode in Join failed,err=", err)
		return nil, err
	}
	resp.Error = cnresp.Error
	resp.Exist = cnresp.Exist
	e.configMutex.Lock()
	e.config.NodeName = req.NodeName
	e.config.Save()
	e.configMutex.Unlock()
	return resp, nil
}

func (e *edgelet) Reset(ctx context.Context, req *pb.ResetRequest) (*pb.ResetResponse, error) {
	logrus.Info("Reset Request:", req)
	resp := &pb.ResetResponse{}

	if e.config.NodeName == "" {
		resp.Error = protoerr.ParamErr("should to join before reset")
	}

	conn, err := grpc.Dial(e.config.RegistryAddress, grpc.WithInsecure()) //grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Error("connect failed,cloudAddress:", e.config.RegistryAddress, " err:", err)
		return nil, err
	}
	client := pb.NewEdgeRegistryServiceClient(conn)
	delresp, err := client.DeleteNode(ctx, &pb.DeleteNodeRequest{NodeName: e.config.NodeName})
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
	if req.Component == pb.EdgeComponent_UNKNOW {
		resp.Error = protoerr.ParamErr("Unknow Componet to upgrade")
		return resp, nil
	}

	if req.Component == pb.EdgeComponent_EDGELET {
		pberr := e.upgradeEdgelet(ctx, req.Image, req.ShellCmds)
		if pberr != nil {
			resp.Error = pberr
		}
		return resp, nil
	}

	if req.Component == pb.EdgeComponent_EDGECTL {
		pberr := e.upgradeEdgelet(ctx, req.Image, req.ShellCmds)
		if pberr != nil {
			resp.Error = pberr
		}
		return resp, nil
	}

	if req.Component == pb.EdgeComponent_CUSTOMIZE {
		err := util.RunLinuxCommands(false, req.ShellCmds...)
		if err != nil {
			resp.Error = protoerr.InternalErr(err)
		}
		return resp, nil
	}

	return resp, nil
}

func (e *edgelet) ListVersion(ctx context.Context, req *pb.ListVersionRequest) (*pb.ListVersionResponse, error) {
	logrus.Info("ListVersion request:", req)
	resp := &pb.ListVersionResponse{}
	resp.EdgeletVersion = e.buildVersion
	return resp, nil
}

func (e *edgelet) upgradeEdgelet(ctx context.Context, image string, shellcmds []string) *pb.Error {
	if len(shellcmds) == 0 {
		if len(image) == 0 {
			return protoerr.ParamErr("image is empty")
		}
		dockerRunCmd := fmt.Sprintf(constant.DockerCopyEdgeletCmd, image)
		err := util.RunLinuxCommands(false, dockerRunCmd, constant.UpdageEdgeletCmd)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	} else {
		err := util.RunLinuxCommands(false, shellcmds...)
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
		err := util.RunLinuxCommands(false, dockerRunCmd, constant.UpdateEdgectlCmd)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	} else {
		err := util.RunLinuxCommands(false, shellcmds...)
		if err != nil {
			return protoerr.InternalErr(err)
		}
	}
	return nil
}
