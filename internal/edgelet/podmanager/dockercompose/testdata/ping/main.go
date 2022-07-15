package main

import (
	"edge/pkg/util"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func main() {
	stopch := util.SetupSignalHandler()

	r := gin.Default()
	r.GET("/ping", healthCheck)

	go r.Run(os.Args[1])
	go func() {
		for {
			logrus.Info("send msg at timestamp:", time.Now())
			time.Sleep(5 * time.Second)
		}
	}()
	<-stopch
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
