package common

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// GetLogLevel aaa
func GetLogLevel(logLevelConfig string) log.Level {
	level := log.InfoLevel

	logLevelConfig = strings.ToUpper(logLevelConfig)

	switch logLevelConfig {
	case "DEBUG":
		level = log.DebugLevel
	case "INFO":
		level = log.InfoLevel
	case "ERROR":
		level = log.ErrorLevel
	case "FATAL":
		level = log.FatalLevel
	case "TRACE":
		level = log.TraceLevel
	case "WARN":
		level = log.WarnLevel
	}
	return level
}

// InitLogger 初始化日志模块
func InitLogger() {
	level := GetLogLevel(viper.GetString("LOG_LEVEL"))
	log.SetLevel(level)
	formatter := &log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		DisableQuote:    true,
		TimestampFormat: "2006-01-02 15:04:05",
	}
	log.SetFormatter(formatter)
	log.Debug("debug log level")
	log.Info("start")
}

func ModifyLogLevel() {

	router := gin.Default()

	router.PUT("/config", func(c *gin.Context) {

		oldlevel := GetLogLevel(viper.GetString("LOG_LEVEL"))
		log.Info("OldLevel", oldlevel)

		tm := 3600
		_time := c.Query("expire")
		_level := c.Query("logLevel")

		level := GetLogLevel(_level)
		log.SetLevel(level)
		log.Info("log level", level) //打印日志级别

		//显示log和viper中的日志级别
		c.JSON(200, gin.H{
			"req time":          _time,
			"req level":         level,
			"successed:":        true,
			"log level now at ": log.GetLevel(),
		})

		if _time != "" {
			tm, _ = strconv.Atoi(_time)
		}

		timer1 := time.NewTimer(time.Duration(tm) * time.Second)
		go func(t *time.Timer) {
			for {
				<-t.C
				t.Stop()
				break
			}
			//定时任务结束，还原日志级别
			log.SetLevel(oldlevel)

		}(timer1)
	})

	go router.Run(":1065")
}
