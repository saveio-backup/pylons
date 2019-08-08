package test_config

import (
	"github.com/saveio/edge/common/config"
	ch "github.com/saveio/pylons"
	"github.com/saveio/pylons/common"
)

const PROTOCOL = "tcp"

var Initiator1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336"},
	ListenAddress: "40.73.96.40:3901",
	Protocol:      PROTOCOL,
}

var Initiator2 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336"},
	ListenAddress: "40.73.96.40:3902",
	Protocol:      PROTOCOL,
}

var DNS1 = &ch.ChannelConfig{
	ClientType:    "rpc",
	ChainNodeURLs: []string{"http://221.179.156.57:10336"},
	ListenAddress: "40.73.102.177:10338",
	Protocol:      PROTOCOL,
}

var Dns1Addr common.Address
var Initiator1Addr common.Address
var Initiator2Addr common.Address
var Parameters *config.DspConfig

func init() {
	Dns1AddrTmp, _ := common.FromBase58("AXUhmdzcAJwaFW91q6UYuPGGJY3fimoTAj")
	Initiator1AddrTmp, _ := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	Initiator2AddrTmp, _ := common.FromBase58("AV8uHP2utZqu8WQbkyKVpCheLqsTa8PD4t")

	Dns1Addr = common.Address(Dns1AddrTmp)
	Initiator1Addr = common.Address(Initiator1AddrTmp)
	Initiator2Addr = common.Address(Initiator2AddrTmp)
	Parameters = TestConfig()
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
