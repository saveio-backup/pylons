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

type BaseConf struct {
	NetworkId           uint32   `json:"NetworkId"`
	NATProxyServerAddrs string   `json:"NATProxyServerAddrs"`
	ChainNodeURLs       []string `json:"ChainNodeURLs"`
	DnsAddr             string   `json:"DnsAddr"`
	DnsListenAddr       string   `json:"DnsListenAddr"`
	Init1ClientType     string   `json:"Init1ClientType"`
	Init1Addr           string   `json:"Init1Addr"`
	Init1ListenAddr     string   `json:"Init1ListenAddr"`
	Init2ClientType     string   `json:"Init2ClientType"`
	Init2Addr           string   `json:"Init2Addr"`
	Init2ListenAddr     string   `json:"Init2ListenAddr"`
}

type TestNetConf struct {
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

func GetTestConf() *TestNetConf {
	configDir := "./config.json"
	existed := FileExisted(configDir)
	if !existed {
		log.Info("[GetTestConf]: config.json is not existed")
		panic("[GetTestConf]: config.json is not existed")
	}
	cfg := &TestNetConf{}
	err := GetJsonObjectFromFile(configDir, cfg)
	if err != nil {
		log.Error("[GetJsonObjectFromFile] error: %s", err.Error())
		panic("[GetJsonObjectFromFile] error")
	}
	return cfg
}

func GetHostAddrCallBack(walletAddr common.Address) (string, error) {
	if walletAddr == Dns1Addr {
		return Parameters.BaseConfig.DnsListenAddr, nil
	} else if walletAddr == Initiator1Addr {
		return Parameters.BaseConfig.Init1ListenAddr, nil
	} else if walletAddr == Initiator2Addr {
		return Parameters.BaseConfig.Init2ListenAddr, nil
	} else {
		return "", fmt.Errorf("[GetHostAddrCallBack] error: not known")
	}
}

var Dns1Addr common.Address
var Initiator1Addr common.Address
var Initiator2Addr common.Address

var Parameters *TestNetConf

func init() {
	Parameters = GetTestConf()

	Dns1AddrTmp, _ := common.FromBase58(Parameters.BaseConfig.DnsAddr)
	Initiator1AddrTmp, _ := common.FromBase58(Parameters.BaseConfig.Init1Addr)
	Initiator2AddrTmp, _ := common.FromBase58(Parameters.BaseConfig.Init2Addr)

	Dns1Addr = common.Address(Dns1AddrTmp)
	Initiator1Addr = common.Address(Initiator1AddrTmp)
	Initiator2Addr = common.Address(Initiator2AddrTmp)

	log.Infof("Config Init NetworkId: %d", Parameters.BaseConfig.NetworkId)
}
