package test_config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/saveio/edge/common/config"
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/common"
	"github.com/saveio/themis/common/log"
	"io/ioutil"
	"os"
)

const PROTOCOL = "tcp"

type BaseConf struct {
	NetWorkId       uint32   `json:"NetWorkId"`
	ChainNodeURLs   []string `json:"ChainNodeURLs"`
	DnsAddr         string   `json:"DnsAddr"`
	DnsListenAddr   string   `json:"DnsListenAddr"`
	Init1Addr       string   `json:"Init1Addr"`
	Init1ListenAddr string   `json:"Init1ListenAddr"`
	Init2Addr       string   `json:"Init2Addr"`
	Init2ListenAddr string   `json:"Init2ListenAddr"`
}

var DefConf = &TestNetConf{
	TF: &BaseConf{
		NetWorkId:       1565267317,
		ChainNodeURLs:   []string{"http://221.179.156.57:10336", "http://221.179.156.57:11336", "http://221.179.156.57:12336"},
		DnsAddr:         "AXUhmdzcAJwaFW91q6UYuPGGJY3fimoTAj",
		DnsListenAddr:   "40.73.102.177:10338",
		Init1Addr:       "AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ",
		Init1ListenAddr: "0.0.0.0:3901",
		Init2Addr:       "AV8uHP2utZqu8WQbkyKVpCheLqsTa8PD4t",
		Init2ListenAddr: "0.0.0.0:3902",
	},
}

type TestNetConf struct {
	TF *BaseConf `json:"TestConf"`
}

// FileExisted checks whether filename exists in filesystem
func FileExisted(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func GetJsonObjectFromFile(filePath string, jsonObject interface{}) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
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
		return DefConf
	}
	cfg := &TestNetConf{}
	err := GetJsonObjectFromFile(configDir, cfg)
	if err != nil {
		log.Error("[GetJsonObjectFromFile] error: %s", err.Error())
		return DefConf
	}
	return cfg
}

var Initiator1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336", "http://221.179.156.57:11336", "http://221.179.156.57:12336"},
	ListenAddress: "0.0.0.0:3901",
	Protocol:      PROTOCOL,
}

var Initiator2 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336", "http://221.179.156.57:11336", "http://221.179.156.57:12336"},
	ListenAddress: "0.0.0.0:3902",
	Protocol:      PROTOCOL,
}

var DNS1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336", "http://221.179.156.57:11336", "http://221.179.156.57:12336"},
	ListenAddress: "40.73.102.177:10338",
	Protocol:      PROTOCOL,
}

var Dns1Addr common.Address
var Initiator1Addr common.Address
var Initiator2Addr common.Address
var Parameters *config.DspConfig

func init() {
	testConf := GetTestConf()
	Initiator1.ChainNodeURLs = testConf.TF.ChainNodeURLs
	Initiator2.ChainNodeURLs = testConf.TF.ChainNodeURLs
	DNS1.ChainNodeURLs = testConf.TF.ChainNodeURLs

	Dns1AddrTmp, _ := common.FromBase58(testConf.TF.DnsAddr)
	Initiator1AddrTmp, _ := common.FromBase58(testConf.TF.Init1Addr)
	Initiator2AddrTmp, _ := common.FromBase58(testConf.TF.Init2Addr)

	Dns1Addr = common.Address(Dns1AddrTmp)
	Initiator1Addr = common.Address(Initiator1AddrTmp)
	Initiator2Addr = common.Address(Initiator2AddrTmp)
	Parameters = TestConfig()
	Parameters.BaseConfig.NetworkId = testConf.TF.NetWorkId
	log.Infof("Config Init NetworkId: %d", testConf.TF.NetWorkId)
}

func TestConfig() *config.DspConfig {
	return &config.DspConfig{
		BaseConfig: config.BaseConfig{
			NetworkId:            1564141146,
			BaseDir:              ".",
			LogPath:              "./Log",
			PortBase:             10000,
			LogLevel:             0,
			LocalRpcPortOffset:   205,
			EnableLocalRpc:       false,
			JsonRpcPortOffset:    204,
			EnableJsonRpc:        true,
			HttpRestPortOffset:   203,
			RestEnable:           true,
			ChannelPortOffset:    202,
			ChannelProtocol:      "udp",
			ChannelClientType:    "rpc",
			ChannelRevealTimeout: "250",
			DBPath:               "./DB",
			ChainRestAddrs:       []string{"http://127.0.0.1:20334"},
			ChainRpcAddrs:        []string{"http://221.179.156.57:10336"},
			NATProxyServerAddrs:  "",
			DspProtocol:          "udp",
			DspPortOffset:        201,
			AutoSetupDNSEnable:   true,
			DnsNodeMaxNum:        100,
			SeedInterval:         3600,
			DnsChannelDeposit:    1000000000,
			WalletPwd:            "pwd",
			WalletDir:            "./wallet.dat",
		},
		FsConfig: config.FsConfig{
			FsRepoRoot:   "./FS",
			FsFileRoot:   "./Downloads",
			FsType:       0,
			EnableBackup: true,
		},
	}
}
