package service

import (
	"edge/internal/constant"
	"edge/pkg/util"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type EdgeletConfig struct {
	CloudAddress string `json:"cloudAddress"`
	DiskPath     string `json:"diskPath"`
}

const (
	configPath = constant.EdgeletCfgPath
	configName = "config.json"
)

var (
	defaultConfig = EdgeletConfig{
		CloudAddress: constant.CenterDomain,
		DiskPath:     "/",
	}
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
		data, err := json.Marshal(defaultConfig)
		if err != nil {
			return err
		}
		f.Write(data)
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
	err = viper.Unmarshal(ec)
	if err != nil {
		return nil, err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		viper.Unmarshal(ec)
	})
	return ec, nil
}

func (ec *EdgeletConfig) Save() error {
	f, err := os.OpenFile(filepath.Join(configPath, configName), os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(ec)
	if err != nil {
		return err
	}
	f.Write(data)
	f.Sync()
	return nil
}
