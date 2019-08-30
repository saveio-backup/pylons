package test_config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
	"io/ioutil"
	"os"
)

type Node struct {
	NodeBase58Addr string `json:"NodeBase58Addr"`
	NodeListenAddr string `json:"NodeListenAddr"`
}

type BaseConf struct {
	NetworkId           uint32   `json:"NetworkId"`
	NATProxyServerAddrs string   `json:"NATProxyServerAddrs"`
	ChainNodeURLs       []string `json:"ChainNodeURLs"`
	ChainClientType     string   `json:"ChainClientType"`
	Nodes               []Node   `json:"Nodes"`
}

type Config struct {
	BaseConfig *BaseConf `json:"BaseConfig"`
}

// FileExisted checks whether filename exists in filesystem
func FileExisted(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func GetJsonObjectFromFile(filePath string, jsonObject interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Error("[GetJsonObjectFromFile] ReadFile error: ", err.Error())
		return err
	}
	// Remove the UTF-8 Byte Order Mark
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	err = json.Unmarshal(data, jsonObject)
	if err != nil {
		return fmt.Errorf("json.Unmarshal %s error:%s", data, err)
	}
	return nil
}

func InitConfig() *Config {
	configDir := "./config.json"
	existed := FileExisted(configDir)
	if !existed {
		log.Info("[GetTestConf]: config.json is not existed")
		panic("[GetTestConf]: config.json is not existed")
	}
	cfg := &Config{}
	err := GetJsonObjectFromFile(configDir, cfg)
	if err != nil {
		log.Error("[GetJsonObjectFromFile] error: %s", err.Error())
		panic("[GetJsonObjectFromFile] error")
	}
	return cfg
}

func GetHostAddrCallBack(walletAddr common.Address) (string, error) {
	if addr, exist := NodeMap[walletAddr]; exist {
		return addr, nil
	} else {
		log.Errorf("[GetHostAddrCallBack] Host: %s is not exist.", common.ToBase58(walletAddr))
		return "", fmt.Errorf("[GetHostAddrCallBack] Host: %s is not exist.", common.ToBase58(walletAddr))
	}
}

var Parameters *Config
var NodeMap map[common.Address]string

func init() {
	Parameters = InitConfig()
	NodeMap = make(map[common.Address]string)

	for _, v := range Parameters.BaseConfig.Nodes {
		walletAddrTmp, _ := common.FromBase58(v.NodeBase58Addr)
		walletAddr := common.Address(walletAddrTmp)
		NodeMap[walletAddr] = v.NodeListenAddr
	}
	log.Infof("Config Init NetworkId: %d", Parameters.BaseConfig.NetworkId)
}
