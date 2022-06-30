package common

import (
	"strings"

	log "github.com/sirupsen/logrus"
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
	log.SetLevel(log.InfoLevel)
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
