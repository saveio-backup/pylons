package test_config

import (
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

func init() {
	Dns1AddrTmp, _ := common.FromBase58("AXUhmdzcAJwaFW91q6UYuPGGJY3fimoTAj")
	Initiator1AddrTmp, _ := common.FromBase58("AMkN2sRQyT3qHZQqwEycHCX2ezdZNpXNdJ")
	Initiator2AddrTmp, _ := common.FromBase58("AV8uHP2utZqu8WQbkyKVpCheLqsTa8PD4t")

	Dns1Addr = common.Address(Dns1AddrTmp)
	Initiator1Addr = common.Address(Initiator1AddrTmp)
	Initiator2Addr = common.Address(Initiator2AddrTmp)
}
