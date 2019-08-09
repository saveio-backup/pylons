package test_config

import (
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/common"
)

const PROTOCOL = "tcp"

var Initiator1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3001",
	Protocol:      PROTOCOL,
}

var Initiator2 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3002",
	Protocol:      PROTOCOL,
}

var Media = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3003",
	Protocol:      PROTOCOL,
}

var Target1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3004",
	Protocol:      PROTOCOL,
}

var Target2 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://127.0.0.1:20336"},
	ListenAddress: "127.0.0.1:3005",
	Protocol:      PROTOCOL,
}

type BaseConf struct {
	NetworkId           uint32   `json:"NetworkId"`
	Protocol            string   `json:"Protocol"`
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

var Parameters = &TestNetConf{
	BaseConfig: &BaseConf{
		NetworkId:           1565267317,
		Protocol:            "tcp",
		NATProxyServerAddrs: "",
	},
}

var Initiator1Addr common.Address
var Initiator2Addr common.Address
var MediaAddr common.Address
var Target1Addr common.Address
var Target2Addr common.Address

func init() {
	Initiator1AddrTmp, _ := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	Initiator2AddrTmp, _ := common.FromBase58("AazwHGkaQk2S91qnxmTYpPuBD3GgGxJmYK")
	MediaAddrTmp, _ := common.FromBase58("AJtzEUDLzsRKbHC1Tfc1oNh8a1edpnVAUf")
	Target1AddrTmp, _ := common.FromBase58("AWpW2ukMkgkgRKtwWxC3viXEX8ijLio2Ng")
	Target2AddrTmp, _ := common.FromBase58("AYNNAk1rqUXtEhNWJhUciDZXrXCkZipiua")

	Initiator1Addr = common.Address(Initiator1AddrTmp)
	Initiator2Addr = common.Address(Initiator2AddrTmp)
	MediaAddr = common.Address(MediaAddrTmp)
	Target1Addr = common.Address(Target1AddrTmp)
	Target2Addr = common.Address(Target2AddrTmp)
}
