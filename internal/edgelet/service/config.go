package service

import (
	"edge/internal/constant"
	"edge/pkg/util"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type EdgeletConfig struct {
	RegistryAddress string `json:"registryAddress"`
	DiskPath        string `json:"diskPath"`
	NodeName        string `json:"nodeName"`
}

const (
	configPath = constant.EdgeletCfgPath
	configName = "config.json"
)

var (
	defaultConfig = EdgeletConfig{
		RegistryAddress: constant.CenterDomain,
		DiskPath:        "/",
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
	if err := configReady(configfile); err != nil {
		return nil, err
	}

	ec := &EdgeletConfig{}
	f, err := os.OpenFile(configfile, os.O_RDONLY, 0755)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, ec)
	if err != nil {
		return nil, err
	}
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
