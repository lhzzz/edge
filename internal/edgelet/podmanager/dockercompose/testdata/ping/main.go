package main

import (
	"edge/pkg/util"
	"flag"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	address = flag.String("address", "", "server listen address")
	conn    = flag.String("conngrpc", "", "grpc connect address")
)

func main() {
	stopch := util.SetupSignalHandler()
	flag.Parse()
	logrus.Infof("address:%v, conn:%v", *address, *conn)
	if *address != "" {
		r := gin.Default()
		r.GET("/ping", healthCheck)
		go r.Run(*address)
		go func() {
			for {
				logrus.Info("send msg at timestamp:", time.Now())
				time.Sleep(5 * time.Second)
			}
		}()
	}

	if *conn != "" {
		conn, err := grpc.Dial(*conn, grpc.WithInsecure())
		if err != nil {
			logrus.Errorf("grpc.Dial %s failed, err=%v", *conn, err)
			return
		}
		logrus.Info("status:", conn.GetState())
	}
	<-stopch
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
