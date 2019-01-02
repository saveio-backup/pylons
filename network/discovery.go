package network

import (
	"fmt"

	"github.com/oniio/oniChannel/network/proxies"
	"github.com/oniio/oniChannel/typing"
)

type Discovery struct {
	nodeidToHostport map[typing.Address]string
}

func (self *Discovery) Reigister(nodeAddress typing.Address, host string, port int) {
	var result string

	result = fmt.Sprintf("%s:%d", host, port)
	self.nodeidToHostport[nodeAddress] = result

	return
}

func (self *Discovery) Get(nodeAddress typing.Address) string {
	return self.nodeidToHostport[nodeAddress]
}

type ContractDiscovery struct {
	NodeAddress    typing.Address
	DiscoveryProxy *proxies.Discovery
}

func (self *ContractDiscovery) Register(nodeAddress typing.Address, host string, port int) {

	endpoint := fmt.Sprintf("%s:%d", host, port)
	self.DiscoveryProxy.RegisterEndpoint(nodeAddress, endpoint)

	return
}

func (self *ContractDiscovery) Get(nodeAddress typing.Address) string {
	result := self.DiscoveryProxy.EndpointByAddress(nodeAddress)
	return result
}

func (self *ContractDiscovery) Version() string {
	result := self.DiscoveryProxy.Version()
	return result
}
