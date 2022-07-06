package cmd

import (
	"edge/internal/constant"
	"edge/pkg/util"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	configDir  = filepath.Join(util.HomeDir(), ".edgectl/conf")
	configfile = filepath.Join(configDir, "config.json")
)

type EdgeCtlConfig struct {
	EdgeletAddress string
}

func NewEdgeCtlConfig() (*EdgeCtlConfig, error) {
	conf := EdgeCtlConfig{
		EdgeletAddress: constant.EdgeletDefaultAddress,
	}
	if err := conf.configReady(); err != nil {
		return nil, err
	}
	return &conf, nil
}

func (ecc *EdgeCtlConfig) configReady() error {
	if !util.IsFileExist(configfile) {
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			return err
		}
		f, err := os.OpenFile(configfile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := json.Marshal(ecc)
		if err != nil {
			return err
		}
		f.Write(data)
	} else {
		f, err := os.OpenFile(configfile, os.O_RDONLY, 0755)
		if err != nil {
			return err
		}
		defer f.Close()
		data, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		err = json.Unmarshal(data, ecc)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ecc *EdgeCtlConfig) Save() error {
	f, err := os.OpenFile(configfile, os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := json.Marshal(ecc)
	if err != nil {
		return err
	}
	f.Write(data)
	f.Sync()
	return nil
}
