package channel

import (
	"errors"
	"fmt"
	"net"

	"github.com/oniio/oniChain/account"
	ch "github.com/oniio/oniChannel/channelservice"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

var Version = "0.1"

type Channel struct {
	Config  *ChannelConfig
	Service *ch.ChannelService
}

type ChannelConfig struct {
	ClientType   string
	ChainNodeURL string

	// Transport config
	ListenAddress  string // ip + port
	Protocol       string // tcp or kcp
	MappingAddress string // ip + port, used to register in Endpoint contract when use address mapping

	DBPath string
}

func DefaultChannelConfig() *ChannelConfig {
	config := &ChannelConfig{
		ClientType:    "rpc",
		ChainNodeURL:  "http://localhost:20336",
		ListenAddress: "127.0.0.1:3001",
		Protocol:      "tcp",
		DBPath:        ".",
	}

	return config
}

func NewChannel(config *ChannelConfig, account *account.Account) (*Channel, error) {
	blockChainService := network.NewBlockchainService(config.ClientType, config.ChainNodeURL, account)
	if blockChainService == nil {
		return nil, errors.New("error createing BlockChainService")
	}

	transport, discovery := setupTransport(blockChainService, config)

	var startBlock typing.BlockHeight

	ipPort := config.ListenAddress
	if config.MappingAddress != "" {
		ipPort = config.MappingAddress
	}

	h, p, err := net.SplitHostPort(ipPort)
	if err != nil {
		fmt.Errorf("parse ipPort err:%s\n", err)
		return nil, err
	}

	// construct the option map
	option := map[string]string{
		"database_path": config.DBPath,
		"host":          h,
		"port":          p,
	}

	service := ch.NewChannelService(
		blockChainService,
		startBlock,
		transport,
		new(ch.ChannelEventHandler),
		new(ch.MessageHandler),
		option,
		discovery)

	channel := &Channel{
		Config:  config,
		Service: service,
	}
	return channel, nil
}

func setupTransport(blockChainService *network.BlockchainService, config *ChannelConfig) (*transport.Transport, *network.ContractDiscovery) {
	discoveryProxy := blockChainService.Discovery()

	discovery := &network.ContractDiscovery{
		NodeAddress:    blockChainService.Address,
		DiscoveryProxy: discoveryProxy,
	}
	trans := transport.NewTransport(config.Protocol, discovery)
	trans.SetAddress(config.ListenAddress)
	trans.SetMappingAddress(config.MappingAddress)

	return trans, discovery
}

func (this *Channel) StartService() {
	this.Service.Start()
}

func (this *Channel) Stop() {
	this.Service.Stop()
}

func (this *Channel) RegisterReceiveNotification(notificaitonChannel chan *transfer.EventPaymentReceivedSuccess) {
	this.Service.ReceiveNotificationChannels[notificaitonChannel] = struct{}{}
}
