package service

import (
	"edge/internal/constant"
	"edge/pkg/common"
	"edge/pkg/util"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type EdgeletConfig struct {
	CloudAddress string
	LogLevel     string
}

const (
	configPath    = constant.EdgeletCfgPath
	configName    = "config.json"
	defaultConfig = `{
		"logLevel": "INFO",
		"cloudAddress" : "center.zhst.com"
}`
)

func configReady(file string) error {
	if !util.IsFileExist(file) {
		err := os.MkdirAll(configPath, 0755)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		f.WriteString(defaultConfig)
	}
	return nil
}

func initConfig() (*EdgeletConfig, error) {
	configfile := filepath.Join(configPath, configName)
	configReady(configfile)
	viper.SetConfigFile(configfile)
	viper.SetConfigType("json")
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	ec := &EdgeletConfig{}
	ec.CloudAddress = viper.GetString("cloudAddress")
	ec.LogLevel = viper.GetString("logLevel")
	common.SetLogLevel(ec.LogLevel)
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		ec.CloudAddress = viper.GetString("cloudAddress")
		ec.LogLevel = viper.GetString("logLevel")
		common.SetLogLevel(ec.LogLevel)
	})
	return ec, nil
}

func (ec *EdgeletConfig) Save() error {
	if ec.CloudAddress != "" {
		viper.Set("cloudAddress", ec.CloudAddress)
	}
	if ec.LogLevel != "" {
		viper.Set("logLevel", ec.LogLevel)
	}
	return nil
}
