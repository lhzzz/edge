package main

import (
	"edge/pkg/util"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	stopch := util.SetupSignalHandler()

	r := gin.Default()
	r.GET("/ping", healthCheck)

	go r.Run(os.Args[1])
	<-stopch
}

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, "pong")
}
