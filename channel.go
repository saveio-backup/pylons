package channel

import (
	"errors"
	"strconv"
	"strings"

	"github.com/oniio/oniChain/account"
	ch "github.com/oniio/oniChannel/channelservice"
	"github.com/oniio/oniChannel/network"
	"github.com/oniio/oniChannel/network/transport"
	"github.com/oniio/oniChannel/transfer"
	"github.com/oniio/oniChannel/typing"
)

type Channel struct {
	Config  *ChannelConfig
	Service *ch.ChannelService
}

type ChannelConfig struct {
	ChainNodeURL string

	// Transport config
	ListenAddress  string // ip + port
	Protocol       string // tcp or kcp
	MappingAddress string // ip + port, used to register in Endpoint contract when use address mapping

	DBPath string
}

func DefaultChannelConfig() *ChannelConfig {
	config := &ChannelConfig{
		ListenAddress: "127.0.0.1:3001",
		Protocol:      "tcp",
		DBPath:        ".",
	}

	return config
}

func NewChannel(config *ChannelConfig, account *account.Account) (*Channel, error) {
	blockChainService := network.NewBlockchainService(config.ChainNodeURL, account)
	if blockChainService == nil {
		return nil, errors.New("error createing BlockChainService")
	}

	transport, discovery := setupTransport(blockChainService, config)

	var startBlock typing.BlockHeight

	ipPort := config.ListenAddress
	if config.MappingAddress != "" {
		ipPort = config.MappingAddress
	}

	host, port, err := parseIPPort(ipPort)
	if err != nil {
		return nil, err
	}

	// construct the option map
	option := map[string]string{
		"database_path": config.DBPath,
		"host":          host,
		"port":          port,
	}

	service := ch.NewChannelService(
		blockChainService,
		startBlock,
		transport,
		blockChainService.GetAccount(),
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

func parseIPPort(address string) (string, string, error) {
	i := strings.Index(address, ":")
	if i < 0 {
		return "", "", errors.New("split ip address error")
	}
	ip := address[:i]

	port, err := strconv.Atoi(address[i+1:])
	if err != nil {
		return "", "", errors.New("parse port error")
	}

	if port <= 0 || port >= 65535 {
		return "", "", errors.New("port out of bound")
	}
	portStr := address[i+1:]

	return ip, portStr, nil
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
