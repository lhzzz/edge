package edgelet

import (
	"edge/api/edge-proto/pb"
	"edge/internal/edgelet/service"
	"edge/pkg/common"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func Run(cloudAddress, runAddress string) {

	common.InitLogger()

	grpcServer := grpc.NewServer()
	edgelet := service.NewEdgelet(cloudAddress)
	//健康检测
	health := health.NewServer()
	health.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, health)
	pb.RegisterEdgeletServer(grpcServer, edgelet)

	listen, err := net.Listen("tcp", runAddress)
	if err != nil {
		logrus.Fatal("failed to listen: ", err)
	}
	logrus.Info("edgelet listen success:", runAddress)
	go func() {
		if err := grpcServer.Serve(listen); err != nil {
			logrus.Fatal("failed to serve:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("Shutting down server...")
	grpcServer.Stop()
	logrus.Info("Server exiting")
}
